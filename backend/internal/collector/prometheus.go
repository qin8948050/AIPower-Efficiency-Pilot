package collector

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/qxw/aipower-efficiency-pilot/internal/storage"
)

type PrometheusCollector struct {
	api   v1.API
	redis *storage.RedisClient
}

// NewPrometheusCollector 初始化 Prometheus 客户端
func NewPrometheusCollector(address string, redisCli *storage.RedisClient) (*PrometheusCollector, error) {
	client, err := api.NewClient(api.Config{
		Address: address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus client: %v", err)
	}

	v1api := v1.NewAPI(client)
	return &PrometheusCollector{
		api:   v1api,
		redis: redisCli,
	}, nil
}

// Start 启动定时轮询，抓取 GPU 指标并存入 Redis
func (p *PrometheusCollector) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Starting Prometheus collector with %s interval", interval.String())
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.fetchAndCacheMetrics(ctx)
		}
	}
}

func (p *PrometheusCollector) fetchAndCacheMetrics(ctx context.Context) {
	// 查询 GPU 利用率，假设使用的指标名为 DCGM_FI_DEV_GPU_UTIL
	query := `DCGM_FI_DEV_GPU_UTIL`

	result, warnings, err := p.api.Query(ctx, query, time.Now())
	if err != nil {
		log.Printf("Failed to query Prometheus: %v", err)
		return
	}
	if len(warnings) > 0 {
		log.Printf("Prometheus queries warnings: %v", warnings)
	}

	// 这里可以后续结合 PodTrace 中保存的具体 Namespace/Pod 过滤数据
	// 或者直接在 Prometheus 侧查出带有 pod 标签的指标 (取决于 DCGM Exporter 的配置)
	switch value := result.(type) {
	case model.Vector:
		for _, s := range value {
			pod := string(s.Metric["pod"])
			namespace := string(s.Metric["namespace"])
			if pod == "" || namespace == "" {
				continue
			}
			util := float64(s.Value)

			// 可以将获取到的利用率更新至 Redis，为看板提供实时展现
			key := fmt.Sprintf("pod_metrics:%s:%s", namespace, pod)
			// TODO: 可以存储为结构体并聚合
			p.redis.SaveNodePoolID(key, fmt.Sprintf("%.2f%%", util)) // 这里权宜使用同一方法存字符串
		}
	}
}

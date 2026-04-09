package collector

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/qxw/aipower-efficiency-pilot/internal/storage"
	"github.com/qxw/aipower-efficiency-pilot/internal/types"
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

// fetchAndCacheMetrics 抓取 GPU 指标并存入 Redis
// 使用 avg_over_time[5m] 聚合过去 5 分钟数据，并通过动态延迟检测确保数据完整
func (p *PrometheusCollector) fetchAndCacheMetrics(ctx context.Context) {
	// 1. 动态检测 Prometheus 数据延迟
	queryTime := p.detectSafeQueryTime(ctx)

	// 2. 使用 avg_over_time[5m] 查询过去 5 分钟的平均值
	queries := map[string]string{
		"gpu_util_avg": `avg by (pod, namespace) (DCGM_FI_DEV_GPU_UTIL)`,
		"gpu_util_max": `max by (pod, namespace) (DCGM_FI_DEV_GPU_UTIL)`,
		"mem_used":     `sum by (pod, namespace) (DCGM_FI_DEV_FB_USED * 1024 * 1024)`,
		"mem_total":    `sum by (pod, namespace) (DCGM_FI_DEV_FB_FREE + DCGM_FI_DEV_FB_USED) * 1024 * 1024`,
		"power_usage":  `avg by (pod, namespace) (DCGM_FI_DEV_POWER_USAGE)`,
	}

	podMetricsMap := make(map[string]*types.GPUMetrics)

	for name, query := range queries {
		// 使用 avg_over_time 聚合 + 动态安全时间
		rangeQuery := fmt.Sprintf("avg_over_time(%s[5m])", query)
		result, _, err := p.api.Query(ctx, rangeQuery, queryTime)
		if err != nil {
			log.Printf("Failed to query Prometheus for %s: %v", name, err)
			continue
		}

		if vec, ok := result.(model.Vector); ok {
			for _, s := range vec {
				pod := string(s.Metric["pod"])
				ns := string(s.Metric["namespace"])
				if pod == "" || ns == "" {
					continue
				}
				key := fmt.Sprintf("%s/%s", ns, pod)
				if _, exists := podMetricsMap[key]; !exists {
					podMetricsMap[key] = &types.GPUMetrics{LastUpdate: time.Now()}
				}

				val := float64(s.Value)
				switch name {
				case "gpu_util_avg":
					podMetricsMap[key].GPUUtilAvg = val
				case "gpu_util_max":
					podMetricsMap[key].GPUUtilMax = val
				case "mem_used":
					podMetricsMap[key].MemUsedBytes = uint64(val)
				case "mem_total":
					podMetricsMap[key].MemTotalBytes = uint64(val)
				case "power_usage":
					podMetricsMap[key].PowerUsageW = val
				}
			}
		}
	}

	// 3. 同步至 Redis
	for key, metrics := range podMetricsMap {
		parts := strings.Split(key, "/")
		ns, pod := parts[0], parts[1]
		if err := p.redis.UpdatePodMetrics(ns, pod, metrics); err != nil {
			// 如果 PodTrace 还未同步到 Redis，该错误可忽略
			continue
		}
		log.Printf("Updated metrics snapshot for %s: UtilAvg=%.2f%%, MemUsed=%dMB",
			key, metrics.GPUUtilAvg, metrics.MemUsedBytes/1024/1024)
	}
}

// detectSafeQueryTime 动态检测 Prometheus 数据延迟，返回安全的查询时间点
// 通过查询实际指标的时间戳来判断延迟，避免数据未就绪
func (p *PrometheusCollector) detectSafeQueryTime(ctx context.Context) time.Time {
	// 使用一个代表性指标查询最新样本时间戳
	query := `max by (pod, namespace) (DCGM_FI_DEV_GPU_UTIL)`
	result, _, err := p.api.Query(ctx, query, time.Now())
	if err != nil {
		log.Printf("Warning: Failed to detect Prometheus delay, using fallback: %v", err)
		return time.Now().Add(-2 * time.Minute) // 保守 fallback
	}

	if vec, ok := result.(model.Vector); ok && len(vec) > 0 {
		// 取最新样本的实际时间戳
		latestSampleTime := vec[0].Timestamp.Time()
		delay := time.Since(latestSampleTime)

		if delay < 30*time.Second {
			// 无延迟或延迟极小，直接使用当前时间
			return time.Now()
		}

		// 有延迟，返回 (当前时间 - 延迟 - 30s 安全边际)
		safeTime := time.Now().Add(-delay - 30*time.Second)
		log.Printf("Detected Prometheus delay: %v, using safe query time: %v", delay, safeTime)
		return safeTime
	}

	// 无数据返回，保守 fallback
	return time.Now().Add(-2 * time.Minute)
}

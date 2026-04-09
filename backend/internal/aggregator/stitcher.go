package aggregator

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/qxw/aipower-efficiency-pilot/internal/storage"
)

// MetricsSample 单个指标样本
type MetricsSample struct {
	AvgUtil   float64
	MaxUtil   float64
	MaxMemMiB uint64
	AvgPowerW float64
}

// StitchFromPrometheus 向 Prometheus 发起聚合查询，计算时间窗口内的指标
// 使用 PromQL 服务端聚合函数，无需传输原始数据
func StitchFromPrometheus(api v1.API, gpuUUID string, start, end time.Time) (*MetricsSample, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	duration := end.Sub(start)
	if duration <= 0 {
		return nil, fmt.Errorf("invalid time range: start=%v, end=%v", start, end)
	}

	sample := &MetricsSample{}

	// GPU 利用率：平均值和峰值
	avgUtilQuery := fmt.Sprintf(`avg_over_time(DCGM_FI_DEV_GPU_UTIL{uuid="%s"}[%s])`, gpuUUID, duration)
	if val, err := queryFloat(api, ctx, avgUtilQuery, end); err == nil {
		sample.AvgUtil = val
	} else {
		log.Printf("[stitcher] avg_util query failed for %s: %v", gpuUUID, err)
	}

	maxUtilQuery := fmt.Sprintf(`max_over_time(DCGM_FI_DEV_GPU_UTIL{uuid="%s"}[%s])`, gpuUUID, duration)
	if val, err := queryFloat(api, ctx, maxUtilQuery, end); err == nil {
		sample.MaxUtil = val
	}

	// 显存使用峰值
	memQuery := fmt.Sprintf(`max_over_time(DCGM_FI_DEV_FB_USED{uuid="%s"}[%s])`, gpuUUID, duration)
	if val, err := queryFloat(api, ctx, memQuery, end); err == nil {
		sample.MaxMemMiB = uint64(val)
	}

	// 功耗平均值
	powerQuery := fmt.Sprintf(`avg_over_time(DCGM_FI_DEV_POWER_USAGE{uuid="%s"}[%s])`, gpuUUID, duration)
	if val, err := queryFloat(api, ctx, powerQuery, end); err == nil {
		sample.AvgPowerW = val
	}

	return sample, nil
}

// queryFloat 向 Prometheus 执行即时查询，返回 float64 值
func queryFloat(api v1.API, ctx context.Context, query string, ts time.Time) (float64, error) {
	result, _, err := api.Query(ctx, query, ts)
	if err != nil {
		return 0, err
	}
	if vec, ok := result.(model.Vector); ok && len(vec) > 0 {
		return float64(vec[0].Value), nil
	}
	return 0, fmt.Errorf("no data returned for query: %s", query)
}

// P95 计算一组 float64 的 P95 值
func P95(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	idx := int(float64(len(sorted)) * 0.95)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// RunStitcher 扫描待缝合记录并完成缝合（使用模拟数据，不依赖真实 Prometheus）
func RunStitcher(db *storage.MySQLClient, limit int) error {
	traces, err := db.GetPendingMetricsTraces(limit)
	if err != nil {
		return fmt.Errorf("scan pending traces: %w", err)
	}
	if len(traces) == 0 {
		log.Println("[stitcher] 无待缝合记录")
		return nil
	}

	log.Printf("[stitcher] 发现 %d 条待缝合记录", len(traces))
	for _, trace := range traces {

		endTime := *trace.EndTime

		// 获取定价
		pricing := GetPoolPricing(db, trace.PoolID)

		// TODO: 真实环境替换为 StitchFromPrometheus(api, gpuUUID, trace.StartTime, endTime)
		// 此处使用模拟数据
		sample := simulateSample()
		// 使用 Pod 创建时记录的 SlicingWeight 直接计算成本
		cost := CalculateCost(trace.StartTime, endTime, pricing, trace.SlicingWeight)

		if err := db.UpdateLifeTraceMetrics(trace.ID, sample.AvgUtil, sample.MaxUtil, sample.MaxMemMiB, sample.AvgPowerW, cost); err != nil {
			log.Printf("[stitcher] 缝合失败 id=%d: %v", trace.ID, err)
			continue
		}
		log.Printf("[stitcher] ✅ id=%d %s/%s avgUtil=%.2f%% cost=¥%.4f",
			trace.ID, trace.Namespace, trace.PodName, sample.AvgUtil, cost)
	}
	return nil
}

func simulateSample() *MetricsSample {
	var sum, max float64
	for i := 0; i < 12; i++ {
		v := 20 + float64(time.Now().UnixNano()%7000)/100.0
		sum += v
		if v > max {
			max = v
		}
	}
	return &MetricsSample{
		AvgUtil:   sum / 12,
		MaxUtil:   max,
		MaxMemMiB: 32 * 1024, // 32 GiB
		AvgPowerW: 230.0,
	}
}

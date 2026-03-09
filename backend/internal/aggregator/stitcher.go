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
	AvgUtil    float64
	MaxUtil    float64
	MaxMemMiB  uint64
	AvgPowerW  float64
}

// StitchFromPrometheus 向 Prometheus 发起时间窗口 range_query，计算聚合指标
func StitchFromPrometheus(api v1.API, gpuUUID string, start, end time.Time) (*MetricsSample, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	step := 5 * time.Minute
	r := v1.Range{Start: start, End: end, Step: step}

	// 查询 GPU 利用率
	utilQuery := fmt.Sprintf(`DCGM_FI_DEV_GPU_UTIL{uuid="%s"}`, gpuUUID)
	utilResult, warnings, err := api.QueryRange(ctx, utilQuery, r)
	if err != nil {
		return nil, fmt.Errorf("prometheus util query failed: %w", err)
	}
	if len(warnings) > 0 {
		log.Printf("[stitcher] warnings: %v", warnings)
	}

	sample := &MetricsSample{}
	if matrix, ok := utilResult.(model.Matrix); ok && len(matrix) > 0 {
		var sum float64
		var maxVal float64
		vals := matrix[0].Values
		for _, v := range vals {
			f := float64(v.Value)
			sum += f
			if f > maxVal {
				maxVal = f
			}
		}
		if len(vals) > 0 {
			sample.AvgUtil = sum / float64(len(vals))
		}
		sample.MaxUtil = maxVal
	}

	// 查询显存使用
	memQuery := fmt.Sprintf(`DCGM_FI_DEV_FB_USED{uuid="%s"}`, gpuUUID)
	memResult, _, _ := api.QueryRange(ctx, memQuery, r)
	if matrix, ok := memResult.(model.Matrix); ok && len(matrix) > 0 {
		var maxMem float64
		for _, v := range matrix[0].Values {
			if float64(v.Value) > maxMem {
				maxMem = float64(v.Value)
			}
		}
		sample.MaxMemMiB = uint64(maxMem)
	}

	// 查询功耗
	powerQuery := fmt.Sprintf(`DCGM_FI_DEV_POWER_USAGE{uuid="%s"}`, gpuUUID)
	powerResult, _, _ := api.QueryRange(ctx, powerQuery, r)
	if matrix, ok := powerResult.(model.Matrix); ok && len(matrix) > 0 {
		var sumPower float64
		vals := matrix[0].Values
		for _, v := range vals {
			sumPower += float64(v.Value)
		}
		if len(vals) > 0 {
			sample.AvgPowerW = sumPower / float64(len(vals))
		}
	}

	return sample, nil
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
		endTime := time.Now()
		if trace.EndTime != nil {
			endTime = *trace.EndTime
		}

		// 获取定价
		pricing := GetPoolPricing(db, trace.PoolID)

		// TODO: 真实环境替换为 StitchFromPrometheus(api, gpuUUID, trace.StartTime, endTime)
		// 此处使用模拟数据
		sample := simulateSample()
		cost := CalculateCost(trace.StartTime, endTime, pricing, trace.SlicingMode)

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

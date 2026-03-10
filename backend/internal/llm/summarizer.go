package llm

import (
	"time"

	"github.com/qxw/aipower-efficiency-pilot/internal/storage"
)

// LowUtilThreshold 低利用率阈值 (30%)
const LowUtilThreshold = 30.0

// HighJitterThreshold 高抖动阈值 (15%)
const HighJitterThreshold = 15.0

// Summarizer 数据脱水降维器
type Summarizer struct {
	mysqlCli *storage.MySQLClient
}

// NewSummarizer 创建新的脱水器
func NewSummarizer(mysqlCli *storage.MySQLClient) *Summarizer {
	return &Summarizer{mysqlCli: mysqlCli}
}

// GeneratePoolSummary 为指定资源池生成统计特征摘要
func (s *Summarizer) GeneratePoolSummary(poolID string, days int) (*InsightSummary, error) {
	// 计算日期范围
	endDate := time.Now()
	startDate := endDate.Add(-time.Duration(days) * 24 * time.Hour)

	// 查询日级快照数据
	snapshots, err := s.mysqlCli.GetDailySnapshotsByPool(poolID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	if len(snapshots) == 0 {
		return &InsightSummary{
			PoolID:    poolID,
			TimeRange: "7d",
		}, nil
	}

	// 聚合统计特征
	var totalCost, totalUtil, maxUtil float64
	var podCount int

	for _, snap := range snapshots {
		totalCost += snap.TotalCost
		totalUtil += snap.AvgUtilP95
		if snap.AvgUtilP95 > maxUtil {
			maxUtil = snap.AvgUtilP95
		}
		podCount += snap.PodSessionCount
	}

	avgUtil := totalUtil / float64(len(snapshots))

	// 计算浪费成本（利用率低于阈值的部分）
	wasteCost := calculateWasteCost(totalCost, avgUtil, LowUtilThreshold)

	// 获取低利用率 Pod 列表
	lowUtilPods, err := s.getLowUtilPods(poolID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// 获取高抖动 Pod 列表（这里简化处理，实际需要更复杂的计算）
	highJitterPods, err := s.getHighJitterPods(poolID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// 获取资源池元数据
	pool, _ := s.mysqlCli.GetResourcePool(poolID)

	return &InsightSummary{
		PoolID:           poolID,
		TimeRange:        "7d",
		AvgUtilization:   avgUtil,
		MaxUtilization:   maxUtil,
		CostTotal:        totalCost,
		WasteCost:        wasteCost,
		PodCount:         podCount,
		LowUtilPods:      lowUtilPods,
		HighJitterPods:   highJitterPods,
		SlicingMode:      pool.SlicingMode,
		HardwareFeatures: pool.HardwareFeatures,
	}, nil
}

// generateAllPoolsSummary 生成所有资源池的摘要
func (s *Summarizer) GenerateAllPoolsSummary(days int) ([]InsightSummary, error) {
	pools, err := s.mysqlCli.GetAllResourcePools()
	if err != nil {
		return nil, err
	}

	var summaries []InsightSummary
	for _, pool := range pools {
		summary, err := s.GeneratePoolSummary(pool.ID, days)
		if err != nil {
			continue
		}
		summaries = append(summaries, *summary)
	}

	return summaries, nil
}

// getLowUtilPods 获取低利用率 Pod 列表
func (s *Summarizer) getLowUtilPods(poolID string, startDate, endDate time.Time) ([]LowUtilPod, error) {
	traces, err := s.mysqlCli.GetPodTracesByPoolAndUtil(poolID, startDate, endDate, LowUtilThreshold)
	if err != nil {
		return nil, err
	}

	var pods []LowUtilPod
	for _, trace := range traces {
		wasteCost := trace.CostAmount * (1 - trace.GPUUtilAvg/100)
		pods = append(pods, LowUtilPod{
			PodName:      trace.PodName,
			Namespace:    trace.Namespace,
			AvgUtil:      trace.GPUUtilAvg,
			EstWasteCost: wasteCost,
		})
	}

	return pods, nil
}

// getHighJitterPods 获取高抖动 Pod 列表
// 注意：这里简化处理，实际应该基于时间序列指标计算抖动
func (s *Summarizer) getHighJitterPods(poolID string, startDate, endDate time.Time) ([]JitterPod, error) {
	traces, err := s.mysqlCli.GetPodTracesByPool(poolID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// 简化：利用 max - avg 作为抖动指标
	var pods []JitterPod
	for _, trace := range traces {
		jitter := trace.GPUUtilMax - trace.GPUUtilAvg
		if jitter > HighJitterThreshold {
			pods = append(pods, JitterPod{
				PodName:   trace.PodName,
				Namespace: trace.Namespace,
				JitterPct: jitter,
			})
		}
	}

	return pods, nil
}

// calculateWasteCost 计算浪费成本
func calculateWasteCost(totalCost, avgUtil, threshold float64) float64 {
	if avgUtil >= threshold {
		return 0
	}
	// 浪费比例 = (阈值 - 实际利用率) / 阈值
	wasteRatio := (threshold - avgUtil) / threshold
	return totalCost * wasteRatio
}

// ToMarkdown 将摘要转换为 Markdown 格式（供 LLM 使用）
func (s *InsightSummary) ToMarkdown() string {
	md := "## 资源池效能画像\n\n"
	md += "| 指标 | 值 |\n|------|-----|\n"
	md += "| 资源池 ID | " + s.PoolID + " |\n"
	md += "| 时间范围 | " + s.TimeRange + " |\n"
	md += "| 平均利用率 | " + formatFloat(s.AvgUtilization) + "% |\n"
	md += "| 峰值利用率 | " + formatFloat(s.MaxUtilization) + "% |\n"
	md += "| 总成本 | $" + formatFloat(s.CostTotal) + " |\n"
	md += "| 浪费成本 | $" + formatFloat(s.WasteCost) + " |\n"
	md += "| Pod 数量 | " + itoa(s.PodCount) + " |\n"
	md += "| 切片模式 | " + s.SlicingMode + " |\n"

	if len(s.LowUtilPods) > 0 {
		md += "\n### 低利用率 Pod (< 30%)\n\n"
		md += "| Pod Name | Namespace | 平均利用率 | 预计浪费成本 |\n"
		md += "|----------|-----------|-----------|-------------|\n"
		for _, p := range s.LowUtilPods {
			md += "| " + p.PodName + " | " + p.Namespace + " | " + formatFloat(p.AvgUtil) + "% | $" + formatFloat(p.EstWasteCost) + " |\n"
		}
	}

	if len(s.HighJitterPods) > 0 {
		md += "\n### 高抖动 Pod (Jitter > 15%)\n\n"
		md += "| Pod Name | Namespace | 抖动率 |\n"
		md += "|----------|-----------|--------|\n"
		for _, p := range s.HighJitterPods {
			md += "| " + p.PodName + " | " + p.Namespace + " | " + formatFloat(p.JitterPct) + "% |\n"
		}
	}

	return md
}

func formatFloat(f float64) string {
	return itoa(int(f))
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var s string
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	return s
}

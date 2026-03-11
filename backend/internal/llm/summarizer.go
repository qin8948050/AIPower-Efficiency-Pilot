package llm

import (
	"fmt"
	"strings"
	"time"

	"github.com/qxw/aipower-efficiency-pilot/internal/storage"
)

// LifeTrace 是 storage.LifeTrace 的别名
type LifeTrace = storage.LifeTrace

// LowUtilThreshold 低利用率阈值 (30%)
const LowUtilThreshold = 30.0

// HighJitterThreshold 高抖动阈值 (15%)
const HighJitterThreshold = 15.0

// FeatureUtilThreshold 特性利用率阈值 (10%)
const FeatureUtilThreshold = 10.0

// ConsecutiveDaysThreshold 连续低利用率天数阈值
const ConsecutiveDaysThreshold = 3

// TaskScene 任务场景类型
type TaskScene string

const (
	ScenePreTraining   TaskScene = "pre_training"   // 大模型预训练
	SceneFineTuning    TaskScene = "fine_tuning"    // 模型微调
	SceneCoreInference TaskScene = "core_inference" // 核心推理
	SceneSmallInference TaskScene = "small_inference" // 小模型推理
	SceneDevTest       TaskScene = "dev_test"       // 研发测试
	SceneUnknown       TaskScene = "unknown"
)

// TaskProfile 任务画像（Pod 特征）
type TaskProfile struct {
	PodName          string
	Namespace        string
	PoolID           string
	SlicingMode      string
	AvgUtil          float64
	MaxUtil          float64
	Jitter           float64
	Cost             float64
	DurationHours    float64   // 运行时长（小时）
	Scene            TaskScene
	RequiredFeatures string    // Pod 声明需要的特性
}

// PoolProfile 资源池画像
type PoolProfile struct {
	ID               string
	PoolID          string
	Scene           string // 适用场景
	HardwareFeatures string
	SlicingMode     string
	Priority        string
}

// MismatchResult 任务场景与资源池不匹配结果
type MismatchResult struct {
	Task          TaskProfile
	CurrentPool   PoolProfile
	SuggestedPool string
	Reason        string
	ReportType    string // "downgrade", "upgrade", "isolation", "feature_mismatch"
	EstSavings    float64
}

// Summarizer 数据脱水降维器
type Summarizer struct {
	mysqlCli *storage.MySQLClient
}

// NewSummarizer 创建新的脱水器
func NewSummarizer(mysqlCli *storage.MySQLClient) *Summarizer {
	return &Summarizer{mysqlCli: mysqlCli}
}

// GeneratePoolSummary 为指定资源池生成统计特征摘要（旧接口，保留兼容）
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

	// 计算浪费成本
	wasteCost := calculateWasteCost(totalCost, avgUtil, LowUtilThreshold)

	// 获取资源池元数据
	pool, _ := s.mysqlCli.GetResourcePool(poolID)

	// 获取各类 Pod 列表
	lowUtilPods, _ := s.getLowUtilPods(poolID, startDate, endDate)
	highJitterPods, _ := s.getHighJitterPods(poolID, startDate, endDate)
	featureMismatchPods, _ := s.getFeatureMismatchPods(poolID, startDate, endDate)

	// 判定分析类型
	isDowngradeTarget := isFullOrMIGPool(pool.SlicingMode) && avgUtil < LowUtilThreshold
	isIsolationTarget := isMPSOrTSPool(pool.SlicingMode) && len(highJitterPods) > 0
	// 特性纠偏: 有 Pod 声明了需要的特性（让 LLM 判断是否匹配）
	isFeatureMismatch := len(featureMismatchPods) > 0

	return &InsightSummary{
		PoolID:              poolID,
		TimeRange:           "7d",
		AvgUtilization:      avgUtil,
		MaxUtilization:      maxUtil,
		CostTotal:           totalCost,
		WasteCost:           wasteCost,
		PodCount:            podCount,
		LowUtilPods:         lowUtilPods,
		HighJitterPods:      highJitterPods,
		FeatureMismatchPods: featureMismatchPods,
		SlicingMode:         pool.SlicingMode,
		HardwareFeatures:    pool.HardwareFeatures,
		IsDowngradeTarget:  isDowngradeTarget,
		IsIsolationTarget:  isIsolationTarget,
		IsFeatureMismatch:  isFeatureMismatch,
	}, nil
}

// AnalyzeTaskPoolMismatch 核心算法：分析任务场景与资源池的匹配关系
// 这是 Phase 3 的核心逻辑
func (s *Summarizer) AnalyzeTaskPoolMismatch(days int) ([]MismatchResult, error) {
	endDate := time.Now()
	startDate := endDate.Add(-time.Duration(days) * 24 * time.Hour)

	// 1. 获取所有资源池信息
	pools, err := s.mysqlCli.GetAllResourcePools()
	if err != nil {
		return nil, err
	}
	poolMap := make(map[string]PoolProfile)
	for _, p := range pools {
		poolMap[p.ID] = PoolProfile{
			ID:               p.ID,
			PoolID:          p.ID,
			Scene:           p.Scene,
			HardwareFeatures: p.HardwareFeatures,
			SlicingMode:     p.SlicingMode,
			Priority:        p.Priority,
		}
	}

	// 2. 获取所有 Pod 的运行记录
	var allTraces []LifeTrace
	for _, pool := range pools {
		traces, err := s.mysqlCli.GetPodTracesByPool(pool.ID, startDate, endDate)
		if err != nil {
			continue
		}
		allTraces = append(allTraces, traces...)
	}

	// 3. 遍历每个 Pod，识别任务场景并判断是否匹配
	var results []MismatchResult
	for _, trace := range allTraces {
		pool, ok := poolMap[trace.PoolID]
		if !ok {
			continue
		}

		// 识别任务场景（保留用于后续扩展）
		_ = s.identifyTaskScene(&trace)

		// 判断是否高效
		mismatch := s.checkMismatch(&trace, &pool)
		if mismatch != nil {
			results = append(results, *mismatch)
		}
	}

	return results, nil
}

// identifyTaskScene 识别任务场景
// 根据 Pod 的 namespace 推断任务类型
func (s *Summarizer) identifyTaskScene(trace *LifeTrace) TaskScene {
	ns := strings.ToLower(trace.Namespace)

	// 根据 namespace 关键字推断
	if strings.Contains(ns, "inference") || strings.Contains(ns, "infer") {
		return SceneCoreInference
	}

	if strings.Contains(ns, "train") {
		return SceneFineTuning
	}

	if strings.Contains(ns, "dev") || strings.Contains(ns, "test") || strings.Contains(ns, "ci") {
		return SceneDevTest
	}

	// 无法识别，返回未知
	return SceneUnknown
}

// checkMismatch 检查任务在当前资源池是否高效
// 核心原则：高效、不浪费
// 1. 降级: Full/MIG 池 + 利用率 < 30% → 降级到 MPS/TS
// 2. 隔离: MPS/TS 池 + Jitter > 15% → 升级到 MIG
// 3. 特性纠偏: 高端特性池 + 利用率 < 10% → 回退到通用池
func (s *Summarizer) checkMismatch(trace *LifeTrace, pool *PoolProfile) *MismatchResult {
	var suggestedPool string
	var reason string
	var reportType string
	var estSavings float64

	// 计算抖动
	jitter := trace.GPUUtilMax - trace.GPUUtilAvg

	// 算法1: 降级迁移 - Full/MIG 池 + 利用率 < 30%
	if isFullOrMIGPool(pool.SlicingMode) && trace.GPUUtilAvg < LowUtilThreshold {
		suggestedPool = findDowngradePool(pool.PoolID)
		reportType = "downgrade"
		reason = fmt.Sprintf("任务在 %s 池利用率仅 %.1f%%，资源浪费严重，建议迁移至 %s",
			pool.SlicingMode, trace.GPUUtilAvg, suggestedPool)
		estSavings = trace.CostAmount * (1 - trace.GPUUtilAvg/100)
	}

	// 算法2: 稳定性升级 - MPS/TS 池 + Jitter > 15%
	if isMPSOrTSPool(pool.SlicingMode) && jitter > HighJitterThreshold {
		suggestedPool = findMIGPool()
		reportType = "isolation"
		reason = fmt.Sprintf("任务在 %s 池抖动达 %.1f%%(>15%%)，受邻居干扰严重，建议迁移至 MIG 硬隔离池",
			pool.SlicingMode, jitter)
		estSavings = trace.CostAmount * 0.3
	}

	// 算法3: 特性纠偏 - Pod 声明需要某特性，但资源池不提供
	// 需要在 Analyzer 中调用 LLM 判断，暂时保留简单逻辑
	// 实际实现会在 callLLM 时传入 Pod 声明的特性和资源池特性进行语义匹配

	// 如果有不匹配结果
	if suggestedPool != "" {
		return &MismatchResult{
			Task: TaskProfile{
				PodName:       trace.PodName,
				Namespace:     trace.Namespace,
				PoolID:        trace.PoolID,
				SlicingMode:   trace.SlicingMode,
				AvgUtil:       trace.GPUUtilAvg,
				MaxUtil:       trace.GPUUtilMax,
				Jitter:        jitter,
				Cost:          trace.CostAmount,
				DurationHours: 0,
				Scene:         SceneUnknown,
			},
			CurrentPool:   *pool,
			SuggestedPool: suggestedPool,
			Reason:        reason,
			ReportType:    reportType,
			EstSavings:    estSavings,
		}
	}

	return nil
}

// findPoolByScene 根据场景名称查找资源池
func findPoolByScene(sceneName string) string {
	poolMap := map[string]string{
		"Dev-T4-TS-Pool":      "Dev-T4-TS-Pool",
		"Infer-L4-MPS-Pool":   "Infer-L4-MPS-Pool",
		"Infer-A100-MIG-Pool": "Infer-A100-MIG-Pool",
		"Train-A100-Full-Pool": "Train-A100-Full-Pool",
		"Train-H800-Full-Pool": "Train-H800-Full-Pool",
	}
	return poolMap[sceneName]
}

// findCheaperPool 查找更便宜的资源池
func findCheaperPool(currentPool string) string {
	// 按价格从高到低排序
	pools := []string{
		"Train-H800-Full-Pool",
		"Train-A100-Full-Pool",
		"Infer-A100-MIG-Pool",
		"Infer-L4-MPS-Pool",
		"Dev-T4-TS-Pool",
	}

	for i, p := range pools {
		if p == currentPool && i < len(pools)-1 {
			return pools[i+1]
		}
	}

	// 默认返回 Dev 池
	return "Dev-T4-TS-Pool"
}

// findDowngradePool Full/MIG 池降级到 MPS/TS 池
func findDowngradePool(currentPool string) string {
	if strings.Contains(currentPool, "Train-H800") {
		return "Infer-L4-MPS-Pool"
	}
	if strings.Contains(currentPool, "Train-A100") {
		return "Infer-L4-MPS-Pool"
	}
	if strings.Contains(currentPool, "Infer-A100-MIG") {
		return "Dev-T4-TS-Pool"
	}
	return "Dev-T4-TS-Pool"
}

// findMIGPool 查找 MIG 硬隔离池
func findMIGPool() string {
	return "Infer-A100-MIG-Pool"
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
func (s *Summarizer) getHighJitterPods(poolID string, startDate, endDate time.Time) ([]JitterPod, error) {
	traces, err := s.mysqlCli.GetPodTracesByPool(poolID, startDate, endDate)
	if err != nil {
		return nil, err
	}

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

// getFeatureMismatchPods 获取特性不匹配 Pod 列表
// 规则: Pod 声明需要某特性，但资源池不提供
func (s *Summarizer) getFeatureMismatchPods(poolID string, startDate, endDate time.Time) ([]FeatureMismatchPod, error) {
	_, err := s.mysqlCli.GetResourcePool(poolID)
	if err != nil {
		return nil, nil
	}

	traces, err := s.mysqlCli.GetPodTracesByPool(poolID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	var pods []FeatureMismatchPod
	for _, trace := range traces {
		// 只关注声明了需要特性的 Pod
		if trace.RequiredFeatures == "" {
			continue
		}

		wasteCost := trace.CostAmount * (1 - trace.GPUUtilAvg/100)
		pods = append(pods, FeatureMismatchPod{
			PodName:          trace.PodName,
			Namespace:        trace.Namespace,
			RequiredFeatures: trace.RequiredFeatures,
			AvgUtil:          trace.GPUUtilAvg,
			EstWasteCost:     wasteCost,
		})
	}

	return pods, nil
}

// 辅助函数
func isFullOrMIGPool(mode string) bool {
	return strings.Contains(mode, "Full") || strings.Contains(mode, "MIG")
}

func isMPSOrTSPool(mode string) bool {
	return strings.Contains(mode, "MPS") || strings.Contains(mode, "TS")
}

func hasHighEndFeatures(features string) bool {
	if features == "" {
		return false
	}
	featuresLower := strings.ToLower(features)
	return strings.Contains(featuresLower, "nvlink") ||
		strings.Contains(featuresLower, "fp8") ||
		strings.Contains(featuresLower, "rdma")
}

func calculateWasteCost(totalCost, avgUtil, threshold float64) float64 {
	if avgUtil >= threshold {
		return 0
	}
	wasteRatio := (threshold - avgUtil) / threshold
	return totalCost * wasteRatio
}

// ToMarkdown 将摘要转换为 Markdown 格式（供 LLM 使用）
func (s *InsightSummary) ToMarkdown() string {
	md := "## 资源池效能画像\n\n"
	md += "| 指标 | 值 |\n|------|-----|\n"
	md += "| 资源池 ID | " + s.PoolID + " |\n"
	md += "| 时间范围 | " + s.TimeRange + " |\n"
	md += "| 切片模式 | " + s.SlicingMode + " |\n"
	md += "| 硬件特性 | " + s.HardwareFeatures + " |\n"
	md += "| 平均利用率 | " + formatFloat(s.AvgUtilization) + "% |\n"
	md += "| 峰值利用率 | " + formatFloat(s.MaxUtilization) + "% |\n"
	md += "| 总成本 | $" + formatFloat(s.CostTotal) + " |\n"
	md += "| 浪费成本 | $" + formatFloat(s.WasteCost) + " |\n"
	md += "| Pod 数量 | " + itoa(s.PodCount) + " |\n"

	// 特性不匹配 Pod（让 LLM 语义匹配）
	if len(s.FeatureMismatchPods) > 0 {
		md += "\n### 特性声明与资源池匹配分析\n\n"
		md += "| Pod Name | Namespace | 声明需要特性 | 资源池特性 | 利用率 |\n"
		md += "|----------|-----------|-------------|----------|--------|\n"
		for _, p := range s.FeatureMismatchPods {
			md += "| " + p.PodName + " | " + p.Namespace + " | " + p.RequiredFeatures + " | " + s.HardwareFeatures + " | " + formatFloat(p.AvgUtil) + "% |\n"
		}
		md += "\n请分析以上 Pod 声明需要的特性与资源池提供的特性是否匹配（如 NVLink vs nvlink，RDMA 是否存在等），判断是否存在特性不匹配导致资源浪费。\n"
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

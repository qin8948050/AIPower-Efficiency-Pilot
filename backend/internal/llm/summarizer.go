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
	PodName        string
	Namespace      string
	TeamLabel      string // 负责团队
	PoolID         string
	SlicingMode    string
	AvgUtil        float64
	MaxUtil        float64
	Jitter         float64
	Cost           float64
	DurationHours float64   // 运行时长（小时）
	Scene          TaskScene

	// 新增字段
	TaskType     TaskType   // 任务类型: training, inference, dev
	Priority     Priority   // 优先级: high, medium, low
	HardwareDeps []string  // 硬件依赖: nvlink, fp8, mig, rdma
	GPUCount     int       // GPU 数量
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
	Task            TaskProfile
	CurrentPool     PoolProfile
	Problem         string            // 问题描述: "利用率低", "抖动高"
	ReportType      string            // "downgrade", "isolation"
	Recommendations []Recommendation  // 治理建议列表（1-2条）
	EstSavings     float64          // 默认选择第一条的预估节省
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

// identifyTaskProfile 识别任务画像
// 根据 namespace, team, annotation 等推断任务类型、优先级、硬件依赖
func (s *Summarizer) identifyTaskProfile(trace *LifeTrace) TaskProfile {
	ns := strings.ToLower(trace.Namespace)
	team := strings.ToLower(trace.TeamLabel)

	// 1. 识别任务类型
	taskType := TaskTypeDev
	if strings.Contains(ns, "train") {
		taskType = TaskTypeTraining
	} else if strings.Contains(ns, "infer") {
		taskType = TaskTypeInference
	}

	// 2. 识别优先级
	priority := PriorityMedium
	if strings.Contains(team, "core") || strings.Contains(team, "prod") || strings.Contains(team, "primary") {
		priority = PriorityHigh
	} else if strings.Contains(team, "infra") || strings.Contains(team, "platform") {
		priority = PriorityMedium
	} else if strings.Contains(team, "dev") || strings.Contains(team, "test") || strings.Contains(team, "lab") {
		priority = PriorityLow
	}

	// 3. 识别硬件依赖（从 RequiredFeatures 或 Annotation 中获取）
	hardwareDeps := parseHardwareDeps(trace.RequiredFeatures)

	// 4. GPU 数量（从资源申请中估算，暂定 1）
	gpuCount := 1
	// TODO: 从 resources.limits 中提取 GPU 数量

	return TaskProfile{
		PodName:        trace.PodName,
		Namespace:      trace.Namespace,
		TeamLabel:      trace.TeamLabel,
		PoolID:         trace.PoolID,
		SlicingMode:    trace.SlicingMode,
		AvgUtil:        trace.GPUUtilAvg,
		MaxUtil:        trace.GPUUtilMax,
		Jitter:         trace.GPUUtilMax - trace.GPUUtilAvg,
		Cost:           trace.CostAmount,
		Scene:          s.identifyTaskScene(trace),
		TaskType:       taskType,
		Priority:       priority,
		HardwareDeps:   hardwareDeps,
		GPUCount:       gpuCount,
	}
}

// parseHardwareDeps 解析硬件依赖
func parseHardwareDeps(features string) []string {
	if features == "" {
		return nil
	}
	featuresLower := strings.ToLower(features)
	var deps []string
	if strings.Contains(featuresLower, "nvlink") {
		deps = append(deps, string(HardwareDepNVLink))
	}
	if strings.Contains(featuresLower, "fp8") {
		deps = append(deps, string(HardwareDepFP8))
	}
	if strings.Contains(featuresLower, "mig") {
		deps = append(deps, string(HardwareDepMIG))
	}
	if strings.Contains(featuresLower, "rdma") {
		deps = append(deps, string(HardwareDepRDMA))
	}
	return deps
}

// hasHardwareDeps 检查是否有硬件依赖
func hasHardwareDeps(deps []string) bool {
	return len(deps) > 0
}

// checkMismatch 检查任务在当前资源池是否高效
// 根据决策流程生成 1-2 条治理建议
func (s *Summarizer) checkMismatch(trace *LifeTrace, pool *PoolProfile) *MismatchResult {
	// 1. 识别任务画像
	taskProfile := s.identifyTaskProfile(trace)

	// 计算抖动
	jitter := trace.GPUUtilMax - trace.GPUUtilAvg

	var recommendations []Recommendation
	var problem string
	var reportType string

	// 决策流程：根据问题类型生成建议

	// === 利用率低场景 ===
	if isFullOrMIGPool(pool.SlicingMode) && trace.GPUUtilAvg < LowUtilThreshold {
		problem = "利用率低"
		reportType = "downgrade"

		// 决策：
		// 1. 有硬件依赖 / High优先级 → 只降配
		// 2. 无硬件依赖 + Low优先级 → 降配+迁移
		// 3. Medium优先级 → 降配 + 降配+迁移（二选一）

		if hasHardwareDeps(taskProfile.HardwareDeps) || taskProfile.Priority == PriorityHigh {
			// 只降配
			rec := generateDowngradeRec(trace, pool, taskProfile)
			recommendations = append(recommendations, rec)
		} else if taskProfile.Priority == PriorityLow {
			// 降配+迁移到TS池
			rec := generateDowngradeMigrateRec(trace, pool, taskProfile)
			recommendations = append(recommendations, rec)
		} else {
			// Medium: 给出两种选择
			rec1 := generateDowngradeRec(trace, pool, taskProfile)
			rec2 := generateDowngradeMigrateRec(trace, pool, taskProfile)
			recommendations = append(recommendations, rec1, rec2)
		}
	}

	// === 抖动高场景 ===
	if isMPSOrTSPool(pool.SlicingMode) && jitter > HighJitterThreshold {
		problem = "抖动高"
		reportType = "isolation"

		// 决策：
		// 1. 无硬件依赖 → 迁移到MIG池
		// 2. Medium优先级 → 迁移 + 降配+迁移

		if !hasHardwareDeps(taskProfile.HardwareDeps) {
			rec := generateMigrateRec(trace, pool, taskProfile)
			recommendations = append(recommendations, rec)
		} else if taskProfile.Priority == PriorityMedium {
			rec1 := generateMigrateRec(trace, pool, taskProfile)
			rec2 := generateDowngradeMigrateRec(trace, pool, taskProfile)
			recommendations = append(recommendations, rec1, rec2)
		}
		// High优先级 + 有硬件依赖：暂不生成建议（需要保持稳定）
	}

	// 如果有建议，返回结果
	if len(recommendations) > 0 {
		estSavings := recommendations[0].EstSavings
		return &MismatchResult{
			Task:        taskProfile,
			CurrentPool: *pool,
			Problem:     problem,
			ReportType:  reportType,
			Recommendations: recommendations,
			EstSavings: estSavings,
		}
	}

	return nil
}

// generateDowngradeRec 生成降配建议
func generateDowngradeRec(trace *LifeTrace, pool *PoolProfile, profile TaskProfile) Recommendation {
	toGPU := calculateTargetGPU(profile.GPUCount, trace.GPUUtilAvg)
	estSavings := trace.CostAmount * (1 - float64(toGPU)/float64(profile.GPUCount)) * 52 * 0.7 // 年化估算

	reason := "保留当前资源池特性"
	if hasHardwareDeps(profile.HardwareDeps) {
		reason = fmt.Sprintf("保留 %s 特性", strings.Join(profile.HardwareDeps, "/"))
	}
	if profile.Priority == PriorityHigh {
		reason = "高优先级任务，保守处理"
	}

	return Recommendation{
		ActionType: string(ActionTypeDowngrade),
		FromGPU:    profile.GPUCount,
		ToGPU:      toGPU,
		FromPool:   pool.PoolID,
		ToPool:     pool.PoolID,
		EstSavings: estSavings,
		Reason:     reason,
	}
}

// generateMigrateRec 生成迁移建议
func generateMigrateRec(trace *LifeTrace, pool *PoolProfile, profile TaskProfile) Recommendation {
	targetPool := findMIGPool()
	// 迁移到MIG池通常费用增加
	estSavings := -trace.CostAmount * 0.3 * 52 * 0.5 // 负数表示增加

	return Recommendation{
		ActionType: string(ActionTypeMigrate),
		FromGPU:    profile.GPUCount,
		ToGPU:      profile.GPUCount,
		FromPool:   pool.PoolID,
		ToPool:     targetPool,
		EstSavings: estSavings,
		Reason:     "迁移到MIG硬隔离池，解决抖动问题",
	}
}

// generateDowngradeMigrateRec 生成降配+迁移建议
func generateDowngradeMigrateRec(trace *LifeTrace, pool *PoolProfile, profile TaskProfile) Recommendation {
	toGPU := calculateTargetGPU(profile.GPUCount, trace.GPUUtilAvg)
	targetPool := findDowngradePool(pool.PoolID)

	// 降配 + 迁移到低成本池，节省更多
	wasteRatio := 1 - trace.GPUUtilAvg/100
	estSavings := trace.CostAmount * wasteRatio * 52 * 0.7

	return Recommendation{
		ActionType: string(ActionTypeDowngradeMigrate),
		FromGPU:    profile.GPUCount,
		ToGPU:      toGPU,
		FromPool:   pool.PoolID,
		ToPool:     targetPool,
		EstSavings: estSavings,
		Reason:     "降配并迁移到低成本资源池，节省更多",
	}
}

// calculateTargetGPU 计算目标GPU数量
func calculateTargetGPU(currentGPU int, util float64) int {
	if util < 10 {
		return max(1, currentGPU/4)
	}
	if util < 20 {
		return max(1, currentGPU/2)
	}
	if util < 30 {
		return max(1, currentGPU*3/4)
	}
	return currentGPU
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

package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/qxw/aipower-efficiency-pilot/internal/config"
	"github.com/qxw/aipower-efficiency-pilot/internal/storage"
)

// Analyzer AI 诊断引擎
type Analyzer struct {
	mysqlCli   *storage.MySQLClient
	summarizer *Summarizer
	cfg        *config.LLMConfig
}

// NewAnalyzer 创建新的诊断引擎
func NewAnalyzer(mysqlCli *storage.MySQLClient, cfg *config.LLMConfig) *Analyzer {
	return &Analyzer{
		mysqlCli:   mysqlCli,
		summarizer: NewSummarizer(mysqlCli),
		cfg:        cfg,
	}
}

// GenerateReport 生成诊断报告
func (a *Analyzer) GenerateReport(days int) (*InsightReport, error) {
	// 核心算法：分析任务场景与资源池的匹配关系
	mismatches, err := a.summarizer.AnalyzeTaskPoolMismatch(days)
	fmt.Printf("mismatches count: %d\n", len(mismatches))
	if err != nil {
		return nil, err
	}

	if len(mismatches) == 0 {
		return &InsightReport{
			ID:             "0",
			GeneratedAt:    time.Now(),
			Problem:        "无",
			ReportType:     "general",
			Summary:        "所有任务与资源池匹配良好，未发现需要优化的场景。",
			RootCause:      "任务场景与资源池匹配合理，利用率正常。",
			Recommendations: "[]",
			EstSavings:     0,
			Status:         "pending",
		}, nil
	}

	// 选择最严重的问题（按预估节省排序）
	selectedMismatch := mismatches[0]
	for _, m := range mismatches {
		if m.EstSavings > selectedMismatch.EstSavings {
			selectedMismatch = m
		}
	}

	// 直接使用 summarizer 生成的建议
	// 将 Recommendations 序列化为 JSON
	recsJSON, _ := json.Marshal(selectedMismatch.Recommendations)

	// 构建摘要和根因
	summary := buildSummaryFromMismatch(&selectedMismatch)
	rootCause := buildRootCauseFromMismatch(&selectedMismatch)

	// 保存报告
	report := &storage.InsightReport{
		GeneratedAt:    time.Now(),
		TaskName:       selectedMismatch.Task.PodName,
		Namespace:      selectedMismatch.Task.Namespace,
		Team:           selectedMismatch.Task.TeamLabel,
		PoolID:         selectedMismatch.CurrentPool.PoolID,
		Problem:        selectedMismatch.Problem,
		ReportType:     selectedMismatch.ReportType,
		Summary:        summary,
		RootCause:      rootCause,
		Recommendations: string(recsJSON),
		EstSavings:     selectedMismatch.EstSavings,
		Status:         "pending",
	}

	if err := a.mysqlCli.SaveInsightReport(report); err != nil {
		return nil, err
	}

	// 转换返回类型
	return &InsightReport{
		ID:             fmt.Sprintf("%d", report.ID),
		GeneratedAt:    report.GeneratedAt,
		TaskName:       report.TaskName,
		Namespace:      report.Namespace,
		Team:           report.Team,
		PoolID:         report.PoolID,
		Problem:        report.Problem,
		ReportType:     report.ReportType,
		Summary:        report.Summary,
		RootCause:      report.RootCause,
		Recommendations: report.Recommendations,
		EstSavings:     report.EstSavings,
		Status:         report.Status,
	}, nil
}

// buildSummaryFromMismatch 根据 mismatch 构建摘要
func buildSummaryFromMismatch(m *MismatchResult) string {
	problem := m.Problem
	util := m.Task.AvgUtil
	jitter := m.Task.Jitter
	pool := m.CurrentPool.PoolID

	if problem == "利用率低" {
		return fmt.Sprintf("任务 [%s] 在 %s 池利用率为 %.1f%%，存在资源浪费问题。",
			m.Task.PodName, pool, util)
	}
	if problem == "抖动高" {
		return fmt.Sprintf("任务 [%s] 在 %s 池抖动达 %.1f%%，受邻居干扰严重。",
			m.Task.PodName, pool, jitter)
	}
	return fmt.Sprintf("任务 [%s] 存在优化空间。", m.Task.PodName)
}

// buildRootCauseFromMismatch 根据 mismatch 构建根因
func buildRootCauseFromMismatch(m *MismatchResult) string {
	problem := m.Problem
	priority := m.Task.Priority
	hasDeps := hasHardwareDeps(m.Task.HardwareDeps)

	if problem == "利用率低" {
		if hasDeps {
			return fmt.Sprintf("任务声明需要 %s 特性，但利用率仅 %.1f%%，可考虑降配保留特性。",
				strings.Join(m.Task.HardwareDeps, "/"), m.Task.AvgUtil)
		}
		if priority == "high" {
			return fmt.Sprintf("高优先级任务，利用率 %.1f%%，建议只降配不迁移。", m.Task.AvgUtil)
		}
		return fmt.Sprintf("任务利用率 %.1f%% 低于阈值，可迁移到低成本池或降配。", m.Task.AvgUtil)
	}
	if problem == "抖动高" {
		if hasDeps {
			return "任务有硬件依赖，为保障稳定性建议只调整资源配额。"
		}
		return fmt.Sprintf("任务抖动 %.1f%% 超过阈值，建议迁移到 MIG 硬隔离池。", m.Task.Jitter)
	}
	return "任务与资源池匹配存在问题。"
}

// selectPriorityPool 选择优先处理的资源池
// 优先级: downgrade > isolation > feature_mismatch
func selectPriorityPool(pools []InsightSummary) *InsightSummary {
	for i := range pools {
		if pools[i].IsDowngradeTarget {
			return &pools[i]
		}
	}
	for i := range pools {
		if pools[i].IsIsolationTarget {
			return &pools[i]
		}
	}
	for i := range pools {
		if pools[i].IsFeatureMismatch {
			return &pools[i]
		}
	}
	// 默认返回第一个
	if len(pools) > 0 {
		return &pools[0]
	}
	return nil
}

// determineReportType 确定报告类型（根据三种核心算法）
func determineReportType(summary *InsightSummary) string {
	// 优先级: downgrade > isolation > feature_mismatch
	if summary.IsDowngradeTarget {
		return "downgrade"
	}
	if summary.IsIsolationTarget {
		return "isolation"
	}
	if summary.IsFeatureMismatch {
		return "feature_mismatch"
	}
	return "general"
}

// LLMResponse LLM 响应结构
type LLMResponse struct {
	Summary      string
	RootCause    string
	ActionsJSON  string
}

// callLLM 调用 LLM API
func (a *Analyzer) callLLM(summary *InsightSummary) (*LLMResponse, error) {
	if a.cfg == nil || a.cfg.Provider == "" {
		return nil, fmt.Errorf("LLM not configured")
	}

	// 构建 prompt
	prompt := buildPrompt(summary)

	// 根据 provider 调用不同的 LLM
	switch a.cfg.Provider {
	case "gemini":
		return a.callGemini(prompt)
	case "openai":
		return a.callOpenAI(prompt)
	case "minimax":
		return a.callMiniMax(prompt)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", a.cfg.Provider)
	}
}

// callGemini 调用 Gemini API
func (a *Analyzer) callGemini(prompt string) (*LLMResponse, error) {
	// TODO: 实现 Gemini API 调用
	// 这里返回错误，让 fallback 处理
	return nil, fmt.Errorf("gemini not implemented")
}

// callOpenAI 调用 OpenAI API
func (a *Analyzer) callOpenAI(prompt string) (*LLMResponse, error) {
	// TODO: 实现 OpenAI API 调用
	// 这里返回错误，让 fallback 处理
	return nil, fmt.Errorf("openai not implemented")
}

// callMiniMax 调用 MiniMax API
func (a *Analyzer) callMiniMax(prompt string) (*LLMResponse, error) {
	if a.cfg.APIKey == "" {
		return nil, fmt.Errorf("minimax api key is empty")
	}

	// MiniMax API 配置
	model := a.cfg.Model
	if model == "" {
		model = "MiniMax-Text-01" // 默认使用最新模型
	}

	// 构建请求
	endpoint := a.cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://api.minimax.chat/v1/text/chatcompletion_v2"
	}

	// 构建请求体
	type Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type Request struct {
		Model       string    `json:"model"`
		Messages    []Message `json:"messages"`
		Temperature float64   `json:"temperature,omitempty"`
		MaxTokens   int      `json:"max_tokens,omitempty"`
	}

	reqBody := Request{
		Model: model,
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
		Temperature: a.cfg.Temperature,
		MaxTokens:   a.cfg.MaxTokens,
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// 发送请求
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.cfg.APIKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call minimax api: %v", err)
	}
	defer resp.Body.Close()

	// 解析响应
	type Choice struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}

	type Response struct {
		Choices []Choice `json:"choices"`
	}

	var respBody Response
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if len(respBody.Choices) == 0 {
		return nil, fmt.Errorf("no response from minimax")
	}

	// 解析 JSON 响应
	content := respBody.Choices[0].Message.Content
	return parseLLMResponse(content)
}

// parseLLMResponse 解析 LLM 响应 JSON
func parseLLMResponse(content string) (*LLMResponse, error) {
	// 尝试解析 JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// 如果不是 JSON，直接返回
		return &LLMResponse{
			Summary:    content,
			RootCause:  "LLM 返回非 JSON 格式",
			ActionsJSON: "[]",
		}, nil
	}

	response := &LLMResponse{}

	if v, ok := result["summary"].(string); ok {
		response.Summary = v
	}
	if v, ok := result["root_cause"].(string); ok {
		response.RootCause = v
	}
	if v, ok := result["actions"]; ok {
		actionsBytes, _ := json.Marshal(v)
		response.ActionsJSON = string(actionsBytes)
	}

	return response, nil
}


// generateDowngradeActions 生成降级迁移建议
// Full/MIG 池利用率 < 30% → 迁移至 MPS/TS 池
func generateDowngradeActions(summary *InsightSummary) []InsightAction {
	var actions []InsightAction

	// 找出低利用率 Pod
	targetPods := summary.LowUtilPods
	if len(targetPods) == 0 {
		// 如果没有低利用率 Pod，但被标记为 downgrade，说明整体利用率低
		targetPods = []LowUtilPod{{
			AvgUtil: summary.AvgUtilization,
		}}
	}

	for _, pod := range targetPods {
		action := InsightAction{
			Type:      "migrate",
			PodName:   pod.PodName,
			Namespace: pod.Namespace,
			FromPool:  summary.PoolID,
			ToPool:    inferTargetPoolForDowngrade(summary.PoolID),
		}
		actions = append(actions, action)
	}

	return actions
}

// generateIsolationActions 生成稳定性升级建议
// MPS/TS 池抖动 > 15% → 迁移至 MIG 硬隔离池
func generateIsolationActions(summary *InsightSummary) []InsightAction {
	var actions []InsightAction

	for _, pod := range summary.HighJitterPods {
		action := InsightAction{
			Type:      "migrate",
			PodName:   pod.PodName,
			Namespace: pod.Namespace,
			FromPool:  summary.PoolID,
			ToPool:    inferTargetPoolForIsolation(summary.PoolID),
		}
		actions = append(actions, action)
	}

	return actions
}

// generateFeatureMismatchActions 生成特性纠偏建议
// 高端特性池利用率 < 10% → 回退至通用池
func generateFeatureMismatchActions(summary *InsightSummary) []InsightAction {
	var actions []InsightAction

	for _, pod := range summary.FeatureMismatchPods {
		action := InsightAction{
			Type:      "pool_change",
			PodName:   pod.PodName,
			Namespace: pod.Namespace,
			FromPool:  summary.PoolID,
			ToPool:    inferTargetPoolForFeatureMismatch(summary.PoolID),
		}
		actions = append(actions, action)
	}

	return actions
}

// inferTargetPoolForDowngrade Full/MIG → MPS/TS
func inferTargetPoolForDowngrade(currentPool string) string {
	// 根据当前池子推断更便宜的目标池
	if strings.Contains(currentPool, "Train") {
		// 训练池: Full → MPS
		return strings.Replace(currentPool, "Full", "MPS", 1)
	}
	if strings.Contains(currentPool, "Infer") {
		// 推理池: MIG → TS
		return strings.Replace(currentPool, "MIG", "TS", 1)
	}
	// 尝试通用替换
	if strings.Contains(currentPool, "Full") {
		return strings.Replace(currentPool, "Full", "MPS", 1)
	}
	if strings.Contains(currentPool, "MIG") {
		return strings.Replace(currentPool, "MIG", "TS", 1)
	}
	return currentPool
}

// inferTargetPoolForIsolation MPS/TS → MIG
func inferTargetPoolForIsolation(currentPool string) string {
	// 寻找对应的 MIG 池
	pools := []string{
		"Infer-A100-MIG-Pool",
		"Infer-L4-MPS-Pool",  // L4 没有 MIG，尝试 L4
		"Dev-T4-TS-Pool",     // T4 没有 MIG
	}

	for _, pool := range pools {
		// 尝试匹配相似名称的 MIG 池
		if strings.Contains(currentPool, strings.Split(pool, "-")[1]) {
			// 如果是 A100 系列的 MPS，返回 MIG 池
			if strings.Contains(currentPool, "A100") && strings.Contains(pool, "A100") {
				return "Infer-A100-MIG-Pool"
			}
		}
	}

	// 默认返回 MIG 池
	return "Infer-A100-MIG-Pool"
}

// inferTargetPoolForFeatureMismatch 高端池 → 通用池
func inferTargetPoolForFeatureMismatch(currentPool string) string {
	// 降级到通用/低成本池
	if strings.Contains(currentPool, "H800") {
		return "Train-A100-Full-Pool"
	}
	if strings.Contains(currentPool, "A100") && strings.Contains(currentPool, "Full") {
		return "Infer-A100-MIG-Pool"
	}
	return "Dev-T4-TS-Pool"
}

// buildSummaryText 构建摘要文本
func buildSummaryText(summary *InsightSummary, reportType string) string {
	utilStr := fmt.Sprintf("%.1f%%", summary.AvgUtilization)
	costStr := fmt.Sprintf("%.2f", summary.CostTotal)
	wasteStr := fmt.Sprintf("%.2f", summary.WasteCost)

	switch reportType {
	case "downgrade":
		return fmt.Sprintf("资源池 %s 在过去 %s 内平均利用率为 %s，存在资源浪费问题。"+
			"检测到 %d 个低利用率 Pod (< 30%%)，建议迁移至低成本池。"+
			"总成本 $%s，预计浪费 $%s。",
			summary.PoolID, summary.TimeRange, utilStr, len(summary.LowUtilPods), costStr, wasteStr)
	case "isolation":
		return fmt.Sprintf("资源池 %s 在过去 %s 内存在算力抖动问题。"+
			"检测到 %d 个高抖动 Pod (Jitter > 15%%)，可能受邻居干扰。"+
			"建议迁移至 MIG 硬隔离池以确保稳定性。",
			summary.PoolID, summary.TimeRange, len(summary.HighJitterPods))
	case "feature_mismatch":
		return fmt.Sprintf("资源池 %s 配置了高端特性 (%s)，但平均利用率仅为 %s。"+
			"检测到 %d 个未充分利用高端特性的 Pod (< 10%%)，建议回退至通用池以节省成本。",
			summary.PoolID, summary.HardwareFeatures, utilStr, len(summary.FeatureMismatchPods))
	default:
		return fmt.Sprintf("资源池 %s 在过去 %s 内运行良好，平均利用率 %s，无需特殊处理。",
			summary.PoolID, summary.TimeRange, utilStr)
	}
}

// buildRootCause 构建根因分析
func buildRootCause(summary *InsightSummary, reportType string) string {
	switch reportType {
	case "downgrade":
		return "该池子中存在多个低利用率任务，占用了高端 GPU 资源池的配额，但实际算力消耗远低于预留资源。建议将这些任务迁移至 MPS/TS 共享池以释放高端资源。"
	case "isolation":
		return "MPS/TS 模式下多个任务共享同一 GPU，导致算力分配不稳定。部分延迟敏感型任务受邻居干扰严重，出现性能抖动。建议迁移至 MIG 硬隔离池。"
	case "feature_mismatch":
		return "该池子配置了高端硬件特性(NVLink/FP8/RDMA)，但任务的算力需求不需要这些特性，造成资源浪费。建议回退至通用资源池。"
	default:
		return "资源池与任务场景匹配良好，利用率正常。"
	}
}

// calculateEstSavings 计算预期节省
func calculateEstSavings(summary *InsightSummary) float64 {
	// 简化：年度节省 = 浪费成本 * 365 / 天数 * 折扣系数
	days := 7.0
	if summary.TimeRange == "30d" {
		days = 30
	}
	// 假设可以回收 70% 的浪费成本
	return summary.WasteCost * (365.0 / days) * 0.7
}

// calculateEstSavingsForMismatch 计算任务不匹配的预期节省/费用增加
// - downgrade/feature_mismatch: 节省（负数表示增加）
// - isolation: 增加费用（返回负数）
func calculateEstSavingsForMismatch(mismatch *MismatchResult) float64 {
	// 简化：年度值 = 任务成本 * 365 * 系数
	days := 7.0
	yearFactor := 365.0 / days

	switch mismatch.ReportType {
	case "downgrade", "feature_mismatch":
		// 降级迁移：节省费用
		return mismatch.EstSavings * yearFactor * 0.7
	case "isolation":
		// 稳定性升级：费用增加（返回负数）
		return -mismatch.EstSavings * yearFactor * 0.5
	default:
		return 0
	}
}

// buildPrompt 构建诊断 Prompt
// 核心：分析"任务场景"与"资源池"的匹配关系
func buildPrompt(summary *InsightSummary) string {
	reportType := determineReportType(summary)

	prompt := `你是一个 AI 算力效能治理专家。你的核心任务是识别"任务场景"与"资源池"的不匹配，并生成优化建议。

## 任务与资源池匹配分析
`

	// 根据报告类型添加对应的任务信息
	switch reportType {
	case "downgrade":
		prompt += `### 降级迁移分析
当前任务在 Full/MIG 高端资源池中利用率较低（<30%），资源浪费严重。
- 资源池类型: ` + summary.SlicingMode + `
- 资源池硬件特性: ` + summary.HardwareFeatures + `
- 平均利用率: ` + fmt.Sprintf("%.1f%%", summary.AvgUtilization) + `
`
		if len(summary.LowUtilPods) > 0 {
			prompt += `### 低利用率任务列表
| Pod Name | Namespace | 利用率 | 预估浪费成本 |
|----------|-----------|--------|------------|
`
			for _, p := range summary.LowUtilPods {
				prompt += fmt.Sprintf("| %s | %s | %.1f%% | $%.2f |\n", p.PodName, p.Namespace, p.AvgUtil, p.EstWasteCost)
			}
		}
		prompt += `
请分析这些任务是否真的需要高端 Full/MIG 资源，建议迁移至 MPS/TS 共享池以节省成本。`

	case "isolation":
		prompt += `### 稳定性升级分析
当前任务在 MPS/TS 共享池中受到邻居干扰，算力抖动较大（>15%）。
- 资源池类型: ` + summary.SlicingMode + `
`
		if len(summary.HighJitterPods) > 0 {
			prompt += `### 高抖动任务列表
| Pod Name | Namespace | 抖动率 |
|----------|-----------|--------|
`
			for _, p := range summary.HighJitterPods {
				prompt += fmt.Sprintf("| %s | %s | %.1f%% |\n", p.PodName, p.Namespace, p.JitterPct)
			}
		}
		prompt += `
请分析这些任务是否需要硬隔离环境，建议迁移至 MIG 硬隔离池以保证稳定性。`

	case "feature_mismatch":
		prompt += `### 特性纠偏分析
当前任务声明需要特定硬件特性，但与资源池特性不匹配。
- 资源池硬件特性: ` + summary.HardwareFeatures + `
`
		if len(summary.FeatureMismatchPods) > 0 {
			prompt += `### 特性不匹配任务列表
| Pod Name | Namespace | 声明需要特性 | 资源池特性 | 利用率 |
|----------|-----------|-------------|----------|--------|
`
			for _, p := range summary.FeatureMismatchPods {
				prompt += fmt.Sprintf("| %s | %s | %s | %s | %.1f%% |\n",
					p.PodName, p.Namespace, p.RequiredFeatures, summary.HardwareFeatures, p.AvgUtil)
			}
		}
		prompt += `
请分析这些任务是否真正需要声明的硬件特性（忽略大小写和顺序差异，如 NVLink vs nvlink），如果不需要建议回退至通用池。`

	default:
		prompt += `资源匹配良好，无需特殊处理。`
	}

	prompt += `

## 输出格式
请以 JSON 格式输出，包含以下字段：
- summary: 总结描述（简洁明了）
- root_cause: 根本原因分析
- actions: 具体优化动作数组，每项包含 type, pod_name, namespace, from_pool, to_pool

请直接输出 JSON，不要其他内容。`

	return prompt
}

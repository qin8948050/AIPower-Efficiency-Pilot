package llm

import (
	"encoding/json"
	"fmt"
	"log"
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
func (a *Analyzer) GenerateReport(poolID string, days int) (*InsightReport, error) {
	// 1. 生成数据摘要
	var summary *InsightSummary
	var err error

	if poolID != "" {
		summary, err = a.summarizer.GeneratePoolSummary(poolID, days)
	} else {
		// 全量分析 - 取第一个有数据的资源池
		pools, err := a.summarizer.GenerateAllPoolsSummary(days)
		if err != nil || len(pools) == 0 {
			return nil, fmt.Errorf("no data available")
		}
		// 选择浪费成本最高的资源池
		summary = &pools[0]
		for _, s := range pools {
			if s.WasteCost > summary.WasteCost {
				summary = &s
			}
		}
	}

	if err != nil {
		return nil, err
	}

	// 2. 确定报告类型
	reportType := determineReportType(summary)

	// 3. 调用 LLM 生成诊断建议
	llmResponse, err := a.callLLM(summary)
	if err != nil {
		log.Printf("[Analyzer] LLM call failed, using fallback: %v", err)
		llmResponse = a.fallbackAnalysis(summary)
	}

	// 4. 保存报告
	report := &storage.InsightReport{
		GeneratedAt: time.Now(),
		PoolID:      summary.PoolID,
		ReportType:  reportType,
		Summary:     llmResponse.Summary,
		RootCause:   llmResponse.RootCause,
		Actions:     llmResponse.ActionsJSON,
		EstSavings:  calculateEstSavings(summary),
		Status:      "pending",
	}

	if err := a.mysqlCli.SaveInsightReport(report); err != nil {
		return nil, err
	}

	// 转换返回类型
	return &InsightReport{
		ID:          fmt.Sprintf("%d", report.ID),
		GeneratedAt: report.GeneratedAt,
		PoolID:      report.PoolID,
		ReportType:  report.ReportType,
		Summary:     report.Summary,
		RootCause:   report.RootCause,
		Actions:     report.Actions,
		EstSavings:  report.EstSavings,
		Status:      report.Status,
	}, nil
}

// determineReportType 确定报告类型
func determineReportType(summary *InsightSummary) string {
	if len(summary.LowUtilPods) > 0 && summary.AvgUtilization < 30 {
		return "downgrade"
	}
	if len(summary.HighJitterPods) > 0 {
		return "isolation"
	}
	return "general"
}

// LLMResponse LLM 响应结构
type LLMResponse struct {
	Summary       string
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

// fallbackAnalysis 回退分析（当 LLM 不可用时）
func (a *Analyzer) fallbackAnalysis(summary *InsightSummary) *LLMResponse {
	var actions []InsightAction

	// 生成降级建议
	if len(summary.LowUtilPods) > 0 {
		for _, pod := range summary.LowUtilPods {
			action := InsightAction{
				Type:      "migrate",
				PodName:   pod.PodName,
				Namespace: pod.Namespace,
				FromPool:  summary.PoolID,
				ToPool:    inferTargetPool(summary.PoolID),
			}
			actions = append(actions, action)
		}
	}

	actionsJSON, _ := json.Marshal(actions)

	summaryText := fmt.Sprintf("资源池 %s 在过去 %s 内平均利用率为 %.1f%%，总成本 $%.2f，预计浪费成本 $%.2f。",
		summary.PoolID, summary.TimeRange, summary.AvgUtilization, summary.CostTotal, summary.WasteCost)

	rootCause := "资源池中存在多个低利用率任务，占用了高端资源池的配额，导致资源浪费。"

	return &LLMResponse{
		Summary:      summaryText,
		RootCause:    rootCause,
		ActionsJSON:  string(actionsJSON),
	}
}

// inferTargetPool 推断目标资源池
func inferTargetPool(currentPool string) string {
	// 根据当前池子推断更便宜的目标池
	if contains(currentPool, "Full") {
		return getPoolByMode(currentPool, "MPS")
	}
	if contains(currentPool, "MIG") {
		return getPoolByMode(currentPool, "TS")
	}
	return currentPool
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func getPoolByMode(pool, mode string) string {
	// 简单替换模式标识
	for _, m := range []string{"Full", "MIG", "MPS", "TS"} {
		if contains(pool, m) {
			return replace(pool, m, mode)
		}
	}
	return pool
}

func replace(s, old, new string) string {
	result := s
	for i := 0; i <= len(s)-len(old); i++ {
		if s[i:i+len(old)] == old {
			result = s[:i] + new + s[i+len(old):]
			break
		}
	}
	return result
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

// buildPrompt 构建诊断 Prompt
func buildPrompt(summary *InsightSummary) string {
	prompt := `你是一个 AI 算力效能治理专家。请分析以下资源池数据，生成优化建议。

## 资源池数据
` + summary.ToMarkdown() + `

## 分析要求
1. 分析资源利用效率
2. 识别资源浪费的根本原因
3. 提出具体的优化动作（迁移、降级等）
4. 估算年度节省成本

## 输出格式
请以 JSON 格式输出，包含以下字段：
- summary: 总结描述
- root_cause: 根本原因分析
- actions: 具体优化动作数组，每项包含 type, pod_name, namespace, from_pool, to_pool

请直接输出 JSON，不要其他内容。`

	return prompt
}

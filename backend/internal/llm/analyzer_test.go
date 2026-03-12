package llm

import (
	"fmt"
	"testing"

	"github.com/qxw/aipower-efficiency-pilot/internal/config"
	"github.com/qxw/aipower-efficiency-pilot/internal/storage"
)

// TestGenerateReportFullFlow 测试完整的报告生成流程（V2 任务视角）
func TestGenerateReportFullFlow(t *testing.T) {
	// 1. 加载配置
	cfg, err := config.LoadConfig("../../configs/config.yaml")
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 2. 初始化 MySQL 客户端
	mysqlCli, err := storage.NewMySQLClient(cfg.MySQL.DSN)
	if err != nil {
		t.Fatalf("连接 MySQL 失败: %v", err)
	}

	// 3. 创建 Analyzer
	analyzer := NewAnalyzer(mysqlCli, &cfg.LLM)

	// 4. 测试生成报告（任务视角，不指定 poolID）
	fmt.Println("开始生成诊断报告（任务视角）...")

	report, err := analyzer.GenerateReport(7)
	if err != nil {
		t.Fatalf("生成报告失败: %v", err)
	}

	// 5. 打印结果
	fmt.Println("\n========== 诊断报告结果 ==========")
	fmt.Printf("报告 ID: %s\n", report.ID)
	fmt.Printf("任务名: %s\n", report.TaskName)
	fmt.Printf("命名空间: %s\n", report.Namespace)
	fmt.Printf("团队: %s\n", report.Team)
	fmt.Printf("资源池: %s\n", report.PoolID)
	fmt.Printf("问题: %s\n", report.Problem)
	fmt.Printf("报告类型: %s\n", report.ReportType)
	fmt.Printf("状态: %s\n", report.Status)
	fmt.Printf("生成时间: %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("\n摘要:\n%s\n", report.Summary)
	fmt.Printf("\n根因分析:\n%s\n", report.RootCause)
	fmt.Printf("\n建议动作:\n%s\n", report.Recommendations)
	fmt.Printf("\n预估年度节省: ¥%.2f\n", report.EstSavings)
	fmt.Println("====================================")
}

// TestGeneratePoolSummary 测试数据脱水降维功能
func TestGeneratePoolSummary(t *testing.T) {
	// 1. 加载配置
	cfg, err := config.LoadConfig("../../configs/config.yaml")
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 2. 初始化 MySQL 客户端
	mysqlCli, err := storage.NewMySQLClient(cfg.MySQL.DSN)
	if err != nil {
		t.Fatalf("连接 MySQL 失败: %v", err)
	}

	// 3. 创建 Summarizer
	summarizer := NewSummarizer(mysqlCli)

	// 4. 测试数据脱水
	summary, err := summarizer.GeneratePoolSummary("Train-A100-Full-Pool", 7)
	if err != nil {
		t.Fatalf("数据脱水失败: %v", err)
	}

	// 5. 验证结果
	fmt.Println("\n========== 数据脱水结果 ==========")
	fmt.Printf("资源池: %s\n", summary.PoolID)
	fmt.Printf("时间范围: %s\n", summary.TimeRange)
	fmt.Printf("平均利用率: %.2f%%\n", summary.AvgUtilization)
	fmt.Printf("最大利用率: %.2f%%\n", summary.MaxUtilization)
	fmt.Printf("总成本: ¥%.2f\n", summary.CostTotal)
	fmt.Printf("浪费成本: ¥%.2f\n", summary.WasteCost)
	fmt.Printf("Pod 数量: %d\n", summary.PodCount)
	fmt.Printf("低利用率 Pod 数量: %d\n", len(summary.LowUtilPods))

	if summary.IsDowngradeTarget {
		fmt.Println("⚠️ 该池是降级目标（利用率 < 30%）")
	}
	if summary.IsIsolationTarget {
		fmt.Println("⚠️ 该池是隔离目标（抖动 > 15%）")
	}
	if summary.IsFeatureMismatch {
		fmt.Println("⚠️ 该池存在特性不匹配问题")
	}
	fmt.Println("==================================")
}

// TestMismatchDetection 测试任务与资源池不匹配检测
func TestMismatchDetection(t *testing.T) {
	// 1. 加载配置
	cfg, err := config.LoadConfig("../../configs/config.yaml")
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 2. 初始化 MySQL 客户端
	mysqlCli, err := storage.NewMySQLClient(cfg.MySQL.DSN)
	if err != nil {
		t.Fatalf("连接 MySQL 失败: %v", err)
	}

	// 3. 创建 Summarizer
	summarizer := NewSummarizer(mysqlCli)

	// 4. 测试不匹配检测
	results, err := summarizer.AnalyzeTaskPoolMismatch(7)
	if err != nil {
		t.Fatalf("不匹配检测失败: %v", err)
	}

	// 5. 打印结果
	fmt.Println("\n========== 任务-资源池不匹配检测结果 ==========")
	fmt.Printf("检测到 %d 个不匹配任务\n", len(results))

	for i, result := range results {
		if i >= 5 { // 只打印前5个
			fmt.Println("... (更多结果省略)")
			break
		}
		fmt.Printf("\n任务 %d: %s/%s\n", i+1, result.Task.Namespace, result.Task.PodName)
		fmt.Printf("  当前池: %s\n", result.Task.PoolID)
		fmt.Printf("  任务类型: %s, 优先级: %s\n", result.Task.TaskType, result.Task.Priority)
		fmt.Printf("  硬件依赖: %v\n", result.Task.HardwareDeps)
		fmt.Printf("  平均利用率: %.2f%%\n", result.Task.AvgUtil)
		fmt.Printf("  问题: %s\n", result.Problem)
		fmt.Printf("  报告类型: %s\n", result.ReportType)
		fmt.Printf("  建议数量: %d\n", len(result.Recommendations))
		for j, rec := range result.Recommendations {
			fmt.Printf("    建议%d: %s, GPU %d→%d, %s→%s, 节省 ¥%.0f\n",
				j+1, rec.ActionType, rec.FromGPU, rec.ToGPU, rec.FromPool, rec.ToPool, rec.EstSavings)
		}
	}
	fmt.Println("==========================================")
}

// TestReportPersistence 测试报告持久化
func TestReportPersistence(t *testing.T) {
	// 1. 加载配置
	cfg, err := config.LoadConfig("../../configs/config.yaml")
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 2. 初始化 MySQL 客户端
	mysqlCli, err := storage.NewMySQLClient(cfg.MySQL.DSN)
	if err != nil {
		t.Fatalf("连接 MySQL 失败: %v", err)
	}

	// 3. 查询已保存的报告
	reports, total, err := mysqlCli.GetInsightReports("", "", 10, 0)
	if err != nil {
		t.Fatalf("查询报告失败: %v", err)
	}

	// 4. 打印结果
	fmt.Println("\n========== 已保存的报告 ==========")
	fmt.Printf("共 %d 条报告\n", total)

	for i, report := range reports {
		if i >= 5 { // 只打印前5个
			fmt.Println("... (更多结果省略)")
			break
		}
		fmt.Printf("\n报告 %d:\n", i+1)
		fmt.Printf("  ID: %d\n", report.ID)
		fmt.Printf("  任务: %s (%s)\n", report.TaskName, report.Namespace)
		fmt.Printf("  团队: %s\n", report.Team)
		fmt.Printf("  资源池: %s\n", report.PoolID)
		fmt.Printf("  问题: %s\n", report.Problem)
		fmt.Printf("  状态: %s\n", report.Status)
		fmt.Printf("  预估节省: ¥%.2f\n", report.EstSavings)
		if report.ApprovedRecommendation != "" {
			fmt.Printf("  已批准建议: %s\n", report.ApprovedRecommendation)
		}
	}
	fmt.Println("==================================")
}

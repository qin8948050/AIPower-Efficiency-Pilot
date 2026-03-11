package llm

import (
	"fmt"
	"testing"

	"github.com/qxw/aipower-efficiency-pilot/internal/config"
	"github.com/qxw/aipower-efficiency-pilot/internal/storage"
)

// TestGenerateReportFullFlow 测试完整的报告生成流程
func TestGenerateReportFullFlow(t *testing.T) {
	// 1. 加载配置（从 backend 目录运行，使用空字符串让 Viper 自动搜索）
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

	// 4. 测试生成报告（不指定 poolID，自动选择问题最严重的池）
	fmt.Println("开始生成诊断报告...")

	report, err := analyzer.GenerateReport("", 7) // 空 poolID = 全量分析
	if err != nil {
		t.Fatalf("生成报告失败: %v", err)
	}

	// 5. 打印结果
	fmt.Println("\n========== 诊断报告结果 ==========")
	fmt.Printf("报告 ID: %s\n", report.ID)
	fmt.Printf("资源池: %s\n", report.PoolID)
	fmt.Printf("报告类型: %s\n", report.ReportType)
	fmt.Printf("状态: %s\n", report.Status)
	fmt.Printf("生成时间: %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("\n摘要:\n%s\n", report.Summary)
	fmt.Printf("\n根因分析:\n%s\n", report.RootCause)
	fmt.Printf("\n建议动作: %s\n", report.Actions)
	fmt.Printf("\n预估年度节省: ¥%.2f\n", report.EstSavings)
	fmt.Println("====================================")
}

// TestGenerateReportByPoolID 测试指定资源池的报告生成
func TestGenerateReportByPoolID(t *testing.T) {
	// 1. 加载配置
	cfg, err := config.LoadConfig("")
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

	// 4. 测试生成指定池的报告
	poolIDs := []string{
		"Train-A100-Full-Pool",
		"Train-H800-Full-Pool",
		"Infer-A100-MIG-Pool",
		"Dev-T4-TS-Pool",
	}

	for _, poolID := range poolIDs {
		fmt.Printf("\n========== 测试资源池: %s ==========\n", poolID)

		report, err := analyzer.GenerateReport(poolID, 7)
		if err != nil {
			t.Logf("生成报告失败: %v", err)
			continue
		}

		fmt.Printf("报告类型: %s\n", report.ReportType)
		fmt.Printf("摘要: %s\n", report.Summary)
		fmt.Printf("根因: %s\n", report.RootCause)
		fmt.Printf("动作: %s\n", report.Actions)
		fmt.Printf("预估节省: ¥%.2f\n", report.EstSavings)
	}
}

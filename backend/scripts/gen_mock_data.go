package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/qxw/aipower-efficiency-pilot/internal/storage"
)

func main() {
	log.Println("Generating mock data for Phase 3 demo...")

	// 初始化 MySQL
	mysqlCli, err := storage.NewMySQLClient("qxw:Gaoji_001#@tcp(10.8.132.147:3306)/aipower?parseTime=true")
	if err != nil {
		log.Fatalf("Failed to connect MySQL: %v", err)
	}

	// 1. 创建资源池
	poolConfigs := []struct {
		ID               string
		Name             string
		Scene            string
		GPUModel         string
		HardwareFeatures string
		SlicingMode      string
		Priority         string
		Description      string
	}{
		{ID: "infer-a100-full-pool", Name: "A100推理完整池", Scene: "推理", GPUModel: "A100-80GB", HardwareFeatures: "NVLink", SlicingMode: "Full", Priority: "High", Description: "生产环境推理池"},
		{ID: "train-a100-mig-pool", Name: "A100训练MIG池", Scene: "训练", GPUModel: "A100-40GB", HardwareFeatures: "MIG", SlicingMode: "MIG", Priority: "High", Description: "训练任务MIG池"},
		{ID: "dev-h100-mps-pool", Name: "H100研发MPS池", Scene: "研发", GPUModel: "H100", HardwareFeatures: "MPS", SlicingMode: "MPS", Priority: "Low", Description: "研发测试MPS池"},
		{ID: "batch-v100-ts-pool", Name: "V100批处理TS池", Scene: "批处理", GPUModel: "V100", HardwareFeatures: "TimeSlicing", SlicingMode: "TS", Priority: "Low", Description: "批处理任务TS池"},
	}

	for _, p := range poolConfigs {
		mysqlCli.UpsertResourcePool(&storage.ResourcePool{
			ID:               p.ID,
			Name:             p.Name,
			Scene:            p.Scene,
			GPUModel:         p.GPUModel,
			HardwareFeatures: p.HardwareFeatures,
			SlicingMode:      p.SlicingMode,
			Priority:         p.Priority,
			Description:      p.Description,
		})
		log.Printf("Created pool: %s", p.ID)
	}

	// 2. 创建每日账单快照 (过去7天)
	baseDate := time.Now().AddDate(0, 0, -7)
	poolCosts := map[string]float64{
		"infer-a100-full-pool":  1500.0,
		"train-a100-mig-pool":   2200.0,
		"dev-h100-mps-pool":     800.0,
		"batch-v100-ts-pool":    500.0,
	}

	namespaces := []string{"ml-platform", "default", "research", "production"}

	for i := 0; i < 7; i++ {
		date := baseDate.AddDate(0, 0, i)
		dateStr := date.Format("2006-01-02")

		for poolID, baseCost := range poolCosts {
			// 添加随机波动
			cost := baseCost * (0.8 + rand.Float64()*0.4)
			util := 20.0 + rand.Float64()*60 // 20-80% 利用率

			// 低利用率池子 (模拟浪费)
			if poolID == "infer-a100-full-pool" && i%2 == 0 {
				util = 15.0 + rand.Float64()*15 // 15-30% 低利用率
			}

			snapshot := &storage.DailyBillingSnapshot{
				SnapshotDate:    dateStr,
				PoolID:         poolID,
				Namespace:       namespaces[rand.Intn(len(namespaces))],
				TotalCost:       cost,
				AvgUtilP95:      util,
				MaxMemGiB:       40 + rand.Float64()*40,
				PodSessionCount: 5 + rand.Intn(20),
			}
			mysqlCli.UpsertDailySnapshot(snapshot)
		}
	}
	log.Println("Created daily billing snapshots")

	// 3. 创建 Pod 会话记录 (带低利用率数据)
	sessions := []struct {
		PodName      string
		Namespace    string
		PoolID       string
		SlicingMode  string
		GPUUtilAvg   float64
		CostAmount   float64
	}{
		// 低利用率 Pod (需要降级)
		{"llm-inference-001", "ml-platform", "infer-a100-full-pool", "Full", 18.5, 45.2},
		{"llm-inference-002", "ml-platform", "infer-a100-full-pool", "Full", 22.3, 52.1},
		{"llm-inference-003", "ml-platform", "infer-a100-full-pool", "Full", 25.0, 48.9},
		{"bert-serving-001", "production", "infer-a100-full-pool", "Full", 28.7, 35.6},

		// 中等利用率 Pod
		{"stable-diffusion-001", "research", "train-a100-mig-pool", "MIG", 55.2, 120.5},
		{"gpt-finetune-001", "ml-platform", "train-a100-mig-pool", "MIG", 62.8, 180.3},

		// 高利用率 Pod
		{"distributed-training-001", "ml-platform", "train-a100-mig-pool", "MIG", 85.5, 250.8},
		{"distributed-training-002", "ml-platform", "train-a100-mig-pool", "MIG", 88.2, 265.4},

		// 研发池 Pod
		{"dev-experiment-001", "default", "dev-h100-mps-pool", "MPS", 35.6, 15.2},
		{"dev-experiment-002", "default", "dev-h100-mps-pool", "MPS", 42.1, 18.7},

		// 批处理池 Pod
		{"batch-job-001", "default", "batch-v100-ts-pool", "TS", 60.5, 8.5},
		{"batch-job-002", "default", "batch-v100-ts-pool", "TS", 55.8, 7.2},
	}

	startTime := time.Now().AddDate(0, 0, -7)
	for i, s := range sessions {
		start := startTime.Add(time.Duration(i*12) * time.Hour)
		end := start.Add(time.Duration(4+rand.Intn(20)) * time.Hour)

		trace := &storage.LifeTrace{
			PodUID:       fmt.Sprintf("mock-pod-%d", i),
			PodName:      s.PodName,
			Namespace:    s.Namespace,
			NodeName:     fmt.Sprintf("node-%d", i%5),
			PoolID:       s.PoolID,
			SlicingMode:  s.SlicingMode,
			StartTime:    start,
			EndTime:      &end,
			TeamLabel:    s.Namespace,
			ProjectLabel: "demo-project",
			GPUUtilAvg:   s.GPUUtilAvg,
			GPUUtilMax:   s.GPUUtilAvg + 10 + rand.Float64()*15,
			MemUsedMax:   uint64(20 + rand.Intn(60)) * 1024 * 1024 * 1024,
			PowerUsageAvg: 150 + rand.Float64()*200,
			CostAmount:   s.CostAmount,
		}
		mysqlCli.SaveRawLifeTrace(trace)
	}
	log.Println("Created pod session records")

	// 4. 创建 AI 诊断报告 (insight_reports)
	reports := []struct {
		PoolID     string
		ReportType string
		Summary    string
		RootCause  string
		Actions    string
		EstSavings float64
		Status     string
	}{
		{
			PoolID:     "infer-a100-full-pool",
			ReportType: "downgrade",
			Summary:    "资源池 infer-a100-full-pool 在过去7天内平均利用率仅为22.5%，存在严重的资源浪费。",
			RootCause:  "该池子中存在4个低利用率推理任务(利用率15-30%)，占用了高端A100 GPU资源，建议迁移至MPS或TS池。",
			Actions:    `[{"type":"migrate","pod_name":"llm-inference-001","namespace":"ml-platform","from_pool":"infer-a100-full-pool","to_pool":"dev-h100-mps-pool"},{"type":"migrate","pod_name":"llm-inference-002","namespace":"ml-platform","from_pool":"infer-a100-full-pool","to_pool":"dev-h100-mps-pool"},{"type":"migrate","pod_name":"llm-inference-003","namespace":"ml-platform","from_pool":"infer-a100-full-pool","to_pool":"batch-v100-ts-pool"}]`,
			EstSavings: 15600.0,
			Status:     "pending",
		},
		{
			PoolID:     "train-a100-mig-pool",
			ReportType: "general",
			Summary:    "资源池 train-a100-mig-pool 利用率良好，平均65.5%，运行稳定。",
			RootCause:  "训练任务资源利用充分，MIG硬隔离效果良好，无需特殊处理。",
			Actions:    "[]",
			EstSavings: 0,
			Status:     "pending",
		},
		{
			PoolID:     "dev-h100-mps-pool",
			ReportType: "isolation",
			Summary:    "资源池 dev-h100-mps-pool 中部分任务存在算力抖动，可能受邻居干扰。",
			RootCause:  "MPS模式下资源共享导致部分任务性能波动，建议对高优先级任务启用MIG隔离。",
			Actions:    `[{"type":"pool_change","pod_name":"dev-experiment-001","namespace":"default","from_pool":"dev-h100-mps-pool","to_pool":"train-a100-mig-pool"}]`,
			EstSavings: 3500.0,
			Status:     "approved",
		},
		{
			PoolID:     "infer-a100-full-pool",
			ReportType: "downgrade",
			Summary:    "上一周期的降级建议已执行，资源利用率提升至45%。",
			RootCause:  "通过将低利用率任务迁移至低成本池，释放了高端GPU资源。",
			Actions:    "[]",
			EstSavings: 8500.0,
			Status:     "approved",
		},
	}

	reportTime := time.Now().AddDate(0, 0, -1)
	for i, r := range reports {
		generatedAt := reportTime.Add(-time.Duration(i*12) * time.Hour)
		report := &storage.InsightReport{
			GeneratedAt: generatedAt,
			PoolID:      r.PoolID,
			ReportType:  r.ReportType,
			Summary:     r.Summary,
			RootCause:   r.RootCause,
			Actions:     r.Actions,
			EstSavings:  r.EstSavings,
			Status:      r.Status,
		}
		mysqlCli.SaveInsightReport(report)
	}
	log.Println("Created insight reports")

	// 打印汇总
	fmt.Println("\n========== Mock Data Summary ==========")
	fmt.Println("✓ Resource Pools: 4")
	fmt.Println("✓ Daily Snapshots: 28 (7 days x 4 pools)")
	fmt.Println("✓ Pod Sessions: 13")
	fmt.Println("✓ Insight Reports: 4")
	fmt.Println("========================================")
	fmt.Println("\nDemo data ready for Phase 3 testing!")
	fmt.Println("Visit /insights to see AI diagnosis reports.")
}

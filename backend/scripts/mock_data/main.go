package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/qxw/aipower-efficiency-pilot/internal/config"
	"github.com/qxw/aipower-efficiency-pilot/internal/storage"
	"github.com/qxw/aipower-efficiency-pilot/internal/types"
)

func main() {
	// 1. 加载配置
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("无法加载配置: %v", err)
	}

	// 2. 初始化存储客户端
	mysqlClient, err := storage.NewMySQLClient(cfg.MySQL.DSN)
	if err != nil {
		log.Fatalf("无法连接 MySQL: %v", err)
	}

	redisClient, err := storage.NewRedisClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Fatalf("无法连接 Redis: %v", err)
	}

	// 2.5 清空数据
	fmt.Println("正在清理 MySQL 和 Redis...")
	mysqlClient.TruncateTable() // 清空 life_trace
	// 手动清理其他表 (演示环境)
	mysqlClient.RawExec("TRUNCATE TABLE pool_pricing")
	mysqlClient.RawExec("TRUNCATE TABLE daily_billing_snapshot")
	mysqlClient.RawExec("TRUNCATE TABLE resource_pool")
	mysqlClient.RawExec("TRUNCATE TABLE insight_reports")
	redisClient.FlushDB()

	// 2.6 注入定价配置 (Phase 2 核心)
	fmt.Println("注入资源池定价与资产配置...")
	pricingList := []storage.PoolPricing{
		{PoolID: "Train-H800-Full-Pool", GPUModel: "NVIDIA H800", BasePricePerHour: 85.0, SlicingWeightFull: 1.0, SlicingWeightMIG: 0.4, SlicingWeightMPS: 0.6, SlicingWeightTS: 0.7},
		{PoolID: "Train-A100-Full-Pool", GPUModel: "NVIDIA A100", BasePricePerHour: 55.0, SlicingWeightFull: 1.0, SlicingWeightMIG: 0.4, SlicingWeightMPS: 0.6, SlicingWeightTS: 0.7},
		{PoolID: "Infer-A100-MIG-Pool", GPUModel: "NVIDIA A100", BasePricePerHour: 55.0, SlicingWeightFull: 1.0, SlicingWeightMIG: 0.35, SlicingWeightMPS: 0.5, SlicingWeightTS: 0.6},
		{PoolID: "Infer-L4-MPS-Pool", GPUModel: "NVIDIA L4", BasePricePerHour: 18.0, SlicingWeightFull: 1.0, SlicingWeightMIG: 0.3, SlicingWeightMPS: 0.4, SlicingWeightTS: 0.5},
		{PoolID: "Dev-T4-TS-Pool", GPUModel: "NVIDIA T4", BasePricePerHour: 12.0, SlicingWeightFull: 1.0, SlicingWeightMIG: 0.25, SlicingWeightMPS: 0.35, SlicingWeightTS: 0.4},
	}
	for _, p := range pricingList {
		mysqlClient.SavePoolPricing(&p)
	}

	poolAssets := []storage.ResourcePool{
		{ID: "Train-H800-Full-Pool", Name: "万卡大模型预训练池", Scene: "大模型预训练", GPUModel: "NVIDIA H800", HardwareFeatures: "NVLink,RDMA,FP8", SlicingMode: "Full", PricingLogic: "资源预留 (Reserved)", Priority: "High", Description: "核心算力底座，用于基座模型训练"},
		{ID: "Train-A100-Full-Pool", Name: "模型全量微调池", Scene: "模型微调", GPUModel: "NVIDIA A100", HardwareFeatures: "NVLink,RDMA", SlicingMode: "Full", PricingLogic: "资源预留 (Reserved)", Priority: "High", Description: "用于部门核心模型 SFT"},
		{ID: "Infer-A100-MIG-Pool", Name: "核心推理服务池", Scene: "核心推理", GPUModel: "NVIDIA A100", HardwareFeatures: "Multi-Instance GPU", SlicingMode: "MIG", PricingLogic: "按规格计费", Priority: "High", Description: "保障核心线上推理 SLA"},
		{ID: "Infer-L4-MPS-Pool", Name: "高并发并行推理池", Scene: "小模型推理", GPUModel: "NVIDIA L4", HardwareFeatures: "MPS Parallel", SlicingMode: "MPS", PricingLogic: "吞吐量分摊", Priority: "Medium", Description: "低延迟、高吞吐推理场景"},
		{ID: "Dev-T4-TS-Pool", Name: "研发测试超分池", Scene: "研发调试", GPUModel: "NVIDIA T4", HardwareFeatures: "Time-Slicing", SlicingMode: "TS", PricingLogic: "极低单价 (Spot)", Priority: "Low", Description: "仅用于 CI/CD 和研发测试"},
	}
	for _, a := range poolAssets {
		mysqlClient.UpsertResourcePool(&a)
		mysqlClient.UpdateResourcePoolMetadata(&a)
	}

	// 3. 定义模拟数据
	fmt.Println("开始注入新的模拟数据...")
	namespaces := []string{"default", "ai-platform", "data-science", "inference-prod"}
	teams := []string{"CV-Team", "NLP-Group", "Search-Algo", "Infrastructure"}
	nodes := []string{"gpu-node-01", "gpu-node-02", "gpu-node-03", "gpu-node-04", "gpu-node-05"}
	pools := []string{"Train-H800-Full-Pool", "Train-A100-Full-Pool", "Infer-A100-MIG-Pool", "Infer-L4-MPS-Pool", "Dev-T4-TS-Pool"}
	modes := []types.SlicingMode{types.SlicingModeMIG, types.SlicingModeMPS, types.SlicingModeTS, types.SlicingModeFull}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// 4. 建立节点池对应关系 (Redis)
	for i, node := range nodes {
		poolID := pools[i%len(pools)]
		redisClient.SaveNodePoolID(node, poolID)
	}

	// 5. 生成历史账单快照 (Daily Snapshots - 过去 7 天)
	fmt.Println("生成过去 7 天的日级账单快照...")
	for i := 7; i >= 1; i-- {
		dateStr := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		for _, pool := range pools {
			for _, ns := range namespaces {
				avgUtil := 10 + r.Float64()*80
				// 针对性生成测试数据触发三种算法
				if pool == "Train-A100-Full-Pool" && ns == "ai-platform" {
					// 算法1: 降级 - Full池利用率<30%持续3天 (最近3天都是低利用率)
					if i <= 3 {
						avgUtil = 15 + r.Float64()*10 // 15-25%
					}
				} else if pool == "Train-H800-Full-Pool" && ns == "default" {
					// 算法3: 特性纠偏 - NVLink池利用率<10%
					avgUtil = 3 + r.Float64()*5 // 3-8%
				} else if pool == "Dev-T4-TS-Pool" {
					// 其他池随机
					avgUtil = 10 + r.Float64()*80
				}

				snapshot := &storage.DailyBillingSnapshot{
					SnapshotDate:    dateStr,
					PoolID:          pool,
					Namespace:       ns,
					TeamLabel:       teams[r.Intn(len(teams))],
					TotalCost:       200 + r.Float64()*1000,
					AvgUtilP95:      avgUtil,
					MaxMemGiB:       4 + r.Float64()*28,
					PodSessionCount: 5 + r.Intn(20),
				}
				mysqlClient.UpsertDailySnapshot(snapshot)
			}
		}
	}

	// 6. 生成历史会话记录 (Stitched LifeTrace)
	fmt.Println("生成已结算的会话明细...")
	for i := 1; i <= 30; i++ {
		ns := namespaces[r.Intn(len(namespaces))]
		node := nodes[r.Intn(len(nodes))]
		poolID := pools[r.Intn(len(pools))]
		team := teams[r.Intn(len(teams))]

		startTime := time.Now().Add(-time.Duration(r.Intn(48)) * time.Hour)
		endTime := startTime.Add(time.Duration(30+r.Intn(300)) * time.Minute)

		// 针对性生成测试数据触发三种算法
		avgUtil := 10 + r.Float64()*80
		maxUtil := avgUtil + r.Float64()*20

		if poolID == "Train-A100-Full-Pool" {
			// 算法1: 降级 - Full池利用率<30%
			avgUtil = 15 + r.Float64()*10 // 15-25%
			maxUtil = avgUtil + 10
		} else if poolID == "Dev-T4-TS-Pool" {
			// 算法2: 隔离 - TS池高抖动 (>15%)
			avgUtil = 40 + r.Float64()*30
			maxUtil = avgUtil + 20 + r.Float64()*10 // 抖动 > 15%
		} else if poolID == "Train-H800-Full-Pool" {
			// 算法3: 特性纠偏 - NVLink池利用率<10%
			avgUtil = 3 + r.Float64()*5 // 3-8%
			maxUtil = avgUtil + 5
		}

		trace := &storage.LifeTrace{
			PodUID:        fmt.Sprintf("hist-uid-%d", i),
			Namespace:     ns,
			PodName:       fmt.Sprintf("offline-job-%03d", i),
			NodeName:      node,
			PoolID:        poolID,
			SlicingMode:   string(modes[r.Intn(len(modes))]),
			StartTime:     startTime,
			EndTime:       &endTime,
			Status:        "Settled",
			TeamLabel:     team,
			GPUUtilAvg:    avgUtil,
			GPUUtilMax:    maxUtil,
			MemUsedMax:    uint64(2048 + r.Intn(30000)),
			PowerUsageAvg: 150 + r.Float64()*100,
			CostAmount:    5 + r.Float64()*50,
		}
		mysqlClient.SaveRawLifeTrace(trace)
	}

	// 7. 生成活跃 Pod 数据 (Phase 1)
	fmt.Println("生成当前活跃 Pod...")
	for i := 1; i <= 5; i++ {
		ns := namespaces[r.Intn(len(namespaces))]
		node := nodes[r.Intn(len(nodes))]
		poolID, _ := redisClient.GetNodePoolID(node)
		podName := fmt.Sprintf("active-task-%03d", i)
		mode := modes[r.Intn(len(modes))]

		trace := &types.PodTrace{
			Namespace:   ns,
			PodName:     podName,
			PodUID:      fmt.Sprintf("active-uid-%d", i),
			NodeName:    node,
			PoolID:      poolID,
			SlicingMode: mode,
			TeamLabel:   teams[r.Intn(len(teams))],
			StartTime:   time.Now().Add(-time.Duration(r.Intn(5)) * time.Hour),
			Metrics: &types.GPUMetrics{
				GPUUtilAvg:    r.Float64() * 100,
				GPUUtilMax:    r.Float64() * 100,
				MemUsedBytes:  uint64(r.Int63n(80 * 1024 * 1024 * 1024)),
				MemTotalBytes: 80 * 1024 * 1024 * 1024,
				PowerUsageW:   100 + r.Float64()*300,
				LastUpdate:    time.Now(),
			},
		}

		mysqlClient.SaveLifeTrace(trace)
		redisClient.SavePodTrace(trace)
	}

	// 8. 生成 AI 诊断报告 (Phase 3)
	fmt.Println("生成 AI 诊断报告...")
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
			PoolID:     "Train-A100-Full-Pool",
			ReportType: "downgrade",
			Summary:    "资源池 Train-A100-Full-Pool 在过去7天内平均利用率仅为22.5%，存在严重的资源浪费。",
			RootCause:  "该池子中存在多个低利用率训练任务(利用率15-30%)，占用了高端A100 GPU资源，建议迁移至MPS或TS池。",
			Actions:    `[{"type":"migrate","pod_name":"offline-job-001","namespace":"ai-platform","from_pool":"Train-A100-Full-Pool","to_pool":"Dev-T4-TS-Pool"},{"type":"migrate","pod_name":"offline-job-005","namespace":"data-science","from_pool":"Train-A100-Full-Pool","to_pool":"Infer-L4-MPS-Pool"}]`,
			EstSavings: 12500.0,
			Status:     "pending",
		},
		{
			PoolID:     "Infer-A100-MIG-Pool",
			ReportType: "general",
			Summary:    "资源池 Infer-A100-MIG-Pool 利用率良好，平均65.5%，运行稳定。",
			RootCause:  "推理任务资源利用充分，MIG硬隔离效果良好，无需特殊处理。",
			Actions:    "[]",
			EstSavings: 0,
			Status:     "pending",
		},
		{
			PoolID:     "Dev-T4-TS-Pool",
			ReportType: "isolation",
			Summary:    "资源池 Dev-T4-TS-Pool 中部分任务存在算力抖动，可能受邻居干扰。",
			RootCause:  "Time-Slicing模式下资源共享导致部分任务性能波动，建议对高优先级任务启用MPS隔离。",
			Actions:    `[{"type":"pool_change","pod_name":"active-task-003","namespace":"default","from_pool":"Dev-T4-TS-Pool","to_pool":"Infer-L4-MPS-Pool"}]`,
			EstSavings: 2800.0,
			Status:     "approved",
		},
		{
			PoolID:     "Train-H800-Full-Pool",
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
		mysqlClient.SaveInsightReport(report)
	}
	fmt.Println("Created insight reports")

	fmt.Println("\n全量 Mock 数据注入完成 (Phase 1 + 2 + 3)！")
}


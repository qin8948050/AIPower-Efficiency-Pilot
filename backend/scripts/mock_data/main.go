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

// 根据资源池获取对应的切片模式
func getSlicingModeForPool(poolID string, modes []types.SlicingMode) string {
	switch poolID {
	case "Train-H800-Full-Pool", "Train-A100-Full-Pool":
		return string(types.SlicingModeFull)
	case "Infer-A100-MIG-Pool":
		return string(types.SlicingModeMIG)
	case "Infer-L4-MPS-Pool":
		return string(types.SlicingModeMPS)
	case "Dev-T4-TS-Pool":
		return string(types.SlicingModeTS)
	default:
		return string(modes[0])
	}
}

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
		mysqlClient.RawExec("TRUNCATE TABLE life_trace")
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

	// 3.1 定义任务（Pod/PyTorchJob）与资源池的不匹配关系
	// 任务场景 (namespace + team + pod前缀) → 当前调度的资源池 → 合适的资源池 → 不匹配原因
	type TaskPoolMismatch struct {
		Namespace   string  // Pod 所在 namespace
		Team        string  // 负责团队
		PodPrefix   string  // Pod 名称前缀（代表某个任务/PyTorchJob）
		CurrentPool string  // 当前调度到的池（可能是错的）
		TargetPool  string  // 建议迁移到的池
		Reason      string  // 不匹配原因
		AvgUtil     float64 // 平均利用率
		Jitter      float64 // 抖动率（-1表示不适用）
	}
	taskMismatches := []TaskPoolMismatch{
		// 场景1: CV-Team 的研发测试任务被调度到了昂贵的训练池（应该用 TS 池）
		{Namespace: "ai-platform", Team: "CV-Team", PodPrefix: "pytorchjob-cv-train", CurrentPool: "Train-A100-Full-Pool", TargetPool: "Dev-T4-TS-Pool", Reason: "研发测试任务不需要NVLink，应使用低成本TS池", AvgUtil: 18.5, Jitter: -1},
		// 场景2: Search-Algo 的推理任务被调度到了TS池，抖动过大（应该用 MPS 池）
		{Namespace: "inference-prod", Team: "Search-Algo", PodPrefix: "pytorchjob-search-serving", CurrentPool: "Dev-T4-TS-Pool", TargetPool: "Infer-L4-MPS-Pool", Reason: "推理服务对延迟敏感，TS池抖动过大", AvgUtil: 45.0, Jitter: 22.5},
		// 场景3: NLP-Group 的训练任务使用了H800 NVLink但利用率很低（应该降级到A100）
		{Namespace: "data-science", Team: "NLP-Group", PodPrefix: "pytorchjob-nlp-sft", CurrentPool: "Train-H800-Full-Pool", TargetPool: "Train-A100-Full-Pool", Reason: "训练任务无需NVLink/FP8高端特性，利用率低造成浪费", AvgUtil: 12.0, Jitter: -1},
		// 场景4: Inference-prod 的批量推理任务被调度到了 MIG 池（应该用 MPS 池）
		{Namespace: "inference-prod", Team: "CV-Team", PodPrefix: "pytorchjob-cv-batch-infer", CurrentPool: "Infer-A100-MIG-Pool", TargetPool: "Infer-L4-MPS-Pool", Reason: "批量推理吞吐量优先，MPS更经济", AvgUtil: 35.0, Jitter: -1},
	}

	// 4. 建立节点池对应关系 (Redis)
	for i, node := range nodes {
		poolID := pools[i%len(pools)]
		redisClient.SaveNodePoolID(node, poolID)
	}

	// 5. 生成历史账单快照 (Daily Snapshots - 过去 7 天)
	// 按任务场景（namespace + team）维度聚合，体现任务与资源池的不匹配
	fmt.Println("生成过去 7 天的日级账单快照...")
	for i := 7; i >= 1; i-- {
		dateStr := time.Now().AddDate(0, 0, -i).Format("2006-01-02")

		// 5.1 先注入任务不匹配的数据（关键场景）
		for _, mismatch := range taskMismatches {
			avgUtil := mismatch.AvgUtil
			if mismatch.Jitter > 0 {
				// 抖动场景：最近3天抖动明显
				if i <= 3 {
					avgUtil = mismatch.AvgUtil
				} else {
					avgUtil = mismatch.AvgUtil - 10
				}
			}
			snapshot := &storage.DailyBillingSnapshot{
				SnapshotDate:    dateStr,
				PoolID:          mismatch.CurrentPool,
				Namespace:       mismatch.Namespace,
				TeamLabel:       mismatch.Team,
				TotalCost:       500 + r.Float64()*1500,
				AvgUtilP95:      avgUtil,
				MaxMemGiB:       8 + r.Float64()*20,
				PodSessionCount: 3 + r.Intn(8),
			}
			mysqlClient.UpsertDailySnapshot(snapshot)
		}

		// 5.2 注入正常匹配的任务数据（填充其他组合）
		for _, pool := range pools {
			for _, ns := range namespaces {
				// 跳过已经在 taskMismatches 中覆盖的组合
				covered := false
				for _, mismatch := range taskMismatches {
					if mismatch.CurrentPool == pool && mismatch.Namespace == ns {
						covered = true
						break
					}
				}
				if covered {
					continue
				}

				avgUtil := 30 + r.Float64()*50 // 正常利用率 30-80%
				snapshot := &storage.DailyBillingSnapshot{
					SnapshotDate:    dateStr,
					PoolID:          pool,
					Namespace:       ns,
					TeamLabel:       teams[r.Intn(len(teams))],
					TotalCost:       200 + r.Float64()*800,
					AvgUtilP95:      avgUtil,
					MaxMemGiB:       4 + r.Float64()*28,
					PodSessionCount: 5 + r.Intn(20),
				}
				mysqlClient.UpsertDailySnapshot(snapshot)
			}
		}
	}

	// 6. 生成历史会话记录 (Stitched LifeTrace)
	// 生成任务不匹配场景的 Pod 会话
	fmt.Println("生成已结算的会话明细...")

	// 6.1 先生成任务不匹配场景的 Pod（体现问题任务）
	for idx, mismatch := range taskMismatches {
		for podIdx := 1; podIdx <= 3; podIdx++ { // 每个任务场景生成 3 个 Pod
			startTime := time.Now().Add(-time.Duration(r.Intn(72)) * time.Hour)
			endTime := startTime.Add(time.Duration(60+r.Intn(240)) * time.Minute)

			avgUtil := mismatch.AvgUtil
			maxUtil := avgUtil
			if mismatch.Jitter > 0 {
				// 有抖动：max 明显高于 avg
				maxUtil = avgUtil * (1 + mismatch.Jitter/100)
			} else {
				maxUtil = avgUtil + 10 + r.Float64()*15
			}

			trace := &storage.LifeTrace{
				PodUID:        fmt.Sprintf("mismatch-%d-%d", idx, podIdx),
				Namespace:     mismatch.Namespace,
				PodName:       fmt.Sprintf("%s-worker-%d", mismatch.PodPrefix, podIdx),
				NodeName:      nodes[r.Intn(len(nodes))],
				PoolID:        mismatch.CurrentPool,
				SlicingMode:   getSlicingModeForPool(mismatch.CurrentPool, modes),
				StartTime:     startTime,
				EndTime:       &endTime,
				Status:        "Settled",
				TeamLabel:     mismatch.Team,
				GPUUtilAvg:    avgUtil,
				GPUUtilMax:    maxUtil,
				MemUsedMax:    uint64(2048 + r.Intn(30000)),
				PowerUsageAvg: 150 + r.Float64()*100,
				CostAmount:    20 + r.Float64()*80,
			}
			mysqlClient.SaveRawLifeTrace(trace)
		}
	}

	// 6.2 生成正常匹配的任务 Pod（填充数据）
	for i := 1; i <= 20; i++ {
		ns := namespaces[r.Intn(len(namespaces))]
		poolID := pools[r.Intn(len(pools))]
		team := teams[r.Intn(len(teams))]

		// 跳过已经在 taskMismatches 中覆盖的组合
		covered := false
		for _, mismatch := range taskMismatches {
			if mismatch.CurrentPool == poolID && mismatch.Namespace == ns {
				covered = true
				break
			}
		}
		if covered {
			continue
		}

		node := nodes[r.Intn(len(nodes))]
		startTime := time.Now().Add(-time.Duration(r.Intn(48)) * time.Hour)
		endTime := startTime.Add(time.Duration(30+r.Intn(300)) * time.Minute)

		// 正常利用率
		avgUtil := 30 + r.Float64()*50
		maxUtil := avgUtil + r.Float64()*20

		trace := &storage.LifeTrace{
			PodUID:        fmt.Sprintf("hist-uid-%d", i),
			Namespace:     ns,
			PodName:       fmt.Sprintf("normal-job-%03d", i),
			NodeName:      node,
			PoolID:        poolID,
			SlicingMode:   getSlicingModeForPool(poolID, modes),
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
	// 基于任务场景与资源池不匹配的数据生成诊断报告
	fmt.Println("生成 AI 诊断报告...")

	// 根据任务不匹配场景生成报告
	insightReports := []struct {
		TaskName   string  // 任务名
		Namespace  string  // 命名空间
		Team       string  // 负责团队
		PoolID     string  // 当前资源池
		ReportType string
		Summary    string
		RootCause  string
		Actions    string
		EstSavings float64
		Status     string
	}{
		// 场景1: CV-Team 研发测试任务 - 降级建议
		{
			TaskName:   "pytorchjob-cv-train",
			Namespace:  "ai-platform",
			Team:       "CV-Team",
			PoolID:     "Train-A100-Full-Pool",
			ReportType: "downgrade",
			Summary:    "任务 [pytorchjob-cv-train] 平均利用率仅 18.5%，被调度在昂贵的高端训练池。",
			RootCause:  "该任务无需 NVLink/FP8 特性，应迁移至 Dev-T4-TS-Pool 降低成本。",
			Actions:    `[{"type":"migrate","pod_name":"pytorchjob-cv-train-worker-1","namespace":"ai-platform","team":"CV-Team","from_pool":"Train-A100-Full-Pool","to_pool":"Dev-T4-TS-Pool"}]`,
			EstSavings: 8500.0,
			Status:     "pending",
		},
		// 场景2: Search-Algo 推理任务 - 稳定性升级建议（费用增加）
		{
			TaskName:   "pytorchjob-search-serving",
			Namespace:  "inference-prod",
			Team:       "Search-Algo",
			PoolID:     "Dev-T4-TS-Pool",
			ReportType: "isolation",
			Summary:    "任务 [pytorch 算力抖动job-search-serving]达 22.5%，影响服务质量。",
			RootCause:  "该任务对延迟敏感，TS 池资源共享导致抖动过大，建议迁移至 Infer-L4-MPS-Pool 以保障 SLA。",
			Actions:    `[{"type":"pool_change","pod_name":"pytorchjob-search-serving-worker-1","namespace":"inference-prod","team":"Search-Algo","from_pool":"Dev-T4-TS-Pool","to_pool":"Infer-L4-MPS-Pool"}]`,
			EstSavings: -2800.0, // 负数表示费用增加
			Status:     "pending",
		},
		// 场景3: NLP-Group 训练任务 - 特性纠偏建议
		{
			TaskName:   "pytorchjob-nlp-sft",
			Namespace:  "data-science",
			Team:       "NLP-Group",
			PoolID:     "Train-H800-Full-Pool",
			ReportType: "downgrade",
			Summary:    "任务 [pytorchjob-nlp-sft] 利用 NVLink/FP8 特性但利用率仅 12%，造成高端资源浪费。",
			RootCause:  "该任务无需 NVLink/FP8 高端特性，利用率低于 30% 阈值，建议迁移至 Train-A100-Full-Pool。",
			Actions:    `[{"type":"migrate","pod_name":"pytorchjob-nlp-sft-worker-1","namespace":"data-science","team":"NLP-Group","from_pool":"Train-H800-Full-Pool","to_pool":"Train-A100-Full-Pool"}]`,
			EstSavings: 15000.0,
			Status:     "pending",
		},
		// 场景4: CV-Team 批量推理任务 - 优化建议
		{
			TaskName:   "pytorchjob-cv-batch-infer",
			Namespace:  "inference-prod",
			Team:       "CV-Team",
			PoolID:     "Infer-A100-MIG-Pool",
			ReportType: "downgrade",
			Summary:    "任务 [pytorchjob-cv-batch-infer] 被调度至 MIG 池，吞吐量未充分发挥。",
			RootCause:  "批量推理对延迟不敏感，MIG 硬隔离单价较高，建议迁移至 Infer-L4-MPS-Pool。",
			Actions:    `[{"type":"migrate","pod_name":"pytorchjob-cv-batch-infer-worker-1","namespace":"inference-prod","team":"CV-Team","from_pool":"Infer-A100-MIG-Pool","to_pool":"Infer-L4-MPS-Pool"}]`,
			EstSavings: 4500.0,
			Status:     "pending",
		},
		// 场景5: 已完成的优化（历史报告）
		{
			TaskName:   "pytorchjob-historical",
			Namespace:  "ai-platform",
			Team:       "NLP-Group",
			PoolID:     "Train-A100-Full-Pool",
			ReportType: "downgrade",
			Summary:    "上一周期的降级建议已执行，资源利用率从 25% 提升至 55%。",
			RootCause:  "通过将低利用率训练任务迁移至低成本池，释放了高端 GPU 资源。",
			Actions:    "[]",
			EstSavings: 9200.0,
			Status:     "approved",
		},
	}

	reportTime := time.Now().AddDate(0, 0, -1)
	for i, r := range insightReports {
		generatedAt := reportTime.Add(-time.Duration(i*12) * time.Hour)
		report := &storage.InsightReport{
			GeneratedAt: generatedAt,
			TaskName:    r.TaskName,
			Namespace:   r.Namespace,
			Team:        r.Team,
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


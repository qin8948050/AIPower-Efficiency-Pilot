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
	mysqlClient.RawExec("TRUNCATE TABLE governance_executions")
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
		TaskName                  string  // 任务名
		Namespace                 string  // 命名空间
		Team                      string  // 负责团队
		PoolID                    string  // 当前资源池
		Problem                   string  // 问题描述: 利用率低, 抖动高, 特性不匹配
		ReportType                string
		Summary                   string
		RootCause                 string
		Recommendations           string
		ApprovedRecommendation   string  // 批准的建议（JSON格式）
		EstSavings                float64
		Status                    string
	}{
		// 场景1: 仅1种建议 - 简单降配场景
		// 问题：利用率低，优先级低，直接在当前池降配即可
		{
			TaskName:   "pytorchjob-cv-dev",
			Namespace:  "ai-platform",
			Team:       "CV-Team",
			PoolID:     "Dev-T4-TS-Pool",
			Problem:    "利用率低",
			ReportType: "downgrade",
			Summary:    "任务 [pytorchjob-cv-dev] 平均利用率仅 15%，属于开发测试任务。",
			RootCause:  "开发测试任务对性能要求不高，可在当前池降配 GPU 数量以节省成本。",
			// 仅1种建议：在当前池降配
			Recommendations: `[{"action_type":"降配","from_gpu":4,"to_gpu":1,"from_pool":"Dev-T4-TS-Pool","to_pool":"Dev-T4-TS-Pool","est_savings":3200,"reason":"开发任务资源需求低，直接降配"}]`,
			EstSavings: 3200.0,
			Status:     "pending",
		},
		// 场景2: 2种建议 - 迁移 vs 降配+迁移
		// 问题：利用率低，可选择保留当前池降配，或迁移到低成本池
		{
			TaskName:   "pytorchjob-cv-train",
			Namespace:  "ai-platform",
			Team:       "CV-Team",
			PoolID:     "Train-A100-Full-Pool",
			Problem:    "利用率低",
			ReportType: "downgrade",
			Summary:    "任务 [pytorchjob-cv-train] 平均利用率仅 18.5%，被调度在昂贵的高端训练池。",
			RootCause:  "该任务无需 NVLink/FP8 特性，可选择降配或迁移至低成本池。",
			// 2种建议：
			// 1. 降配：保留当前池，降配 GPU 数量
			// 2. 降配+迁移：降配同时迁移到低成本池（节省更多）
			Recommendations: `[
				{"action_type":"降配","from_gpu":8,"to_gpu":4,"from_pool":"Train-A100-Full-Pool","to_pool":"Train-A100-Full-Pool","est_savings":8500,"reason":"保留当前池特性，简单降配"},
				{"action_type":"降配+迁移","from_gpu":8,"to_gpu":4,"from_pool":"Train-A100-Full-Pool","to_pool":"Dev-T4-TS-Pool","est_savings":12500,"reason":"降配并迁移到低成本池，节省更多"}
			]`,
			EstSavings: 12500.0, // 默认选节省最多的
			Status:     "pending",
		},
		// 场景3: 3种建议 - 降配、迁移、降配+迁移
		// 问题：利用率低，有多种优化路径可选
		{
			TaskName:   "pytorchjob-nlp-training",
			Namespace:  "data-science",
			Team:       "NLP-Group",
			PoolID:     "Train-H800-Full-Pool",
			Problem:    "利用率低",
			ReportType: "downgrade",
			Summary:    "任务 [pytorchjob-nlp-training] 利用 NVLink/FP8 特性但利用率仅 12%，造成高端资源浪费。",
			RootCause:  "该任务无需高端特性，可选择多种优化方案。",
			// 3种建议：
			// 1. 降配：保留当前 H800 池，降配 GPU
			// 2. 迁移：不降配，迁移到 A100 池（保留算力）
			// 3. 降配+迁移：降配并迁移到 A100 池（节省最多）
			Recommendations: `[
				{"action_type":"降配","from_gpu":8,"to_gpu":4,"from_pool":"Train-H800-Full-Pool","to_pool":"Train-H800-Full-Pool","est_savings":15000,"reason":"保留NVLink+FP8高端特性，仅降配GPU数量"},
				{"action_type":"迁移","from_gpu":8,"to_gpu":8,"from_pool":"Train-H800-Full-Pool","to_pool":"Train-A100-Full-Pool","est_savings":8000,"reason":"迁移到A100池，保留8卡算力，降低单卡成本"},
				{"action_type":"降配+迁移","from_gpu":8,"to_gpu":4,"from_pool":"Train-H800-Full-Pool","to_pool":"Train-A100-Full-Pool","est_savings":22000,"reason":"降配并迁移到A100池，大幅降低成本"}
			]`,
			EstSavings: 22000.0,
			Status:     "pending",
		},
		// 场景4: 稳定性问题 - 抖动高（费用增加场景）
		{
			TaskName:   "pytorchjob-search-serving",
			Namespace:  "inference-prod",
			Team:       "Search-Algo",
			PoolID:     "Dev-T4-TS-Pool",
			Problem:    "抖动高",
			ReportType: "isolation",
			Summary:    "任务 [pytorchjob-search-serving] 算力抖动达 22.5%，影响服务质量。",
			RootCause:  "该任务对延迟敏感，TS 池资源共享导致抖动过大。",
			// 2种建议（解决抖动，费用会增加）：
			// 1. 迁移到 MIG 池：硬隔离，解决抖动
			// 2. 迁移到 MPS 池：中等隔离，平衡成本
			Recommendations: `[
				{"action_type":"迁移","from_gpu":8,"to_gpu":8,"from_pool":"Dev-T4-TS-Pool","to_pool":"Infer-A100-MIG-Pool","est_savings":-2800,"reason":"MIG硬隔离，彻底解决抖动问题"},
				{"action_type":"迁移","from_gpu":8,"to_gpu":8,"from_pool":"Dev-T4-TS-Pool","to_pool":"Infer-L4-MPS-Pool","est_savings":-1200,"reason":"MPS中等隔离，平衡成本与稳定性"}
			]`,
			EstSavings: -1200.0, // 默认选增加较少的
			Status:     "pending",
		},
		// 场景5: 已批准的优化（历史报告）
		{
			TaskName:   "pytorchjob-historical",
			Namespace:  "ai-platform",
			Team:       "NLP-Group",
			PoolID:     "Train-A100-Full-Pool",
			Problem:    "利用率低",
			ReportType: "downgrade",
			Summary:    "上一周期的降级建议已执行，资源利用率从 25% 提升至 55%。",
			RootCause:  "通过将低利用率训练任务迁移至低成本池，释放了高端 GPU 资源。",
			Recommendations: `[
				{"action_type":"降配","from_gpu":8,"to_gpu":4,"from_pool":"Train-A100-Full-Pool","to_pool":"Train-A100-Full-Pool","est_savings":8500,"reason":"保留当前池特性"},
				{"action_type":"降配+迁移","from_gpu":8,"to_gpu":4,"from_pool":"Train-A100-Full-Pool","to_pool":"Dev-T4-TS-Pool","est_savings":12500,"reason":"降配并迁移到低成本池"}
			]`,
			ApprovedRecommendation: `{"action_type":"降配+迁移","from_gpu":8,"to_gpu":4,"from_pool":"Train-A100-Full-Pool","to_pool":"Dev-T4-TS-Pool","est_savings":12500,"reason":"降配并迁移到低成本池"}`,
			EstSavings: 12500.0,
			Status:     "approved",
		},
	}

	reportTime := time.Now().AddDate(0, 0, -1)
	for i, r := range insightReports {
		generatedAt := reportTime.Add(-time.Duration(i*12) * time.Hour)
		report := &storage.InsightReport{
			GeneratedAt:              generatedAt,
			TaskName:                 r.TaskName,
			Namespace:                r.Namespace,
			Team:                    r.Team,
			PoolID:                  r.PoolID,
			Problem:                 r.Problem,
			ReportType:              r.ReportType,
			Summary:                 r.Summary,
			RootCause:               r.RootCause,
			Recommendations:         r.Recommendations,
			ApprovedRecommendation:  r.ApprovedRecommendation,
			EstSavings:              r.EstSavings,
			Status:                  r.Status,
		}
		mysqlClient.SaveInsightReport(report)
	}
	fmt.Println("Created insight reports")

	// 9. 生成治理执行记录 (Phase 4)
	fmt.Println("生成治理执行记录...")

	executions := []struct {
		ReportID     uint
		TaskName     string
		Namespace    string
		ActionType   string
		FromPool     string
		ToPool       string
		FromGPU      int
		ToGPU        int
		PatchType    string
		PatchContent string
		Status       string
	}{
		// 已完成
		{
			ReportID:     1,
			TaskName:     "pytorchjob-historical",
			Namespace:    "ai-platform",
			ActionType:   "downgrade_migrate",
			FromPool:     "Train-A100-Full-Pool",
			ToPool:       "Dev-T4-TS-Pool",
			FromGPU:      8,
			ToGPU:        4,
			PatchType:    "strategic-merge-patch",
			PatchContent: `{"spec":{"containers":[{"name":"*","resources":{"limits":{"nvidia.com/gpu":"4"}}},"metadata":{"labels":{"governance.migration/pending":"true","governance.migration/to-pool":"Dev-T4-TS-Pool"}}]}`,
			Status:       "completed",
		},
		// 执行中
		{
			ReportID:     2,
			TaskName:     "pytorchjob-cv-train",
			Namespace:    "ai-platform",
			ActionType:   "downgrade_migrate",
			FromPool:     "Train-A100-Full-Pool",
			ToPool:       "Dev-T4-TS-Pool",
			FromGPU:      8,
			ToGPU:        4,
			PatchType:    "strategic-merge-patch",
			PatchContent: `{"spec":{"containers":[{"name":"*","resources":{"limits":{"nvidia.com/gpu":"4"}}},"metadata":{"labels":{"governance.migration/pending":"true","governance.migration/to-pool":"Dev-T4-TS-Pool"}}]}`,
			Status:       "executing",
		},
		// 待执行
		{
			ReportID:     3,
			TaskName:     "pytorchjob-nlp-training",
			Namespace:    "data-science",
			ActionType:   "downgrade",
			FromPool:     "Train-H800-Full-Pool",
			ToPool:       "Train-H800-Full-Pool",
			FromGPU:      8,
			ToGPU:        4,
			PatchType:    "strategic-merge-patch",
			PatchContent: `{"spec":{"containers":[{"name":"*","resources":{"limits":{"nvidia.com/gpu":"4"}}}]}}`,
			Status:       "pending",
		},
		// 失败
		{
			ReportID:     4,
			TaskName:     "pytorchjob-search-serving",
			Namespace:    "inference-prod",
			ActionType:   "migrate",
			FromPool:     "Dev-T4-TS-Pool",
			ToPool:       "Infer-L4-MPS-Pool",
			FromGPU:      8,
			ToGPU:        8,
			PatchType:    "strategic-merge-patch",
			PatchContent: `{"metadata":{"labels":{"governance.migration/pending":"true","governance.migration/to-pool":"Infer-L4-MPS-Pool"}}}`,
			Status:       "failed",
		},
		// 待执行
		{
			ReportID:     5,
			TaskName:     "pytorchjob-cv-dev",
			Namespace:    "ai-platform",
			ActionType:   "downgrade",
			FromPool:     "Dev-T4-TS-Pool",
			ToPool:       "Dev-T4-TS-Pool",
			FromGPU:      4,
			ToGPU:        1,
			PatchType:    "strategic-merge-patch",
			PatchContent: `{"spec":{"containers":[{"name":"*","resources":{"limits":{"nvidia.com/gpu":"1"}}}]}}`,
			Status:       "pending",
		},
		// 已取消
		{
			ReportID:     6,
			TaskName:     "pytorchjob-cancelled",
			Namespace:    "data-science",
			ActionType:   "migrate",
			FromPool:     "Train-A100-Full-Pool",
			ToPool:       "Dev-T4-TS-Pool",
			FromGPU:      8,
			ToGPU:        8,
			PatchType:    "strategic-merge-patch",
			PatchContent: `{"metadata":{"labels":{"governance.migration/pending":"true","governance.migration/to-pool":"Dev-T4-TS-Pool"}}}`,
			Status:       "cancelled",
		},
	}

	for i, e := range executions {
		exec := &storage.GovernanceExecution{
			ReportID:     e.ReportID,
			TaskName:     e.TaskName,
			Namespace:    e.Namespace,
			ActionType:   e.ActionType,
			FromPool:     e.FromPool,
			ToPool:       e.ToPool,
			FromGPU:      e.FromGPU,
			ToGPU:        e.ToGPU,
			PatchType:    e.PatchType,
			PatchContent: e.PatchContent,
			Status:       e.Status,
		}
		mysqlClient.SaveGovernanceExecution(exec)

		// 为已完成和失败的添加执行时间
		if e.Status == "completed" || e.Status == "failed" {
			execTime := time.Now().Add(-time.Duration(i+1) * time.Hour)
			mysqlClient.RawExec(fmt.Sprintf("UPDATE governance_executions SET executed_at = '%s' WHERE id = %d",
				execTime.Format("2006-01-02 15:04:05"), exec.ID))
		}
		if e.Status == "failed" {
			mysqlClient.RawExec(fmt.Sprintf("UPDATE governance_executions SET error_msg = 'Pod not found or insufficient permissions' WHERE id = %d", exec.ID))
		}
	}
	fmt.Println("Created governance executions")

	fmt.Println("\n全量 Mock 数据注入完成 (Phase 1 + 2 + 3 + 4)！")
}


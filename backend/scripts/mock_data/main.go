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
	redisClient.FlushDB()

	// 2.6 注入定价配置 (Phase 2 核心)
	fmt.Println("注入资源池定价配置...")
	pricingList := []storage.PoolPricing{
		{PoolID: "pool-v100-shared", GPUModel: "NVIDIA V100", BasePricePerHour: 28.0, SlicingWeightFull: 1.0, SlicingWeightMIG: 0.35, SlicingWeightMPS: 0.5, SlicingWeightTS: 0.6},
		{PoolID: "pool-a100-priority", GPUModel: "NVIDIA A100", BasePricePerHour: 55.0, SlicingWeightFull: 1.0, SlicingWeightMIG: 0.4, SlicingWeightMPS: 0.6, SlicingWeightTS: 0.7},
		{PoolID: "pool-t4-lowcost", GPUModel: "NVIDIA T4", BasePricePerHour: 12.0, SlicingWeightFull: 1.0, SlicingWeightMIG: 0.3, SlicingWeightMPS: 0.4, SlicingWeightTS: 0.5},
	}
	for _, p := range pricingList {
		mysqlClient.SavePoolPricing(&p)
	}

	// 3. 定义模拟数据
	fmt.Println("开始注入新的模拟数据...")
	namespaces := []string{"default", "ai-platform", "data-science", "inference-prod"}
	teams := []string{"CV-Team", "NLP-Group", "Search-Algo", "Infrastructure"}
	nodes := []string{"gpu-node-01", "gpu-node-02", "gpu-node-03"}
	pools := []string{"pool-v100-shared", "pool-a100-priority", "pool-t4-lowcost"}
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
				snapshot := &storage.DailyBillingSnapshot{
					SnapshotDate:    dateStr,
					PoolID:          pool,
					Namespace:       ns,
					TeamLabel:       teams[r.Intn(len(teams))],
					TotalCost:       200 + r.Float64()*1000,
					AvgUtilP95:      10 + r.Float64()*80,
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
		
		trace := &storage.LifeTrace{
			PodUID:        fmt.Sprintf("hist-uid-%d", i),
			Namespace:     ns,
			PodName:       fmt.Sprintf("offline-job-%03d", i),
			NodeName:      node,
			PoolID:        poolID,
			SlicingMode:   string(modes[r.Intn(len(modes))]),
			StartTime:     startTime,
			EndTime:       &endTime,
			TeamLabel:     team,
			GPUUtilAvg:    10 + r.Float64()*80,
			GPUUtilMax:    90 + r.Float64()*10,
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

	fmt.Println("\n二阶段 Mock 数据注入完成！")
}


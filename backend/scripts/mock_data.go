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
	fmt.Println("正在清理 MySQL (life_trace) 和 Redis...")
	if err := mysqlClient.TruncateTable(); err != nil {
		log.Printf("清理 MySQL 失败: %v", err)
	}
	if err := redisClient.FlushDB(); err != nil {
		log.Printf("清理 Redis 失败: %v", err)
	}

	// 3. 定义模拟数据
	fmt.Println("开始注入新的模拟数据...")
	namespaces := []string{"default", "ai-platform", "data-science", "inference-prod"}
	nodes := []string{"gpu-node-01", "gpu-node-02", "gpu-node-03"}
	pools := []string{"pool-v100-shared", "pool-a100-priority", "pool-t4-lowcost"}
	modes := []types.SlicingMode{types.SlicingModeMIG, types.SlicingModeMPS, types.SlicingModeTS, types.SlicingModeFull}

	rand.Seed(time.Now().UnixNano())

	// 4. 建立节点池对应关系 (Redis)
	for i, node := range nodes {
		poolID := pools[i%len(pools)]
		if err := redisClient.SaveNodePoolID(node, poolID); err != nil {
			log.Printf("保存节点池映射失败 [%s]: %v", node, err)
		} else {
			fmt.Printf("节点 %s -> 池 %s (已保存)\n", node, poolID)
		}
	}

	// 5. 生成 Pod 数据
	for i := 1; i <= 10; i++ {
		ns := namespaces[rand.Intn(len(namespaces))]
		node := nodes[rand.Intn(len(nodes))]
		poolID, _ := redisClient.GetNodePoolID(node)
		podName := fmt.Sprintf("ai-task-%03d", i)
		podUID := fmt.Sprintf("uid-%d-%d", time.Now().Unix(), i)
		mode := modes[rand.Intn(len(modes))]

		trace := &types.PodTrace{
			Namespace:   ns,
			PodName:     podName,
			PodUID:      podUID,
			NodeName:    node,
			PoolID:      poolID,
			SlicingMode: mode,
			StartTime:   time.Now().Add(-time.Duration(rand.Intn(24)) * time.Hour),
			Metrics: &types.GPUMetrics{
				GPUUtilAvg:    rand.Float64() * 100,
				GPUUtilMax:    rand.Float64() * 100,
				MemUsedBytes:  uint64(rand.Int63n(80 * 1024 * 1024 * 1024)), // 0-80GB
				MemTotalBytes: 80 * 1024 * 1024 * 1024,
				PowerUsageW:   100 + rand.Float64()*300,
				LastUpdate:    time.Now(),
			},
		}

		// 写入 MySQL (生命周期追踪)
		if err := mysqlClient.SaveLifeTrace(trace); err != nil {
			log.Printf("保存 MySQL 生命周期记录失败 [%s]: %v", podName, err)
		}

		// 写入 Redis (实时快照)
		if err := redisClient.SavePodTrace(trace); err != nil {
			log.Printf("保存 Redis 实时快照失败 [%s]: %v", podName, err)
		}

		fmt.Printf("已生成 Pod: %s/%s (节点: %s, 模式: %s)\n", ns, podName, node, mode)
	}

	fmt.Println("\n模拟数据注入完成！")
}

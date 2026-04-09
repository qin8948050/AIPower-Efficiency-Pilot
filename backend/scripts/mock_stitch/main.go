package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/qxw/aipower-efficiency-pilot/internal/config"
	"github.com/qxw/aipower-efficiency-pilot/internal/storage"
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

	fmt.Println("=== 步骤四：离线指标缝合 (模拟测试) ===")

	// 3. 查找待缝合的记录
	fmt.Println("1. 正在扫描待缝合记录 (end_time != null AND gpu_util_avg = 0)...")
	traces, err := mysqlClient.GetPendingMetricsTraces(10)
	if err != nil {
		log.Printf("扫描失败: %v", err)
	}

	if len(traces) == 0 {
		// 兜底：创建一条用于演示的关闭记录
		fmt.Println("   未找到待缝合记录，先模拟关闭一条 Pod 记录...")
		_ = mysqlClient.CloseLifeTrace("default", "ai-task-001", time.Now())
		traces, _ = mysqlClient.GetPendingMetricsTraces(10)
	}

	fmt.Printf("   找到 %d 条待缝合记录\n\n", len(traces))

	// 4. 对每条记录执行缝合
	rand.Seed(time.Now().UnixNano())
	for _, trace := range traces {
		startTime := trace.StartTime
		var endTime time.Time
		if trace.EndTime != nil {
			endTime = *trace.EndTime
		} else {
			endTime = time.Now()
		}

		fmt.Printf("2. 处理 Pod: %s/%s (窗口: %s -> %s)\n",
			trace.Namespace, trace.PodName,
			startTime.Format("15:04:05"), endTime.Format("15:04:05"))

		// 5. 模拟 Prometheus range_query
		fmt.Println("   模拟向 Prometheus 发起 range_query...")
		sampleCount := 10
		var sumUtil float64
		var maxUtil float64
		maxMem := uint64(40 * 1024 * 1024 * 1024) // 模拟 40GiB
		avgPower := 250.0

		for i := 0; i < sampleCount; i++ {
			val := 20 + rand.Float64()*70
			sumUtil += val
			if val > maxUtil {
				maxUtil = val
			}
		}
		avgUtil := sumUtil / float64(sampleCount)

		// 6. 模拟计费 (按小时 × 单价 × 权重)
		durationH := endTime.Sub(startTime).Hours()
		if durationH < 0.01 {
			durationH = 0.1 // 最小计费单元：6分钟
		}
		basePricePerHour := 30.0 // 模拟 V100 单价 ¥30/h
		slicingWeight := 1.0     // 模拟 Full 模式权重
		cost := durationH * basePricePerHour * slicingWeight

		// 7. 持久化回填
		fmt.Printf("   avgUtil=%.2f%% maxUtil=%.2f%% maxMem=40GiB avgPower=%.1fW cost=¥%.4f\n",
			avgUtil, maxUtil, avgPower, cost)

		if err := mysqlClient.UpdateLifeTraceMetrics(trace.ID, avgUtil, maxUtil, maxMem, avgPower, cost); err != nil {
			log.Printf("   持久化失败: %v", err)
		} else {
			fmt.Println("   ✅ 指标与费用已回填至 MySQL！")
		}

		fmt.Printf("   评估结论: %s\n\n", evaluateROI(avgUtil))
	}

	fmt.Println("=== 缝合完成 ===")
}

func evaluateROI(util float64) string {
	if util < 30 {
		return "⚠️  资源浪费 (利用率极低)，建议缩减规格或迁移至共享池"
	} else if util > 80 {
		return "✅ 高效运行 (资源利用充分)"
	}
	return "🆗 运行正常"
}

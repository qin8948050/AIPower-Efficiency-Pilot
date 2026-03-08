package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/qxw/aipower-efficiency-pilot/internal/config"
	"github.com/qxw/aipower-efficiency-pilot/internal/storage"
)

var (
	configFile string
)

func init() {
	flag.StringVar(&configFile, "config", "", "Path to configuration file")
	flag.Parse()
}

func main() {
	log.Println("Starting AIPower-Efficiency-Pilot API Server...")

	// 0. 加载配置
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 1. 初始化 Redis
	redisCli, err := storage.NewRedisClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}

	// 2. 初始化 MySQL
	mysqlCli, err := storage.NewMySQLClient(cfg.MySQL.DSN)
	if err != nil {
		log.Fatalf("Failed to initialize MySQL: %v", err)
	}

	// 3. 设置路由
	r := gin.Default()

	// 允许跨域 (简单版)
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Next()
	})

	v1 := r.Group("/api/v1")
	{
		// 获取所有活跃资源池的统计信息
		v1.GET("/pools", func(c *gin.Context) {
			traces, err := redisCli.GetAllPodTraces()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 按 PoolID 聚合数据
			poolStats := make(map[string]map[string]interface{})
			for _, t := range traces {
				if _, ok := poolStats[t.PoolID]; !ok {
					poolStats[t.PoolID] = map[string]interface{}{
						"id":             t.PoolID,
						"slicing_mode":   t.SlicingMode,
						"pod_count":      0,
						"gpu_util_avg":   0.0,
						"mem_used_bytes": uint64(0),
					}
				}
				stats := poolStats[t.PoolID]
				stats["pod_count"] = stats["pod_count"].(int) + 1
				if t.Metrics != nil {
					// 简单加权平均 (此处可根据需要优化)
					currAvg := stats["gpu_util_avg"].(float64)
					stats["gpu_util_avg"] = currAvg + t.Metrics.GPUUtilAvg
					stats["mem_used_bytes"] = stats["mem_used_bytes"].(uint64) + t.Metrics.MemUsedBytes
				}
			}

			// 转换为数组返回
			var result []interface{}
			for _, v := range poolStats {
				count := v["pod_count"].(int)
				if count > 0 {
					v["gpu_util_avg"] = v["gpu_util_avg"].(float64) / float64(count)
				}
				result = append(result, v)
			}

			c.JSON(http.StatusOK, result)
		})

		// 获取所有活跃 Pod 的列表
		v1.GET("/traces", func(c *gin.Context) {
			traces, err := redisCli.GetAllPodTraces()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, traces)
		})

		// 获取单个 Pod 的实时追踪与指标
		v1.GET("/traces/:namespace/:pod", func(c *gin.Context) {
			ns := c.Param("namespace")
			podName := c.Param("pod")

			trace, err := redisCli.GetPodTrace(ns, podName)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if trace == nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "trace not found"})
				return
			}

			c.JSON(http.StatusOK, trace)
		})

		// 获取 Pod 的历史审计记录 (Life-Trace)
		v1.GET("/traces/:namespace/:pod/history", func(c *gin.Context) {
			ns := c.Param("namespace")
			podName := c.Param("pod")

			// 使用 mysqlCli 查询历史 (此处简单演示从 MySQL 读取逻辑)
			// 实际项目中可以在 internal/storage/mysql.go 中增加更丰富的查询接口
			trace, err := mysqlCli.GetActivePodTrace(ns, podName)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "no active trace or history found"})
				return
			}
			c.JSON(http.StatusOK, trace)
		})

		// 健康检查
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}

	log.Printf("API Server listening on %s", cfg.Server.Addr)
	if err := r.Run(cfg.Server.Addr); err != nil {
		log.Fatalf("Failed to run API server: %v", err)
	}
}

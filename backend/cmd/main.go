package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/qxw/aipower-efficiency-pilot/internal/storage"
)

var (
	addr      string
	redisAddr string
	mysqlDSN  string
)

func init() {
	flag.StringVar(&addr, "addr", ":8080", "API server listen address")
	flag.StringVar(&redisAddr, "redis-addr", "localhost:6379", "Redis server address")
	flag.StringVar(&mysqlDSN, "mysql-dsn", "root:password@tcp(localhost:3306)/aipower?parseTime=true", "MySQL DSN")
	flag.Parse()
}

func main() {
	log.Println("Starting AIPower-Efficiency-Pilot API Server...")

	// 1. 初始化 Redis
	redisCli, err := storage.NewRedisClient(redisAddr, "", 0)
	if err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}

	// 2. 初始化 MySQL
	mysqlCli, err := storage.NewMySQLClient(mysqlDSN)
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

	log.Printf("API Server listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to run API server: %v", err)
	}
}

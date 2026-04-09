package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/qxw/aipower-efficiency-pilot/internal/aggregator"
	"github.com/qxw/aipower-efficiency-pilot/internal/config"
	"github.com/qxw/aipower-efficiency-pilot/internal/llm"
	"github.com/qxw/aipower-efficiency-pilot/internal/pilot"
	"github.com/qxw/aipower-efficiency-pilot/internal/storage"
	"time"
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

	// 3. 启动后台任务 (Stitcher & Daily Aggregation)
	go func() {
		log.Println("[worker] Background tasks started")

		// Stitcher: 每 10 分钟执行一次
		stitchTicker := time.NewTicker(10 * time.Minute)
		defer stitchTicker.Stop()

		// Daily Aggregation: 每天 01:00 执行
		// 计算距离下次 1 点的等待时间
		now := time.Now()
		next1AM := time.Date(now.Year(), now.Month(), now.Day(), 1, 0, 0, 0, now.Location())
		if now.Hour() >= 1 {
			next1AM = next1AM.Add(24 * time.Hour)
		}
		firstWait := next1AM.Sub(now)
		log.Printf("[worker] First daily aggregation at %s (in %v)", next1AM.Format("15:04:05"), firstWait)

		// 使用 Timer 等待第一次执行
		dailyTimer := time.NewTimer(firstWait)
		// 之后每 24 小时执行一次
		dailyTicker := time.NewTicker(24 * time.Hour)
		defer dailyTimer.Stop()
		defer dailyTicker.Stop()

		for {
			select {
			case <-stitchTicker.C:
				log.Println("[worker] Running metrics stitcher...")
				if err := aggregator.RunStitcher(mysqlCli, 50); err != nil {
					log.Printf("[worker] Stitcher error: %v", err)
				}
			case <-dailyTimer.C:
				log.Println("[worker] Running daily aggregation...")
				yesterday := time.Now().Add(-24 * time.Hour)
				if err := aggregator.RunDailyAggregation(mysqlCli, yesterday); err != nil {
					log.Printf("[worker] Daily aggregation error: %v", err)
				}
				// 切换到 Ticker 模式
				log.Println("[worker] Daily aggregation completed, next run in 24h")
			case <-dailyTicker.C:
				log.Println("[worker] Running daily aggregation...")
				yesterday := time.Now().Add(-24 * time.Hour)
				if err := aggregator.RunDailyAggregation(mysqlCli, yesterday); err != nil {
					log.Printf("[worker] Daily aggregation error: %v", err)
				}
			}
		}
	}()

	// 4. 设置路由
	r := gin.Default()

	// 允许跨域 (简单版)
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
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

	// Phase 2: Billing & Consolidation API
	v2 := r.Group("/api/v2")
	{
		// 日级账单汇总
		v2.GET("/billing/daily", func(c *gin.Context) {
			date := c.Query("date")
			// 若未指定日期，则返回所有历史聚合数据 (用于趋势图)
			snapshots, err := mysqlCli.GetDailySnapshots(date)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, snapshots)
		})

		// 账单明细查询 (Pod Session 粒度)
		v2.GET("/billing/sessions", func(c *gin.Context) {
			date := c.Query("date")
			if date == "" {
				date = time.Now().Add(-24 * time.Hour).Format("2006-01-02")
			}
			poolID := c.Query("pool_id")
			ns := c.Query("namespace")
			team := c.Query("team_label")

			sessions, err := mysqlCli.GetBillingSessions(date, poolID, ns, team)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, sessions)
		})

		// 获取定价配置
		v2.GET("/pricing", func(c *gin.Context) {
			pricing, err := mysqlCli.GetAllPoolPricing()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, pricing)
		})

		// 更新定价配置
		v2.PUT("/pricing/:pool_id", func(c *gin.Context) {
			poolID := c.Param("pool_id")
			var p storage.PoolPricing
			if err := c.ShouldBindJSON(&p); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			p.PoolID = poolID
			if err := mysqlCli.SavePoolPricing(&p); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "updated"})
		})

		// --- 资源池资产管理 ---

		// 获取所有已感知的资源池资产
		v2.GET("/pools", func(c *gin.Context) {
			pools, err := mysqlCli.GetAllResourcePools()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, pools)
		})

		// 更新资源池业务元数据
		v2.PUT("/pools/:id", func(c *gin.Context) {
			id := c.Param("id")
			var p storage.ResourcePool
			if err := c.ShouldBindJSON(&p); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			p.ID = id
			if err := mysqlCli.UpdateResourcePoolMetadata(&p); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "metadata updated"})
		})
	}

	// Phase 3: LLM Insights API
	insightsAnalyzer := llm.NewAnalyzer(mysqlCli, &cfg.LLM)

	v3 := r.Group("/api/v3")
	{
		// 触发 AI 诊断分析
		v3.POST("/insights/generate", func(c *gin.Context) {
			var req llm.GenerateRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				req.Days = 7 // 默认 7 天
			}

			report, err := insightsAnalyzer.GenerateReport(req.Days)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, report)
		})

		// 获取报告列表（按任务/团队过滤）
		v3.GET("/insights/reports", func(c *gin.Context) {
			taskName := c.Query("task_name")
			team := c.Query("team")
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
			offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

			reports, total, err := mysqlCli.GetInsightReports(taskName, team, limit, offset)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 转换为 API 响应格式（核心对象是任务）
			var reportList []llm.InsightReport
			for _, r := range reports {
				reportList = append(reportList, llm.InsightReport{
					ID:                       strconv.FormatUint(uint64(r.ID), 10),
					GeneratedAt:              r.GeneratedAt,
					TaskName:                 r.TaskName,
					Namespace:                r.Namespace,
					Team:                     r.Team,
					PoolID:                   r.PoolID,
					Problem:                   r.Problem,
					ReportType:               r.ReportType,
					Summary:                   r.Summary,
					RootCause:                 r.RootCause,
					Recommendations:           r.Recommendations,
					ApprovedRecommendation:    r.ApprovedRecommendation,
					ApprovedAt:                r.ApprovedAt,
					EstSavings:               r.EstSavings,
					Status:                   r.Status,
				})
			}

			c.JSON(http.StatusOK, llm.ReportListResponse{
				Reports: reportList,
				Total:   total,
			})
		})

		// 获取报告详情
		v3.GET("/insights/reports/:id", func(c *gin.Context) {
			id, err := strconv.ParseUint(c.Param("id"), 10, 32)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
				return
			}

			report, err := mysqlCli.GetInsightReportByID(uint(id))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
				return
			}

			c.JSON(http.StatusOK, llm.InsightReport{
				ID:                       strconv.FormatUint(uint64(report.ID), 10),
				GeneratedAt:              report.GeneratedAt,
				TaskName:                 report.TaskName,
				Namespace:                report.Namespace,
				Team:                     report.Team,
				PoolID:                   report.PoolID,
				Problem:                   report.Problem,
				ReportType:               report.ReportType,
				Summary:                   report.Summary,
				RootCause:                 report.RootCause,
				Recommendations:           report.Recommendations,
				ApprovedRecommendation:    report.ApprovedRecommendation,
				ApprovedAt:                report.ApprovedAt,
				EstSavings:               report.EstSavings,
				Status:                   report.Status,
			})
		})

		// 更新报告状态（审批流）
		v3.PUT("/insights/reports/:id/status", func(c *gin.Context) {
			id, err := strconv.ParseUint(c.Param("id"), 10, 32)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
				return
			}

			var req struct {
				Status        string `json:"status"`
				Recommendation string `json:"recommendation,omitempty"` // 批准时选中的建议（JSON字符串）
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			if req.Status != "approved" && req.Status != "rejected" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
				return
			}

			// 批准时必须选中一条建议
			if req.Status == "approved" && req.Recommendation == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "must select a recommendation when approving"})
				return
			}

			if err := mysqlCli.UpdateInsightReportStatus(uint(id), req.Status); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 如果是批准，更新选中的建议
			if req.Status == "approved" && req.Recommendation != "" {
				if err := mysqlCli.UpdateInsightReportRecommendation(uint(id), req.Recommendation); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}

			c.JSON(http.StatusOK, gin.H{"status": "updated"})
		})
	}

	// Phase 4: Governance Executor API
	governanceExecutor, err := pilot.NewGovernanceExecutor(cfg.K8s.Kubeconfig, mysqlCli)
	if err != nil {
		log.Fatalf("Failed to initialize governance executor: %v", err)
	}

	v4 := r.Group("/api/v4")
	{
		// 执行治理
		v4.POST("/governance/execute", func(c *gin.Context) {
			var req pilot.ExecuteRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			resp, err := governanceExecutor.Execute(c.Request.Context(), req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, resp)
		})

		// 查询执行列表
		v4.GET("/governance/executions", func(c *gin.Context) {
			status := c.Query("status")
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
			offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

			execs, total, err := governanceExecutor.ListExecutions(status, limit, offset)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"executions": execs,
				"total":      total,
			})
		})

		// 查询执行详情
		v4.GET("/governance/executions/:id", func(c *gin.Context) {
			id, err := strconv.ParseUint(c.Param("id"), 10, 32)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
				return
			}

			exec, err := governanceExecutor.GetExecution(uint(id))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "execution not found"})
				return
			}
			c.JSON(http.StatusOK, exec)
		})

		// 取消执行
		v4.PUT("/governance/executions/:id/cancel", func(c *gin.Context) {
			id, err := strconv.ParseUint(c.Param("id"), 10, 32)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
				return
			}

			if err := governanceExecutor.CancelExecution(uint(id)); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
		})

		// 治理统计
		v4.GET("/governance/stats", func(c *gin.Context) {
			stats, err := governanceExecutor.GetStats()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, stats)
		})
	}

	log.Printf("API Server listening on %s", cfg.Server.Addr)
	if err := r.Run(cfg.Server.Addr); err != nil {
		log.Fatalf("Failed to run API server: %v", err)
	}
}

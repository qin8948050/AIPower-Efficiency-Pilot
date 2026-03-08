package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qxw/aipower-efficiency-pilot/internal/collector"
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
	log.Println("Starting AIPower-Efficiency-Pilot Collector...")

	// 0. 加载配置
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. 初始化 Redis
	redisCli, err := storage.NewRedisClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	log.Println("Connected to Redis successfully.")

	// 1.5 初始化 MySQL
	mysqlCli, err := storage.NewMySQLClient(cfg.MySQL.DSN)
	if err != nil {
		log.Fatalf("Failed to initialize MySQL: %v", err)
	}
	log.Println("Connected to MySQL successfully.")

	// 2. 初始化 K8s Collector
	k8sColl, err := collector.NewK8sCollector(cfg.K8s.Kubeconfig, redisCli, mysqlCli)
	if err != nil {
		log.Fatalf("Failed to initialize K8s Collector: %v", err)
	}

	// 3. 初始化 Prometheus Collector
	promColl, err := collector.NewPrometheusCollector(cfg.Prometheus.URL, redisCli)
	if err != nil {
		log.Fatalf("Failed to initialize Prometheus Collector: %v", err)
	}

	// 启动收集器协程
	go func() {
		// 阻塞运行 Informer
		if err := k8sColl.Start(ctx); err != nil {
			log.Fatalf("K8s collector stopped with error: %v", err)
		}
	}()

	go func() {
		// 每 5 分钟轮询一次 Prometheus 指标
		promColl.Start(ctx, 5*time.Minute)
	}()

	// 优雅退出机制
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	s := <-sigCh

	log.Printf("Received signal %s, initiating shutdown...", s)
	cancel()
	time.Sleep(2 * time.Second)
	log.Println("Shutdown complete.")
}

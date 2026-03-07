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
	"github.com/qxw/aipower-efficiency-pilot/internal/storage"
)

var (
	kubeconfig    string
	redisAddr     string
	prometheusURL string
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&redisAddr, "redis-addr", "localhost:6379", "Redis server address")
	flag.StringVar(&prometheusURL, "prometheus-url", "http://localhost:9090", "Prometheus server URL")
	flag.Parse()
}

func main() {
	log.Println("Starting AIPower-Efficiency-Pilot Collector...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. 初始化 Redis
	redisCli, err := storage.NewRedisClient(redisAddr, "", 0)
	if err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	log.Println("Connected to Redis successfully.")

	// 2. 初始化 K8s Collector
	k8sColl, err := collector.NewK8sCollector(kubeconfig, redisCli)
	if err != nil {
		log.Fatalf("Failed to initialize K8s Collector: %v", err)
	}

	// 3. 初始化 Prometheus Collector
	promColl, err := collector.NewPrometheusCollector(prometheusURL, redisCli)
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

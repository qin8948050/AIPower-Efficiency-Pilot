package collector

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/qxw/aipower-efficiency-pilot/internal/storage"
	"github.com/qxw/aipower-efficiency-pilot/internal/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sCollector struct {
	client *kubernetes.Clientset
	redis  *storage.RedisClient
	mysql  *storage.MySQLClient
}

// NewK8sCollector 初始化 K8s 采集器
func NewK8sCollector(kubeconfig string, redisCli *storage.RedisClient, mysqlCli *storage.MySQLClient) (*K8sCollector, error) {
	var config *rest.Config
	var err error

	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to build kube config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %v", err)
	}

	return &K8sCollector{
		client: clientset,
		redis:  redisCli,
		mysql:  mysqlCli,
	}, nil
}

func (c *K8sCollector) Start(ctx context.Context) error {
	factory := informers.NewSharedInformerFactory(c.client, time.Minute*10)

	// 监听 Node
	nodeInformer := factory.Core().V1().Nodes().Informer()
	_, err := nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.handleNode(obj.(*v1.Node))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			// 更新时如果 label 变化可以更新 Redis
			c.handleNode(newObj.(*v1.Node))
		},
	})
	if err != nil {
		return fmt.Errorf("failed to add node event handler: %v", err)
	}

	// 监听 Pod
	podInformer := factory.Core().V1().Pods().Informer()
	_, err = podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.handlePodAdd(obj.(*v1.Pod))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.handlePodUpdate(oldObj.(*v1.Pod), newObj.(*v1.Pod))
		},
		DeleteFunc: func(obj interface{}) {
			c.handlePodDelete(obj.(*v1.Pod))
		},
	})
	if err != nil {
		return fmt.Errorf("failed to add pod event handler: %v", err)
	}

	log.Println("Starting Informers...")
	factory.Start(ctx.Done())

	for typ, ok := range factory.WaitForCacheSync(ctx.Done()) {
		if !ok {
			return fmt.Errorf("failed to wait for cache sync for %v", typ)
		}
	}

	log.Println("Cache synced, collector is running.")
	<-ctx.Done()
	return nil
}

func (c *K8sCollector) handleNode(node *v1.Node) {
	labels := node.GetLabels()
	poolID, ok := labels["nvidia.com/pool-id"]
	gpuModel := labels["nvidia.com/gpu.product"]

	if !ok {
		// 回退策略：基于型号
		if gpuModel != "" {
			poolID = fmt.Sprintf("Default-%s-Pool", strings.ToUpper(strings.ReplaceAll(gpuModel, " ", "-")))
		} else {
			poolID = "Unknown-Pool"
		}
	}

	// 从 PoolID 解析切片配置
	slicingMode, maxUnits := parsePoolSlicingConfig(poolID, gpuModel, labels)

	// 识别硬指标特性 (NVLink, RDMA 等)
	var features []string
	if val, ok := labels["nvidia.com/gpu.family"]; ok && strings.Contains(strings.ToLower(val), "nvlink") {
		features = append(features, "NVLink")
	}
	if _, ok := labels["nvidia.com/rdma.capable"]; ok {
		features = append(features, "RDMA")
	}

	// 保存至 Redis (实时调度感知)
	if err := c.redis.SaveNodePoolID(node.Name, poolID); err != nil {
		log.Printf("Failed to save node pool info for %s: %v", node.Name, err)
	}

	// 自动注册资产存根至 MySQL (资产管理)
	poolAsset := &storage.ResourcePool{
		ID:              poolID,
		Name:            poolID,
		GPUModel:        gpuModel,
		GPUVendor:       "nvidia", // 默认值，后续可扩展
		HardwareFeatures: strings.Join(features, ","),
		SlicingMode:     slicingMode,
		MaxSlicingUnits: maxUnits,
	}
	if err := c.mysql.UpsertResourcePool(poolAsset); err != nil {
		log.Printf("Failed to auto-register pool asset %s: %v", poolID, err)
	} else {
		log.Printf("Asset Synced: Pool %s (Model: %s, Mode: %s, MaxUnits: %d)",
			poolID, gpuModel, slicingMode, maxUnits)
	}
}

// parsePoolSlicingConfig 从池子 ID 和节点标签解析切片配置
// 命名规范: pool-{mode}-{vendor}-{spec}
// 例如: pool-full-A100, pool-mig-A100-2g, pool-ts-T4-30
func parsePoolSlicingConfig(poolID, gpuModel string, nodeLabels map[string]string) (mode string, maxUnits int) {
	// 优先从节点标签判断
	if _, ok := nodeLabels["nvidia.com/mig.config"]; ok {
		return "MIG", getDefaultMIGUnits(gpuModel)
	}
	if _, ok := nodeLabels["nvidia.com/mps.capable"]; ok {
		return "MPS", 100 // MPS 按百分比
	}

	// 从 PoolID 解析
	// 格式: pool-{mode}-{vendor}-{spec}
	// 例如: pool-full-A100, pool-mig-A100-2g, pool-ts-T4-30
	parts := strings.Split(poolID, "-")
	if len(parts) < 2 {
		return "Full", 1
	}

	modePart := parts[1]
	switch modePart {
	case "full":
		return "Full", 1
	case "mig":
		// pool-mig-A100-2g → MIG 模式，A100 最大 7 个 MIG 实例
		return "MIG", getDefaultMIGUnits(gpuModel)
	case "mps":
		return "MPS", 100
	case "ts":
		return "TS", 100
	}

	return "Full", 1
}

// getDefaultMIGUnits 获取 GPU 型号对应的最大 MIG 实例数
func getDefaultMIGUnits(gpuModel string) int {
	defaults := map[string]int{
		"NVIDIA A100-SXM4-80GB": 7,
		"NVIDIA A100-SXM4-40GB": 7,
		"NVIDIA H100-SXM5-80GB": 7,
	}
	if v, ok := defaults[gpuModel]; ok {
		return v
	}
	// 未知型号，默认返回 1
	return 1
}

func (c *K8sCollector) handlePodAdd(pod *v1.Pod) {
	if pod.Spec.NodeName == "" {
		return // 还没调度完毕
	}
	c.processPod(pod)
}

func (c *K8sCollector) handlePodUpdate(oldPod, newPod *v1.Pod) {
	if oldPod.Spec.NodeName == "" && newPod.Spec.NodeName != "" {
		// Pod just scheduled
		c.processPod(newPod)
	}
}

func (c *K8sCollector) handlePodDelete(pod *v1.Pod) {
	// 1. 删除 Redis 实时快照
	err := c.redis.DeletePodTrace(pod.Namespace, pod.Name)
	if err != nil {
		log.Printf("Failed to delete pod trace %s/%s: %v", pod.Namespace, pod.Name, err)
	} else {
		log.Printf("Pod Deleted, removed from trace: %s/%s", pod.Namespace, pod.Name)
	}

	// 2. 关闭 MySQL 审计记录 (Life-Trace)
	// 使用 Pod 的实际删除时间 (DeletionTimestamp)，而非当前时间
	endTime := time.Now()
	if pod.DeletionTimestamp != nil && !pod.DeletionTimestamp.IsZero() {
		endTime = pod.DeletionTimestamp.Time
	}
	if err := c.mysql.CloseLifeTrace(pod.Namespace, pod.Name, endTime); err != nil {
		log.Printf("Failed to close MySQL life-trace for %s/%s: %v", pod.Namespace, pod.Name, err)
	}
}

func (c *K8sCollector) processPod(pod *v1.Pod) {
	// 过滤没有请求 GPU 的 Pod (通常为系统组件)
	hasGPU := false
	for _, container := range pod.Spec.Containers {
		for resourceName := range container.Resources.Limits {
			rn := string(resourceName)
			if strings.HasPrefix(rn, "nvidia.com/gpu") ||
				strings.HasPrefix(rn, "nvidia.com/mig") ||
				strings.Contains(rn, "vcuda") ||
				strings.Contains(rn, "gpu-core") {
				hasGPU = true
				break
			}
		}
	}
	if !hasGPU {
		return // 忽略非 GPU 任务
	}

	// 1. 获取 Node 所属池子
	poolID, err := c.redis.GetNodePoolID(pod.Spec.NodeName)
	if err != nil || poolID == "" {
		// Fallback
		node, err := c.client.CoreV1().Nodes().Get(context.TODO(), pod.Spec.NodeName, metav1.GetOptions{})
		if err == nil {
			c.handleNode(node)
			poolID, _ = c.redis.GetNodePoolID(pod.Spec.NodeName)
		}
	}

	// 2. 获取池子切片配置
	nodePool, err := c.mysql.GetResourcePool(poolID)
	if err != nil || nodePool == nil {
		// fallback: 默认配置
		nodePool = &storage.ResourcePool{
			SlicingMode:     "Full",
			MaxSlicingUnits: 1,
		}
	}

	// 3. 计算 Pod 申请的切片单元数和权重
	slicingUnits := countSlicingUnits(pod, nodePool.SlicingMode)
	slicingWeight := 1.0
	if nodePool.MaxSlicingUnits > 0 {
		slicingWeight = float64(slicingUnits) / float64(nodePool.MaxSlicingUnits)
	}

	// 4. 提取 GPU 设备信息
	gpus := c.extractGPUInfo(pod)

	// 5. 提取业务归属标签
	teamLabel := pod.Labels["app.kubernetes.io/team"]
	projectLabel := pod.Labels["app.kubernetes.io/project"]

	// 6. 构建 Trace 实体
	trace := &types.PodTrace{
		Namespace:     pod.Namespace,
		PodName:       pod.Name,
		PodUID:        string(pod.UID),
		NodeName:      pod.Spec.NodeName,
		PoolID:        poolID,
		SlicingMode:   types.SlicingMode(nodePool.SlicingMode),
		SlicingUnits:  slicingUnits,
		SlicingWeight: slicingWeight,
		GPUs:          gpus,
		StartTime:     pod.CreationTimestamp.Time,
		Labels:        pod.Labels,
		TeamLabel:     teamLabel,
		ProjectLabel:  projectLabel,
	}

	if err := c.redis.SavePodTrace(trace); err != nil {
		log.Printf("Error caching Pod Trace %s/%s: %v", trace.Namespace, trace.PodName, err)
	} else {
		log.Printf("Pod %s/%s scheduled on %s (Pool: %s, Mode: %s, Units: %d, Weight: %.3f, GPUs: %v)",
			trace.Namespace, trace.PodName, trace.NodeName, trace.PoolID,
			trace.SlicingMode, slicingUnits, slicingWeight, trace.GPUs)
	}

	// 7. 持久化 MySQL 审计记录 (Life-Trace)
	if err := c.mysql.SaveLifeTrace(trace); err != nil {
		log.Printf("Failed to save MySQL life-trace for %s/%s: %v", trace.Namespace, trace.PodName, err)
	}
}

// countSlicingUnits 统计 Pod 申请的切片单元数
func countSlicingUnits(pod *v1.Pod, mode string) int {
	switch mode {
	case "Full":
		// 整卡：请求 nvidia.com/gpu=1 表示使用 1 张卡
		for _, c := range pod.Spec.Containers {
			for resName, qty := range c.Resources.Limits {
				if string(resName) == "nvidia.com/gpu" {
					return int(qty.Value())
				}
			}
		}
		return 1

	case "MIG":
		// MIG：统计 nvidia.com/mig-* 资源请求总和
		var total int64
		for _, c := range pod.Spec.Containers {
			for resName, qty := range c.Resources.Limits {
				if strings.HasPrefix(string(resName), "nvidia.com/mig-") {
					total += qty.Value()
				}
			}
		}
		if total == 0 {
			return 1 // 默认申请 1 个
		}
		return int(total)

	case "MPS":
		// MPS：从环境变量获取百分比（可能是 "50" 或 "50%"）
		for _, c := range pod.Spec.Containers {
			for _, env := range c.Env {
				if env.Name == "CUDA_MPS_ACTIVE_THREAD_PERCENTAGE" {
					pct := stripNonDigits(env.Value)
					if pct > 0 {
						return pct
					}
				}
			}
		}
		return 50 // 默认 50%

	case "TS":
		// TS/vGPU：从注解获取百分比（可能是 "30" 或 "30%"）
		if pctStr, ok := pod.Annotations["nvidia.com/gpu-percentage"]; ok {
			pct := stripNonDigits(pctStr)
			if pct > 0 {
				return pct
			}
		}
		return 100 // 默认 100%
	}

	return 1
}

// stripNonDigits 去除字符串中的非数字字符，返回整数
// 例如: "30%" -> 30, "50" -> 50, "100%" -> 100
func stripNonDigits(s string) int {
	var result int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		}
	}
	return result
}

// extractGPUInfo 提取分配给 Pod 的 GPU 标识
func (c *K8sCollector) extractGPUInfo(pod *v1.Pod) []string {
	var gpus []string
	// 1. 尝试环境变量 NVIDIA_VISIBLE_DEVICES
	for _, container := range pod.Spec.Containers {
		for _, env := range container.Env {
			if env.Name == "NVIDIA_VISIBLE_DEVICES" && env.Value != "" && env.Value != "all" && env.Value != "none" {
				gpus = append(gpus, strings.Split(env.Value, ",")...)
			}
		}
	}

	// 2. 尝试常见注解 (例如 Volcano/TKE)
	if val, ok := pod.Annotations["nvidia.com/gpu-uuid"]; ok {
		gpus = append(gpus, strings.Split(val, ",")...)
	}
	if val, ok := pod.Annotations["tke.cloud.tencent.com/gpu-assigned-ids"]; ok {
		gpus = append(gpus, strings.Split(val, ",")...)
	}
	return gpus
}

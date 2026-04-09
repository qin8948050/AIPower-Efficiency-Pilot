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
	// 1. 识别 Node 的 PoolID
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

	// 2. 识别硬指标特性 (NVLink, RDMA 等)
	var features []string
	if val, ok := labels["nvidia.com/gpu.family"]; ok && strings.Contains(strings.ToLower(val), "nvlink") {
		features = append(features, "NVLink")
	}
	if _, ok := labels["nvidia.com/rdma.capable"]; ok {
		features = append(features, "RDMA")
	}
	// 简单的切分模式感知
	slicing := "Full"
	if _, ok := labels["nvidia.com/mig.config"]; ok {
		slicing = "MIG"
	} else if _, ok := labels["nvidia.com/mps.capable"]; ok {
		slicing = "MPS"
	}

	// 3. 保存至 Redis (实时调度感知)
	if err := c.redis.SaveNodePoolID(node.Name, poolID); err != nil {
		log.Printf("Failed to save node pool info for %s: %v", node.Name, err)
	}

	// 4. 自动注册资产存根至 MySQL (资产管理)
	poolAsset := &storage.ResourcePool{
		ID:               poolID,
		Name:             poolID, // 默认名与 ID 一致，待管理员修改
		GPUModel:         gpuModel,
		HardwareFeatures: strings.Join(features, ","),
		SlicingMode:      slicing,
	}
	if err := c.mysql.UpsertResourcePool(poolAsset); err != nil {
		log.Printf("Failed to auto-register pool asset %s: %v", poolID, err)
	} else {
		log.Printf("Asset Synced: Pool %s (Model: %s, Features: %v)", poolID, gpuModel, features)
	}
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
	if err := c.mysql.CloseLifeTrace(pod.Namespace, pod.Name); err != nil {
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

	// 2. 识别切分模式
	slicingMode := c.detectSlicingMode(pod)

	// 3. 提取 GPU 设备信息
	gpus := c.extractGPUInfo(pod)

	// 4. 提取业务归属标签
	teamLabel := pod.Labels["app.kubernetes.io/team"]
	projectLabel := pod.Labels["app.kubernetes.io/project"]

	// 5. 构建 Trace 实体
	trace := &types.PodTrace{
		Namespace:    pod.Namespace,
		PodName:      pod.Name,
		PodUID:       string(pod.UID),
		NodeName:     pod.Spec.NodeName,
		PoolID:       poolID,
		SlicingMode:  slicingMode,
		GPUs:         gpus,
		StartTime:    pod.CreationTimestamp.Time,
		Labels:       pod.Labels,
		TeamLabel:    teamLabel,
		ProjectLabel: projectLabel,
	}

	if err := c.redis.SavePodTrace(trace); err != nil {
		log.Printf("Error caching Pod Trace %s/%s: %v", trace.Namespace, trace.PodName, err)
	} else {
		log.Printf("Pod %s/%s scheduled on %s (%s, Mode: %s, GPUs: %v)",
			trace.Namespace, trace.PodName, trace.NodeName, trace.PoolID, trace.SlicingMode, trace.GPUs)
	}

	// 4. 持久化 MySQL 审计记录 (Life-Trace)
	if err := c.mysql.SaveLifeTrace(trace); err != nil {
		log.Printf("Failed to save MySQL life-trace for %s/%s: %v", trace.Namespace, trace.PodName, err)
	}
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

// detectSlicingMode 基于资源申请和环境变量探测虚拟化模式
func (c *K8sCollector) detectSlicingMode(pod *v1.Pod) types.SlicingMode {
	for _, container := range pod.Spec.Containers {
		// 1. MIG 探测 (通过资源名)
		for resName := range container.Resources.Limits {
			if strings.HasPrefix(string(resName), "nvidia.com/mig-") {
				return types.SlicingModeMIG
			}
		}
		// 2. MPS 探测 (通过环境变量)
		for _, env := range container.Env {
			if env.Name == "CUDA_MPS_PIPE_DIRECTORY" || env.Name == "NVIDIA_MPS_COPY_THRESHOLD" {
				return types.SlicingModeMPS
			}
		}
		// 3. TS (Time Slicing/vGPU) 探测
		for resName := range container.Resources.Limits {
			rn := string(resName)
			if strings.Contains(rn, "vcuda") || strings.Contains(rn, "gpu-core") || strings.Contains(rn, "gpu-percentage") {
				return types.SlicingModeTS
			}
		}
	}
	return types.SlicingModeFull
}

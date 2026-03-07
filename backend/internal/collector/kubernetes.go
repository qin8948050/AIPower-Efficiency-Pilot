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
}

// NewK8sCollector 初始化 K8s 采集器
func NewK8sCollector(kubeconfig string, redisCli *storage.RedisClient) (*K8sCollector, error) {
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
	// 识别 Node 的 PoolID
	labels := node.GetLabels()
	poolID, ok := labels["nvidia.com/pool-id"]
	if !ok {
		// 回退策略：基于型号
		if gpuModel, hasModel := labels["nvidia.com/gpu.product"]; hasModel {
			poolID = fmt.Sprintf("Default-%s-Pool", strings.ToUpper(strings.ReplaceAll(gpuModel, " ", "-")))
		} else {
			poolID = "Unknown-Pool"
		}
	}

	if err := c.redis.SaveNodePoolID(node.Name, poolID); err != nil {
		log.Printf("Failed to save node pool info for %s: %v", node.Name, err)
	} else {
		log.Printf("Discovered Node Pool: %s -> %s", node.Name, poolID)
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
	err := c.redis.DeletePodTrace(pod.Namespace, pod.Name)
	if err != nil {
		log.Printf("Failed to delete pod trace %s/%s: %v", pod.Namespace, pod.Name, err)
	} else {
		log.Printf("Pod Deleted, removed from trace: %s/%s", pod.Namespace, pod.Name)
	}
}

func (c *K8sCollector) processPod(pod *v1.Pod) {
	// 过滤没有请求 GPU 的 Pod (通常为系统组件)
	hasGPU := false
	for _, container := range pod.Spec.Containers {
		for resourceName := range container.Resources.Limits {
			if strings.HasPrefix(string(resourceName), "nvidia.com/gpu") || strings.HasPrefix(string(resourceName), "nvidia.com/mig") {
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

	// 3. 构建 Trace 实体
	trace := &types.PodTrace{
		Namespace:   pod.Namespace,
		PodName:     pod.Name,
		PodUID:      string(pod.UID),
		NodeName:    pod.Spec.NodeName,
		PoolID:      poolID,
		SlicingMode: slicingMode,
		StartTime:   pod.CreationTimestamp.Time,
		Labels:      pod.Labels,
	}

	if err := c.redis.SavePodTrace(trace); err != nil {
		log.Printf("Error caching Pod Trace %s/%s: %v", trace.Namespace, trace.PodName, err)
	} else {
		log.Printf("Pod %s/%s scheduled on %s (%s, Mode: %s)", trace.Namespace, trace.PodName, trace.NodeName, trace.PoolID, trace.SlicingMode)
	}
}

// detectSlicingMode 基于正则表达式和环境变量识别虚拟化模式
func (c *K8sCollector) detectSlicingMode(pod *v1.Pod) types.SlicingMode {
	// 简单的探针策略
	for _, container := range pod.Spec.Containers {
		// 1. 是否申请了 MIG
		for resName := range container.Resources.Requests {
			if strings.HasPrefix(string(resName), "nvidia.com/mig-") {
				return types.SlicingModeMIG
			}
		}
		// 2. 是否注入了 MPS 变量
		for _, env := range container.Env {
			if env.Name == "CUDA_MPS_PIPE_DIRECTORY" {
				return types.SlicingModeMPS
			}
		}
	}

	// 3. 检查是否有 TS (Time Slicing - 超分) 的标签或特定资源名
	// 例如 TKE 可能会使用 tencent.com/vcuda-core
	for _, container := range pod.Spec.Containers {
		for resName := range container.Resources.Requests {
			if strings.Contains(string(resName), "vcuda") || strings.Contains(string(resName), "gpu-core") {
				return types.SlicingModeTS
			}
		}
	}

	return types.SlicingModeFull
}

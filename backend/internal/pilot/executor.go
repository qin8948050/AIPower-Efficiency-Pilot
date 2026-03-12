package pilot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/qxw/aipower-efficiency-pilot/internal/storage"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GovernanceAction 治理动作类型
type GovernanceAction string

const (
	ActionDowngrade       GovernanceAction = "downgrade"        // 降配
	ActionMigrate        GovernanceAction = "migrate"          // 迁移
	ActionDowngradeMigrate GovernanceAction = "downgrade_migrate" // 降配+迁移
)

// ExecutionStatus 执行状态
type ExecutionStatus string

const (
	StatusPending    ExecutionStatus = "pending"
	StatusExecuting  ExecutionStatus = "executing"
	StatusCompleted  ExecutionStatus = "completed"
	StatusFailed     ExecutionStatus = "failed"
	StatusCancelled  ExecutionStatus = "cancelled"
)

// GovernanceExecutor 治理执行器
type GovernanceExecutor struct {
	client   *kubernetes.Clientset
	mysql    *storage.MySQLClient
	recorder *ExecutionRecorder
}

// NewGovernanceExecutor 创建治理执行器
func NewGovernanceExecutor(kubeconfig string, mysqlCli *storage.MySQLClient) (*GovernanceExecutor, error) {
	var clientset *kubernetes.Clientset
	var err error

	if kubeconfig != "" {
		clientset, err = NewClientsetFromKubeconfig(kubeconfig)
	} else {
		clientset, err = NewInClusterClientset()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %v", err)
	}

	return &GovernanceExecutor{
		client:   clientset,
		mysql:    mysqlCli,
		recorder: NewExecutionRecorder(mysqlCli),
	}, nil
}

// ExecuteRequest 执行请求
type ExecuteRequest struct {
	ReportID              uint                   `json:"report_id"`
	TaskName              string                 `json:"task_name"`
	Namespace             string                 `json:"namespace"`
	ActionType            GovernanceAction       `json:"action_type"`
	FromPool             string                 `json:"from_pool"`
	ToPool               string                 `json:"to_pool"`
	FromGPU              int                    `json:"from_gpu"`
	ToGPU                int                    `json:"to_gpu"`
	ExecuteNow           bool                   `json:"execute_now"`
}

// ExecuteResponse 执行响应
type ExecuteResponse struct {
	ExecutionID uint           `json:"execution_id"`
	Status      ExecutionStatus `json:"status"`
	Message     string         `json:"message,omitempty"`
}

// Execute 执行治理动作
func (e *GovernanceExecutor) Execute(ctx context.Context, req ExecuteRequest) (*ExecuteResponse, error) {
	// 生成 patch 内容
	patchType, patchContent := generatePatch(req)

	// 1. 创建执行记录
	execution := &storage.GovernanceExecution{
		ReportID:     req.ReportID,
		TaskName:     req.TaskName,
		Namespace:    req.Namespace,
		ActionType:   string(req.ActionType),
		FromPool:     req.FromPool,
		ToPool:       req.ToPool,
		FromGPU:      req.FromGPU,
		ToGPU:        req.ToGPU,
		PatchType:    patchType,
		PatchContent: patchContent,
		Status:       string(StatusPending),
	}

	if err := e.recorder.Create(execution); err != nil {
		return nil, fmt.Errorf("failed to create execution record: %v", err)
	}

	// 2. 如果立即执行
	if req.ExecuteNow {
		go func() {
			e.executeAsync(execution.ID, req)
		}()
	}

	return &ExecuteResponse{
		ExecutionID: execution.ID,
		Status:      StatusPending,
		Message:     "Execution scheduled",
	}, nil
}

// executeAsync 异步执行治理
func (e *GovernanceExecutor) executeAsync(execID uint, req ExecuteRequest) {
	ctx := context.Background()

	// 更新状态为执行中
	if err := e.recorder.UpdateStatus(execID, string(StatusExecuting)); err != nil {
		log.Printf("Failed to update status to executing: %v", err)
		return
	}

	var err error
	switch req.ActionType {
	case ActionDowngrade:
		err = e.executeDowngrade(ctx, req.Namespace, req.TaskName, req.ToGPU)
	case ActionMigrate:
		err = e.executeMigrate(ctx, req.Namespace, req.TaskName, req.ToPool)
	case ActionDowngradeMigrate:
		err = e.executeDowngradeMigrate(ctx, req.Namespace, req.TaskName, req.ToGPU, req.ToPool)
	default:
		err = fmt.Errorf("unknown action type: %s", req.ActionType)
	}

	// 更新最终状态
	if err != nil {
		e.recorder.UpdateStatusWithError(execID, string(StatusFailed), err.Error())
		log.Printf("Governance execution %d failed: %v", execID, err)
	} else {
		e.recorder.MarkCompleted(execID)
		log.Printf("Governance execution %d completed successfully", execID)
	}
}

// executeDowngrade 降配执行
func (e *GovernanceExecutor) executeDowngrade(ctx context.Context, namespace, podName string, toGPU int) error {
	pod, err := e.client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get pod %s/%s: %v", namespace, podName, err)
	}

	// 构建新的资源请求
	newPod := pod.DeepCopy()
	for i := range newPod.Spec.Containers {
		container := &newPod.Spec.Containers[i]
		if container.Resources.Limits == nil {
			container.Resources.Limits = v1.ResourceList{}
		}
		// 更新 GPU 限制
		if gpuReq, ok := container.Resources.Limits["nvidia.com/gpu"]; ok {
			newVal := *resource.NewQuantity(int64(toGPU), gpuReq.Format)
			container.Resources.Limits["nvidia.com/gpu"] = newVal
		}
	}

	// Patch Pod
	_, err = e.client.CoreV1().Pods(namespace).Patch(ctx, podName, "strategic-merge-patch",
		generatePatchBytes(pod, newPod), metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to patch pod: %v", err)
	}

	log.Printf("Downgraded pod %s/%s to %d GPUs", namespace, podName, toGPU)
	return nil
}

// executeMigrate 迁移执行
func (e *GovernanceExecutor) executeMigrate(ctx context.Context, namespace, podName, toPool string) error {
	pod, err := e.client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get pod %s/%s: %v", namespace, podName, err)
	}

	// 为 Pod 添加目标池标签，触发调度器重新调度
	newPod := pod.DeepCopy()
	if newPod.Labels == nil {
		newPod.Labels = make(map[string]string)
	}
	// 添加治理迁移标签
	newPod.Labels["governance.migration/pending"] = "true"
	newPod.Labels["governance.migration/to-pool"] = toPool

	_, err = e.client.CoreV1().Pods(namespace).Patch(ctx, podName, "strategic-merge-patch",
		generateLabelPatchBytes(pod, newPod), metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to add migration labels: %v", err)
	}

	// 注意：实际生产环境中需要实现完整的迁移逻辑：
	// 1. 在目标池创建新 Pod
	// 2. 等待新 Pod 就绪
	// 3. 删除原 Pod

	log.Printf("Added migration labels to pod %s/%s for pool %s", namespace, podName, toPool)
	return nil
}

// executeDowngradeMigrate 降配+迁移执行
func (e *GovernanceExecutor) executeDowngradeMigrate(ctx context.Context, namespace, podName string, toGPU int, toPool string) error {
	// 先降配
	if err := e.executeDowngrade(ctx, namespace, podName, toGPU); err != nil {
		return fmt.Errorf("downgrade failed: %v", err)
	}

	// 再迁移
	if err := e.executeMigrate(ctx, namespace, podName, toPool); err != nil {
		return fmt.Errorf("migration failed: %v", err)
	}

	log.Printf("Downgrade+Migration completed for pod %s/%s", namespace, podName)
	return nil
}

// generatePatch 根据请求生成 patch 类型和内容
func generatePatch(req ExecuteRequest) (string, string) {
	switch req.ActionType {
	case ActionDowngrade:
		patch := fmt.Sprintf(`{"spec":{"containers":[{"name":"*","resources":{"limits":{"nvidia.com/gpu":"%d"}}}}}]}`, req.ToGPU)
		return "strategic-merge-patch", patch
	case ActionMigrate:
		patch := fmt.Sprintf(`{"metadata":{"labels":{"governance.migration/pending":"true","governance.migration/to-pool":"%s"}}}`, req.ToPool)
		return "strategic-merge-patch", patch
	case ActionDowngradeMigrate:
		// 降配+迁移的 patch
		patch := fmt.Sprintf(`{"spec":{"containers":[{"name":"*","resources":{"limits":{"nvidia.com/gpu":"%d"}}}]},"metadata":{"labels":{"governance.migration/pending":"true","governance.migration/to-pool":"%s"}}}`, req.ToGPU, req.ToPool)
		return "strategic-merge-patch", patch
	default:
		return "", ""
	}
}

// generatePatchBytes 生成 Patch 字节
func generatePatchBytes(oldPod, newPod *v1.Pod) []byte {
	type patch struct {
		Spec struct {
			Containers []struct {
				Name      string `json:"name"`
				Resources struct {
					Limits v1.ResourceList `json:"limits"`
				} `json:"resources"`
			} `json:"containers"`
		} `json:"spec"`
	}

	var p patch
	p.Spec.Containers = make([]struct {
		Name      string `json:"name"`
		Resources struct {
			Limits v1.ResourceList `json:"limits"`
		} `json:"resources"`
	}, len(newPod.Spec.Containers))

	for i, c := range newPod.Spec.Containers {
		p.Spec.Containers[i].Name = c.Name
		p.Spec.Containers[i].Resources.Limits = c.Resources.Limits
	}

	b, _ := json.Marshal(p)
	return b
}

// generateLabelPatchBytes 生成标签 Patch 字节
func generateLabelPatchBytes(oldPod, newPod *v1.Pod) []byte {
	type patch struct {
		Metadata struct {
			Labels map[string]string `json:"labels"`
		} `json:"metadata"`
	}

	var p patch
	p.Metadata.Labels = newPod.Labels

	b, _ := json.Marshal(p)
	return b
}

// GetExecution 获取执行记录
func (e *GovernanceExecutor) GetExecution(id uint) (*storage.GovernanceExecution, error) {
	return e.recorder.GetByID(id)
}

// ListExecutions 获取执行列表
func (e *GovernanceExecutor) ListExecutions(status string, limit, offset int) ([]storage.GovernanceExecution, int64, error) {
	return e.recorder.List(status, limit, offset)
}

// CancelExecution 取消执行
func (e *GovernanceExecutor) CancelExecution(id uint) error {
	return e.recorder.UpdateStatus(id, string(StatusCancelled))
}

// GetStats 获取治理统计
func (e *GovernanceExecutor) GetStats() (*GovernanceStats, error) {
	return e.recorder.GetStats()
}

// GovernanceStats 治理统计
type GovernanceStats struct {
	TotalExecuted int64 `json:"total_executed"`
	PendingCount  int64 `json:"pending_count"`
}

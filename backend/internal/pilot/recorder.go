package pilot

import (
	"strconv"

	"github.com/qxw/aipower-efficiency-pilot/internal/storage"
)

// ExecutionRecorder 执行记录器
type ExecutionRecorder struct {
	mysql *storage.MySQLClient
}

// NewExecutionRecorder 创建执行记录器
func NewExecutionRecorder(mysqlCli *storage.MySQLClient) *ExecutionRecorder {
	return &ExecutionRecorder{mysql: mysqlCli}
}

// Create 创建执行记录
func (r *ExecutionRecorder) Create(exec *storage.GovernanceExecution) error {
	return r.mysql.SaveGovernanceExecution(exec)
}

// UpdateStatus 更新执行状态
func (r *ExecutionRecorder) UpdateStatus(id uint, status string) error {
	return r.mysql.UpdateGovernanceExecutionStatus(id, status)
}

// UpdateStatusWithError 更新执行状态（带错误信息）
func (r *ExecutionRecorder) UpdateStatusWithError(id uint, status, errMsg string) error {
	return r.mysql.UpdateGovernanceExecutionStatusWithError(id, status, errMsg)
}

// MarkCompleted 标记完成
func (r *ExecutionRecorder) MarkCompleted(id uint) error {
	return r.mysql.MarkGovernanceExecutionCompleted(id)
}

// GetByID 根据ID获取
func (r *ExecutionRecorder) GetByID(id uint) (*storage.GovernanceExecution, error) {
	return r.mysql.GetGovernanceExecutionByID(id)
}

// List 获取执行列表
func (r *ExecutionRecorder) List(status string, limit, offset int) ([]storage.GovernanceExecution, int64, error) {
	return r.mysql.ListGovernanceExecutions(status, limit, offset)
}

// GetStats 获取统计
func (r *ExecutionRecorder) GetStats() (*GovernanceStats, error) {
	stats, err := r.mysql.GetGovernanceStats()
	if err != nil {
		return nil, err
	}
	return &GovernanceStats{
		TotalExecuted: stats["total_executed"],
		PendingCount:  stats["pending_count"],
	}, nil
}

// StringToUint 字符串转Uint
func StringToUint(s string) uint {
	i, _ := strconv.ParseUint(s, 10, 32)
	return uint(i)
}

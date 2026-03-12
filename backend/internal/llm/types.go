package llm

import "time"

// InsightSummary AI 特征摘要（脱敏后供 LLM 使用）
type InsightSummary struct {
	PoolID              string              `json:"pool_id"`
	TimeRange           string              `json:"time_range"` // e.g., "7d"
	AvgUtilization      float64             `json:"avg_utilization"`
	MaxUtilization      float64             `json:"max_utilization"`
	CostTotal           float64             `json:"cost_total"`
	WasteCost           float64             `json:"waste_cost"` // 低利用率导致的浪费
	PodCount            int                 `json:"pod_count"`
	LowUtilPods         []LowUtilPod        `json:"low_util_pods,omitempty"`
	HighJitterPods      []JitterPod         `json:"high_jitter_pods,omitempty"`
	FeatureMismatchPods []FeatureMismatchPod `json:"feature_mismatch_pods,omitempty"`
	HardwareFeatures    string              `json:"hardware_features,omitempty"`
	SlicingMode         string              `json:"slicing_mode"`
	// 分析类型标志（根据池子类型和条件判定）
	IsDowngradeTarget  bool `json:"is_downgrade_target"`  // Full/MIG 池，利用率<30%持续3天
	IsIsolationTarget  bool `json:"is_isolation_target"`  // MPS/TS 池，抖动>15%
	IsFeatureMismatch bool `json:"is_feature_mismatch"`  // 高端特性(NVLink/FP8)利用率<10%
}

type LowUtilPod struct {
	PodName      string  `json:"pod_name"`
	Namespace    string  `json:"namespace"`
	AvgUtil      float64 `json:"avg_util"`
	EstWasteCost float64 `json:"est_waste_cost"`
}

type JitterPod struct {
	PodName   string  `json:"pod_name"`
	Namespace string  `json:"namespace"`
	JitterPct float64 `json:"jitter_pct"`
}

// FeatureMismatchPod 特性不匹配 Pod
type FeatureMismatchPod struct {
	PodName          string  `json:"pod_name"`
	Namespace        string  `json:"namespace"`
	RequiredFeatures string  `json:"required_features"` // Pod 声明需要的特性
	AvgUtil          float64 `json:"avg_util"`
	EstWasteCost     float64 `json:"est_waste_cost"`
}

// InsightReport AI 诊断报告
// 核心对象是任务（Pod/PyTorchJob），而非资源池
type InsightReport struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	GeneratedAt time.Time `json:"generated_at" gorm:"column:generated_at"`
	TaskName    string    `json:"task_name" gorm:"column:task_name;index"`    // 任务名（Pod名称或PyTorchJob名称）
	Namespace   string    `json:"namespace" gorm:"column:namespace"`          // 任务所在命名空间
	Team        string    `json:"team" gorm:"column:team"`                  // 负责团队
	PoolID      string    `json:"pool_id" gorm:"column:pool_id;index"`       // 任务当前所在的资源池
	ReportType  string    `json:"report_type" gorm:"column:report_type"`     // "downgrade", "isolation", "feature_mismatch"
	Summary     string    `json:"summary" gorm:"column:summary;type:text"`
	RootCause   string    `json:"root_cause" gorm:"column:root_cause;type:text"`
	Actions     string    `json:"actions" gorm:"column:actions;type:text"` // JSON string
	EstSavings  float64   `json:"est_savings" gorm:"column:est_savings"`
	Status      string    `json:"status" gorm:"column:status;default:'pending'"` // "pending", "approved", "rejected"
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (InsightReport) TableName() string {
	return "insight_reports"
}

// InsightAction 治理动作
type InsightAction struct {
	Type      string `json:"type"` // "migrate", "scale_down", "pool_change"
	PodName   string `json:"pod_name"`
	Namespace string `json:"namespace"`
	FromPool  string `json:"from_pool"`
	ToPool    string `json:"to_pool"`
	PatchJSON string `json:"patch_json,omitempty"`
}

// GenerateRequest 生成报告请求
type GenerateRequest struct {
	PoolID   string `json:"pool_id,omitempty"`   // 空表示全量
	Days     int    `json:"days"`                 // 默认 7 天
	Force    bool   `json:"force"`                // 是否强制重新生成
}

// GenerateResponse 生成报告响应
type GenerateResponse struct {
	ReportID string `json:"report_id"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

// ReportListResponse 报告列表响应
type ReportListResponse struct {
	Reports []InsightReport `json:"reports"`
	Total   int64            `json:"total"`
}

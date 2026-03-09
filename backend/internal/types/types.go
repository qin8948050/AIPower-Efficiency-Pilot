package types

import "time"

// SlicingMode 枚举不同的虚拟化模式
type SlicingMode string

const (
	SlicingModeFull SlicingMode = "Full"
	SlicingModeMIG  SlicingMode = "MIG"
	SlicingModeMPS  SlicingMode = "MPS"
	SlicingModeTS   SlicingMode = "TS"
	SlicingModeUnk  SlicingMode = "Unknown"
)

// PodTrace 存储关于 Pod 的池化感知信息
type PodTrace struct {
	Namespace    string            `json:"namespace"`
	PodName      string            `json:"pod_name"`
	PodUID       string            `json:"pod_uid"`
	NodeName     string            `json:"node_name"`
	PoolID       string            `json:"pool_id"`
	SlicingMode  SlicingMode       `json:"slicing_mode"`
	GPUs         []string          `json:"gpus,omitempty"` // 挂载的 GPU UUID 或 Index
	StartTime    time.Time         `json:"start_time"`
	Labels       map[string]string `json:"labels,omitempty"`
	TeamLabel    string            `json:"team_label,omitempty"`    // 业务 Team 标签
	ProjectLabel string            `json:"project_label,omitempty"` // 业务 Project 标签
	Metrics      *GPUMetrics       `json:"metrics,omitempty"`       // 实时指标快照
}

// GPUMetrics 存储 5 分钟级的 GPU 实时指标平均值/峰值
type GPUMetrics struct {
	GPUUtilAvg    float64   `json:"gpu_util_avg"`    // 算力利用率平均值
	GPUUtilMax    float64   `json:"gpu_util_max"`    // 算力利用率峰值
	MemUsedBytes  uint64    `json:"mem_used_bytes"`  // 显存占用 (Bytes)
	MemTotalBytes uint64    `json:"mem_total_bytes"` // 显存总量 (Bytes)
	PowerUsageW   float64   `json:"power_usage_w"`   // 功耗 (Watt)
	LastUpdate    time.Time `json:"last_update"`
}

// NodePoolInfo 存储节点维度的池子信息
type NodePoolInfo struct {
	NodeName string `json:"node_name"`
	PoolID   string `json:"pool_id"`
}

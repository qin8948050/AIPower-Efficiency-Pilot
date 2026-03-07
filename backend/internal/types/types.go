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
	Namespace   string      `json:"namespace"`
	PodName     string      `json:"pod_name"`
	PodUID      string      `json:"pod_uid"`
	NodeName    string      `json:"node_name"`
	PoolID      string      `json:"pool_id"`
	SlicingMode SlicingMode `json:"slicing_mode"`
	GPUs        []string    `json:"gpus,omitempty"` // 挂载的 GPU UUID 或 Index
	StartTime   time.Time   `json:"start_time"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// NodePoolInfo 存储节点维度的池子信息
type NodePoolInfo struct {
	NodeName string `json:"node_name"`
	PoolID   string `json:"pool_id"`
}

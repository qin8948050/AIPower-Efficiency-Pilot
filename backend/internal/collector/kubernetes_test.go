package collector

import (
	"testing"

	"github.com/qxw/aipower-efficiency-pilot/internal/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDetectSlicingMode(t *testing.T) {
	c := &K8sCollector{}

	tests := []struct {
		name     string
		pod      *v1.Pod
		expected types.SlicingMode
	}{
		{
			name: "MIG Mode",
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"nvidia.com/mig-1g.5b": {},
								},
							},
						},
					},
				},
			},
			expected: types.SlicingModeMIG,
		},
		{
			name: "MPS Mode via Env",
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Env: []v1.EnvVar{
								{Name: "CUDA_MPS_PIPE_DIRECTORY", Value: "/tmp/mps"},
							},
						},
					},
				},
			},
			expected: types.SlicingModeMPS,
		},
		{
			name: "Time Slicing Mode (vcuda)",
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"nvidia.com/vcuda-core": {},
								},
							},
						},
					},
				},
			},
			expected: types.SlicingModeTS,
		},
		{
			name: "Full Mode (Default)",
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"nvidia.com/gpu": {},
								},
							},
						},
					},
				},
			},
			expected: types.SlicingModeFull,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.detectSlicingMode(tt.pod)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		} )
	}
}

func TestExtractGPUInfo(t *testing.T) {
	c := &K8sCollector{}

	tests := []struct {
		name     string
		pod      *v1.Pod
		expected []string
	}{
		{
			name: "Visible Devices Env",
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Env: []v1.EnvVar{
								{Name: "NVIDIA_VISIBLE_DEVICES", Value: "GPU-uuid-1,GPU-uuid-2"},
							},
						},
					},
				},
			},
			expected: []string{"GPU-uuid-1", "GPU-uuid-2"},
		},
		{
			name: "GPU UUID Annotation",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"nvidia.com/gpu-uuid": "GPU-uuid-3",
					},
				},
			},
			expected: []string{"GPU-uuid-3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.extractGPUInfo(tt.pod)
			if len(result) != len(tt.expected) {
				t.Errorf("expected length %d, got %d", len(tt.expected), len(result))
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("expected %v, got %v", tt.expected[i], result[i])
				}
			}
		})
	}
}

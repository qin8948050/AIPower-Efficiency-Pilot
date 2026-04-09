package collector

import (
	"fmt"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)



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

func Test_stripNonDigits(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		s    string
		want int
	}{
		// TODO: Add test cases.
		{
			"30%",
			"30%",
			30,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripNonDigits(tt.s)
			fmt.Println(got,tt.want)
			// TODO: update the condition below to compare got with tt.want.
			if got != tt.want {
				t.Errorf("stripNonDigits() = %v, want %v", got, tt.want)
			}
		})
	}
}

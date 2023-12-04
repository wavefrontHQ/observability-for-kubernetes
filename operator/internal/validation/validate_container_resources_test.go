package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/operator/api/common"
)

func TestValidateContainerResources(t *testing.T) {
	t.Run("valid resource limits", func(t *testing.T) {
		resources := &common.ContainerResources{
			Requests: common.ContainerResource{
				CPU:    "10Mi",
				Memory: "10Gi",
			},
			Limits: common.ContainerResource{
				CPU:    "100Mi",
				Memory: "100Gi",
			},
		}
		result := ValidateContainerResources(resources, "my-resource")
		require.True(t, result.IsValid())
	})

	t.Run("does not require requests", func(t *testing.T) {
		resources := &common.ContainerResources{
			Limits: common.ContainerResource{
				CPU:    "10Mi",
				Memory: "10Gi",
			},
		}
		result := ValidateContainerResources(resources, "my-resource")
		require.True(t, result.IsValid())
	})

	t.Run("requires limits", func(t *testing.T) {
		resources := &common.ContainerResources{
			Limits:   common.ContainerResource{},
			Requests: common.ContainerResource{},
		}
		result := ValidateContainerResources(resources, "my-resource")
		require.False(t, result.IsValid())
		require.Equal(t, "[invalid my-resource.resources.limits.memory must be set, invalid my-resource.resources.limits.cpu must be set]", result.Message())
	})

	t.Run("missing cpu limit", func(t *testing.T) {
		resources := &common.ContainerResources{
			Requests: common.ContainerResource{
				CPU:    "10Mi",
				Memory: "10Gi",
			},
			Limits: common.ContainerResource{
				Memory: "100Gi",
			},
		}
		result := ValidateContainerResources(resources, "my-resource")
		require.False(t, result.IsValid())
		require.Equal(t, "invalid my-resource.resources.limits.cpu must be set", result.Message())
	})

	t.Run("missing memory limit", func(t *testing.T) {
		resources := &common.ContainerResources{
			Requests: common.ContainerResource{
				CPU:    "10Mi",
				Memory: "10Gi",
			},
			Limits: common.ContainerResource{
				CPU: "100Mi",
			},
		}
		result := ValidateContainerResources(resources, "my-resource")
		require.False(t, result.IsValid())
		require.Equal(t, "invalid my-resource.resources.limits.memory must be set", result.Message())
	})

	t.Run("invalid cpu request", func(t *testing.T) {
		resources := &common.ContainerResources{
			Requests: common.ContainerResource{
				CPU:    "10MM",
				Memory: "10Gi",
			},
			Limits: common.ContainerResource{
				CPU:    "100Mi",
				Memory: "100Gi",
			},
		}
		result := ValidateContainerResources(resources, "my-resource")
		require.False(t, result.IsValid())
		require.Equal(t, "invalid my-resource.resources.requests.cpu: '10MM'", result.Message())
	})

	t.Run("invalid cpu limit", func(t *testing.T) {
		resources := &common.ContainerResources{
			Requests: common.ContainerResource{
				CPU:    "10Mi",
				Memory: "10Gi",
			},
			Limits: common.ContainerResource{
				CPU:    "100MM",
				Memory: "100Gi",
			},
		}
		result := ValidateContainerResources(resources, "my-resource")
		require.False(t, result.IsValid())
		require.Equal(t, "invalid my-resource.resources.limits.cpu: '100MM'", result.Message())
	})

	t.Run("invalid memory request", func(t *testing.T) {
		resources := &common.ContainerResources{
			Requests: common.ContainerResource{
				CPU:    "10Mi",
				Memory: "10GG",
			},
			Limits: common.ContainerResource{
				CPU:    "100Mi",
				Memory: "100Gi",
			},
		}
		result := ValidateContainerResources(resources, "")
		require.False(t, result.IsValid())
		require.Equal(t, "invalid .resources.requests.memory: '10GG'", result.Message())
	})

	t.Run("invalid memory limit", func(t *testing.T) {
		resources := &common.ContainerResources{
			Requests: common.ContainerResource{
				CPU:    "10Mi",
				Memory: "10Gi",
			},
			Limits: common.ContainerResource{
				CPU:    "100Mi",
				Memory: "100GG",
			},
		}
		result := ValidateContainerResources(resources, "my-resource")
		require.False(t, result.IsValid())
		require.Equal(t, "invalid my-resource.resources.limits.memory: '100GG'", result.Message())
	})

	t.Run("invalid request memory > limit memory", func(t *testing.T) {
		resources := &common.ContainerResources{
			Requests: common.ContainerResource{
				CPU:    "10Mi",
				Memory: "10Gi",
			},
			Limits: common.ContainerResource{
				CPU:    "100Mi",
				Memory: "1Gi",
			},
		}
		result := ValidateContainerResources(resources, "my-resource")
		require.False(t, result.IsValid())
		require.Equal(t, "invalid my-resource.resources.requests.memory: 10Gi must be less than or equal to memory limit", result.Message())
	})

	t.Run("invalid request cpu > limit cpu", func(t *testing.T) {
		resources := &common.ContainerResources{
			Requests: common.ContainerResource{
				CPU:    "1000Mi",
				Memory: "10Gi",
			},
			Limits: common.ContainerResource{
				CPU:    "100Mi",
				Memory: "10Gi",
			},
		}
		result := ValidateContainerResources(resources, "my-resource")
		require.False(t, result.IsValid())
		require.Equal(t, "invalid my-resource.resources.requests.cpu: 1000Mi must be less than or equal to cpu limit", result.Message())
	})
}

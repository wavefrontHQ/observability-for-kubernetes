package patch_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/operator/api/common"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/patch"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const ResourceLimit = "limit"
const ResourceRequest = "request"

func TestContainerResources(t *testing.T) {
	t.Run("apply-able to non-resourced objects", func(t *testing.T) {
		p := patch.ContainerResources{
			Requests: common.Resource{
				EphemeralStorage: "10Gi",
			},
		}
		actualObj := &unstructured.Unstructured{Object: map[string]any{}}
		expectedObj := actualObj.DeepCopy()

		p.Apply(actualObj)

		require.Equal(t, expectedObj, actualObj)
	})

	t.Run("does not apply when empty", func(t *testing.T) {
		p := patch.ContainerResources{}
		obj := resourcedObj(nil)

		p.Apply(obj)

		containers, _, _ := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers")
		_, exists := containers[0].(map[string]any)["resources"]
		require.False(t, exists, "expected resources to not be set")
	})

	t.Run("sets the requests when only the limits are specified", func(t *testing.T) {
		p := patch.ContainerResources{
			Limits: common.Resource{
				CPU:              "100m",
				Memory:           "1Gi",
				EphemeralStorage: "10Gi",
			},
		}
		obj := resourcedObj(nil)

		p.Apply(obj)

		requests := getResourceRequirement(obj, "requests")
		require.Equal(t, p.Limits.CPU, requests["cpu"], "cpu request")
		require.Equal(t, p.Limits.Memory, requests["memory"], "memory request")
		require.Equal(t, p.Limits.EphemeralStorage, requests["ephemeral-storage"], "ephemeral-storage request")

		limits := getResourceRequirement(obj, "limits")
		require.Equal(t, p.Limits.CPU, limits["cpu"], "cpu limit")
		require.Equal(t, p.Limits.Memory, limits["memory"], "memory limit")
		require.Equal(t, p.Limits.EphemeralStorage, limits["ephemeral-storage"], "memory limit")
	})

	scenarios := []struct {
		name      string
		resources patch.ContainerResources
	}{
		{
			name: "cpu",
			resources: patch.ContainerResources{
				Limits: common.Resource{
					CPU: ResourceLimit,
				},
				Requests: common.Resource{
					CPU: ResourceRequest,
				},
			},
		},
		{
			name: "memory",
			resources: patch.ContainerResources{
				Limits: common.Resource{
					Memory: ResourceLimit,
				},
				Requests: common.Resource{
					Memory: ResourceRequest,
				},
			},
		},
		{
			name: "ephemeral-storage",
			resources: patch.ContainerResources{
				Limits: common.Resource{
					EphemeralStorage: ResourceLimit,
				},
				Requests: common.Resource{
					EphemeralStorage: ResourceRequest,
				},
			},
		},
	}
	for _, scenario := range scenarios {
		t.Run(fmt.Sprintf("sets %s limits", scenario.name), func(t *testing.T) {
			obj := resourcedObj(nil)

			scenario.resources.Apply(obj)

			require.Equal(t, ResourceLimit, getResourceRequirement(obj, "limits")[scenario.name])
		})

		t.Run(fmt.Sprintf("sets %s requests", scenario.name), func(t *testing.T) {
			obj := resourcedObj(nil)

			scenario.resources.Apply(obj)

			require.Equal(t, ResourceRequest, getResourceRequirement(obj, "requests")[scenario.name])
		})

		t.Run(fmt.Sprintf("does not override when the %s limit when empty", scenario.name), func(t *testing.T) {
			obj := resourcedObj(map[string]any{"limits": map[string]any{scenario.name: ResourceLimit}})

			patch.ContainerResources{}.Apply(obj)

			require.Equal(t, ResourceLimit, getResourceRequirement(obj, "limits")[scenario.name])
		})

		t.Run(fmt.Sprintf("does not override when the %s request when empty", scenario.name), func(t *testing.T) {
			obj := resourcedObj(map[string]any{"requests": map[string]any{scenario.name: ResourceRequest}})

			patch.ContainerResources{}.Apply(obj)

			require.Equal(t, ResourceRequest, getResourceRequirement(obj, "requests")[scenario.name])
		})
	}
}

func getResourceRequirement(obj *unstructured.Unstructured, requirement string) map[string]interface{} {
	containers, _, _ := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers")
	limits, _, _ := unstructured.NestedMap(containers[0].(map[string]any), "resources", requirement)
	return limits
}

func resourcedObj(resources map[string]any) *unstructured.Unstructured {
	container := map[string]any{}
	if resources != nil {
		container["resources"] = resources
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"spec": map[string]any{
					"containers": []any{container},
				},
			},
		},
	}}
}

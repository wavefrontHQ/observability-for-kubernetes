package patch_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/wavefront/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/patch"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const ResourceLimit = "limit"
const ResourceRequest = "request"

func TestContainerResources(t *testing.T) {
	t.Run("apply-able to non-resourced objects", func(t *testing.T) {
		p := patch.ContainerResources(wf.Resources{
			Requests: wf.Resource{
				EphemeralStorage: "10Gi",
			},
		})
		actualObj := &unstructured.Unstructured{Object: map[string]any{}}
		expectedObj := actualObj.DeepCopy()

		p.Apply(actualObj)

		require.Equal(t, expectedObj, actualObj)
	})

	scenarios := []struct {
		name      string
		resources wf.Resources
	}{
		{
			name: "cpu",
			resources: wf.Resources{
				Limits: wf.Resource{
					CPU: ResourceLimit,
				},
				Requests: wf.Resource{
					CPU: ResourceRequest,
				},
			},
		},
		{
			name: "memory",
			resources: wf.Resources{
				Limits: wf.Resource{
					Memory: ResourceLimit,
				},
				Requests: wf.Resource{
					Memory: ResourceRequest,
				},
			},
		},
		{
			name: "ephemeral-storage",
			resources: wf.Resources{
				Limits: wf.Resource{
					EphemeralStorage: ResourceLimit,
				},
				Requests: wf.Resource{
					EphemeralStorage: ResourceRequest,
				},
			},
		},
	}
	for _, scenario := range scenarios {
		t.Run(fmt.Sprintf("sets %s limits", scenario.name), func(t *testing.T) {
			obj := resourcedObj(nil)

			patch.ContainerResources(scenario.resources).Apply(obj)

			require.Equal(t, ResourceLimit, getResourceRequirement(obj, "limits")[scenario.name])
		})

		t.Run(fmt.Sprintf("sets %s requests", scenario.name), func(t *testing.T) {
			obj := resourcedObj(nil)

			patch.ContainerResources(scenario.resources).Apply(obj)

			require.Equal(t, ResourceRequest, getResourceRequirement(obj, "requests")[scenario.name])
		})

		t.Run(fmt.Sprintf("does not override when the %s limit is empty", scenario.name), func(t *testing.T) {
			obj := resourcedObj(map[string]any{"limits": map[string]any{scenario.name: ResourceLimit}})

			patch.ContainerResources(wf.Resources{}).Apply(obj)

			require.Equal(t, ResourceLimit, getResourceRequirement(obj, "limits")[scenario.name])
		})

		t.Run(fmt.Sprintf("does not override when the %s request is empty", scenario.name), func(t *testing.T) {
			obj := resourcedObj(map[string]any{"requests": map[string]any{scenario.name: ResourceRequest}})

			patch.ContainerResources(wf.Resources{}).Apply(obj)

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

package patch_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/patch"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestByName(t *testing.T) {
	t.Run("applies patch to resource with matching name", func(t *testing.T) {
		p := patch.ByName{"some-resource": patch.ApplyFn(func(resource *unstructured.Unstructured) {
			resource.SetLabels(map[string]string{"foo": "bar"})
		})}
		obj1 := &unstructured.Unstructured{Object: map[string]any{
			"metadata": map[string]any{
				"name": "some-resource",
			},
		}}
		obj2 := &unstructured.Unstructured{Object: map[string]any{
			"metadata": map[string]any{
				"name": "some-other-resource",
			},
		}}

		p.Apply(obj1)
		p.Apply(obj2)

		require.Equal(t, "bar", obj1.GetLabels()["foo"])
		require.Empty(t, obj2.GetLabels()["foo"])
	})
}

package patch_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/patch"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestComposed(t *testing.T) {
	t.Run("applies patches in order", func(t *testing.T) {
		p := patch.Composed{
			patch.ApplyFn(func(resource *unstructured.Unstructured) {
				resource.SetLabels(map[string]string{"foo": "1"})
			}),
			patch.ApplyFn(func(resource *unstructured.Unstructured) {
				resource.SetLabels(map[string]string{"foo": "2"})
			}),
			nil,
		}
		obj := &unstructured.Unstructured{Object: map[string]any{}}

		p.Apply(obj)

		require.Equal(t, "2", obj.GetLabels()["foo"])
	})
}

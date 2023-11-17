package patch_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/patch"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var tolerationSec = int64(11)

var tolerationToAdd = corev1.Toleration{
	Key:               "foo",
	Operator:          corev1.TolerationOpEqual,
	Value:             "bar",
	Effect:            corev1.TaintEffectNoSchedule,
	TolerationSeconds: &tolerationSec,
}

var existingTolerationObj = map[string]any{
	"key":               "some-key",
	"operator":          string(corev1.TolerationOpEqual),
	"value":             "some-value",
	"effect":            string(corev1.TaintEffectNoExecute),
	"tolerationSeconds": int64(2),
}

var tolerationToAddObj = map[string]any{
	"key":               tolerationToAdd.Key,
	"operator":          string(tolerationToAdd.Operator),
	"value":             tolerationToAdd.Value,
	"effect":            string(tolerationToAdd.Effect),
	"tolerationSeconds": *tolerationToAdd.TolerationSeconds,
}

var tolerationOnlyKey = corev1.Toleration{
	Key: "foo",
}

var tolerationOnlyOp = corev1.Toleration{
	Operator: corev1.TolerationOpEqual,
}

var tolerationOnlyValue = corev1.Toleration{
	Value: "bar",
}

var tolerationOnlyEffect = corev1.Toleration{
	Effect: corev1.TaintEffectNoSchedule,
}

var tolerationOnlySeconds = corev1.Toleration{
	TolerationSeconds: &tolerationSec,
}

func TestTolerations(t *testing.T) {
	t.Run("can add tolerations to a resource with no tolerations", func(t *testing.T) {
		p := patch.Tolerations{Add: []corev1.Toleration{tolerationToAdd}}
		resource := &unstructured.Unstructured{Object: map[string]any{
			"spec": map[string]any{
				"template": map[string]any{
					"spec": map[string]any{},
				},
			},
		}}

		p.Apply(resource)

		tolerations, _, _ := unstructured.NestedSlice(resource.Object, "spec", "template", "spec", "tolerations")
		require.Len(t, tolerations, 1)
		toleration := tolerations[0].(map[string]any)
		require.Equal(t, "foo", toleration["key"], "toleration key")
		require.Equal(t, string(corev1.TolerationOpEqual), toleration["operator"], "toleration operator")
		require.Equal(t, "bar", toleration["value"], "toleration value")
		require.Equal(t, string(corev1.TaintEffectNoSchedule), toleration["effect"], "toleration effect")
	})

	t.Run("can add tolerations to a resource with existing tolerations", func(t *testing.T) {
		p := patch.Tolerations{Add: []corev1.Toleration{tolerationToAdd}}
		resource := &unstructured.Unstructured{Object: map[string]any{
			"spec": map[string]any{
				"template": map[string]any{
					"spec": map[string]any{
						"tolerations": []any{existingTolerationObj},
					},
				},
			},
		}}

		p.Apply(resource)

		tolerations, _, _ := unstructured.NestedSlice(resource.Object, "spec", "template", "spec", "tolerations")
		require.Len(t, tolerations, 2)
		toleration := tolerations[1].(map[string]any)
		require.Equal(t, "foo", toleration["key"], "toleration key")
		require.Equal(t, string(corev1.TolerationOpEqual), toleration["operator"], "toleration operator")
		require.Equal(t, "bar", toleration["value"], "toleration value")
		require.Equal(t, string(corev1.TaintEffectNoSchedule), toleration["effect"], "toleration effect")
	})

	t.Run("does not add duplicate tolerations", func(t *testing.T) {
		p := patch.Tolerations{Add: []corev1.Toleration{tolerationToAdd}}
		resource := &unstructured.Unstructured{Object: map[string]any{
			"spec": map[string]any{
				"template": map[string]any{
					"spec": map[string]any{
						"tolerations": []any{tolerationToAddObj},
					},
				},
			},
		}}

		p.Apply(resource)

		tolerations, _, _ := unstructured.NestedSlice(resource.Object, "spec", "template", "spec", "tolerations")
		require.Len(t, tolerations, 1)
	})

	t.Run("does not add empty keys", func(t *testing.T) {
		p := patch.Tolerations{Add: []corev1.Toleration{
			tolerationOnlyKey, tolerationOnlyOp, tolerationOnlyValue, tolerationOnlyEffect, tolerationOnlySeconds,
		}}
		resource := &unstructured.Unstructured{Object: map[string]any{
			"spec": map[string]any{
				"template": map[string]any{
					"spec": map[string]any{},
				},
			},
		}}

		p.Apply(resource)

		tolerations, _, _ := unstructured.NestedSlice(resource.Object, "spec", "template", "spec", "tolerations")
		require.Len(t, tolerations, 5)

		require.Equal(t, []string{"key"}, keys(tolerations[0].(map[string]any)))
		require.Equal(t, []string{"operator"}, keys(tolerations[1].(map[string]any)))
		require.Equal(t, []string{"value"}, keys(tolerations[2].(map[string]any)))
		require.Equal(t, []string{"effect"}, keys(tolerations[3].(map[string]any)))
		require.Equal(t, []string{"tolerationSeconds"}, keys(tolerations[4].(map[string]any)))
	})

	t.Run("does not change resources which do not have tolerations", func(t *testing.T) {
		p := patch.Tolerations{Add: []corev1.Toleration{tolerationToAdd}}
		resource := &unstructured.Unstructured{Object: map[string]any{"spec": map[string]any{}}}

		p.Apply(resource)

		_, exists, _ := unstructured.NestedSlice(resource.Object, "spec", "template", "spec", "tolerations")
		require.False(t, exists, "tolerations should not be added")
	})
}

func keys(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

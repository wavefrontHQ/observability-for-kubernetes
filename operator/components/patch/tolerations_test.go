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

var existingTolerationSec int64 = 2

var existingToleration = corev1.Toleration{
	Key:               "some-key",
	Operator:          corev1.TolerationOpEqual,
	Value:             "some-value",
	Effect:            corev1.TaintEffectNoExecute,
	TolerationSeconds: &existingTolerationSec,
}

var existingTolerationObj = map[string]any{
	"key":               existingToleration.Key,
	"operator":          string(existingToleration.Operator),
	"value":             existingToleration.Value,
	"effect":            string(existingToleration.Effect),
	"tolerationSeconds": *existingToleration.TolerationSeconds,
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
		requireTolerationExists(t, tolerationToAdd, tolerations)
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
		requireTolerationExists(t, tolerationToAdd, tolerations)
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

		keyCounts := map[string]int{
			"key":               0,
			"operator":          0,
			"value":             0,
			"effect":            0,
			"tolerationSeconds": 0,
		}
		for _, toleration := range tolerations {
			for _, k := range keys(toleration.(map[string]any)) {
				keyCounts[k] += 1
			}
		}

		for key, count := range keyCounts {
			require.Equalf(t, 1, count, "expect 1 tolerations with %s key but got %d", key, count)
		}
	})

	t.Run("does not change resources which do not have tolerations", func(t *testing.T) {
		p := patch.Tolerations{Add: []corev1.Toleration{tolerationToAdd}}
		resource := &unstructured.Unstructured{Object: map[string]any{"spec": map[string]any{}}}

		p.Apply(resource)

		_, exists, _ := unstructured.NestedSlice(resource.Object, "spec", "template", "spec", "tolerations")
		require.False(t, exists, "tolerations should not be added")
	})

	t.Run("can remove tolerations", func(t *testing.T) {
		p := patch.Tolerations{Remove: []corev1.Toleration{existingToleration}}
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
		require.Len(t, tolerations, 0)
	})
}

func requireTolerationExists(t *testing.T, expected corev1.Toleration, tolerationObjs []any) {
	t.Helper()
	for _, anyTolerationObj := range tolerationObjs {
		tolerationObj := anyTolerationObj.(map[string]any)
		if tolerationObj["key"] != expected.Key {
			continue
		}
		if tolerationObj["operator"] != string(expected.Operator) {
			continue
		}
		if tolerationObj["value"] != expected.Value {
			continue
		}
		if tolerationObj["effect"] != string(expected.Effect) {
			continue
		}
		if expected.TolerationSeconds != nil && tolerationObj["tolerationSeconds"] == nil {
			continue
		}
		if expected.TolerationSeconds == nil && tolerationObj["tolerationSeconds"] != nil {
			continue
		}
		if expected.TolerationSeconds != nil && tolerationObj["tolerationSeconds"] != nil && *expected.TolerationSeconds != tolerationObj["tolerationSeconds"].(int64) {
			continue
		}
		return
	}
	t.Fatalf("could not find toleration %#+v", expected)
}

func keys(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

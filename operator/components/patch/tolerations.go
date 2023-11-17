package patch

import (
	"bytes"
	"fmt"
	"sort"

	rc "github.com/wavefronthq/observability-for-kubernetes/operator/api/resourcecustomizations/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Tolerations rc.Tolerations

func (t Tolerations) Apply(resource *unstructured.Unstructured) {
	if !hasTemplateSpec(resource) {
		return
	}
	tolerations, _, _ := unstructured.NestedSlice(resource.Object, "spec", "template", "spec", "tolerations")
	tm := tolerationMap(tolerations)
	for _, toleration := range t.Add {
		obj := tolerationObj(toleration)
		lookupKey := tolerationLookupKey(obj)
		if _, exists := tm[lookupKey]; exists {
			continue
		}
		tm[lookupKey] = obj
	}
	for _, toleration := range t.Remove {
		obj := tolerationObj(toleration)
		delete(tm, tolerationLookupKey(obj))
	}
	_ = unstructured.SetNestedSlice(resource.Object, tolerationMapToSlice(tm), "spec", "template", "spec", "tolerations")
}

func hasTemplateSpec(resource *unstructured.Unstructured) bool {
	_, exists, _ := unstructured.NestedFieldNoCopy(resource.Object, "spec", "template", "spec")
	return exists
}

func tolerationMap(tolerations []any) map[string]map[string]any {
	tolerationExists := map[string]map[string]any{}
	for _, toleration := range tolerations {
		tolerationExists[tolerationLookupKey(toleration.(map[string]any))] = toleration.(map[string]any)
	}
	return tolerationExists
}

func tolerationMapToSlice(tm map[string]map[string]any) []any {
	tolerations := make([]any, 0, len(tm))
	for _, toleration := range tm {
		tolerations = append(tolerations, toleration)
	}
	return tolerations
}

func tolerationLookupKey(obj map[string]any) string {
	buf := bytes.NewBuffer(nil)
	for _, key := range keys(obj) {
		fmt.Fprintf(buf, ":%s=%v", key, obj[key])
	}
	return buf.String()
}

func tolerationObj(toleration v1.Toleration) map[string]any {
	obj := map[string]any{}
	if len(toleration.Key) > 0 {
		obj["key"] = toleration.Key
	}
	if len(toleration.Operator) > 0 {
		obj["operator"] = string(toleration.Operator)
	}
	if len(toleration.Value) > 0 {
		obj["value"] = toleration.Value
	}
	if len(toleration.Effect) > 0 {
		obj["effect"] = string(toleration.Effect)
	}
	if toleration.TolerationSeconds != nil {
		obj["tolerationSeconds"] = *toleration.TolerationSeconds
	}
	return obj
}

func keys(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

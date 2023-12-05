package result

import "k8s.io/apimachinery/pkg/runtime/schema"

type Aggregate map[schema.GroupVersionKind]Result

func (a Aggregate) HasErrors() bool {
	for _, result := range a {
		if result.IsError() {
			return true
		}
	}
	return false
}

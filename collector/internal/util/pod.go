package util

import (
	v1 "k8s.io/api/core/v1"
)

func IsStuckInTerminating(pod *v1.Pod) bool {
	return false
}

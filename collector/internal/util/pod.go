package util

import (
	v1 "k8s.io/api/core/v1"
)

func IsStuckInTerminating(pod *v1.Pod) bool {
	if pod.DeletionTimestamp == nil {
		return false
	}
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Terminated != nil && status.State.Terminated.Reason == "Error" {
			return true
		}
	}
	return false
}

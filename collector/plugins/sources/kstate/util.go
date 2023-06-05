package kstate

import v1 "k8s.io/api/core/v1"

func ConditionStatusFloat64(status v1.ConditionStatus) float64 {
	switch status {
	case v1.ConditionTrue:
		return 1.0
	case v1.ConditionFalse:
		return 0.0
	default:
		return -1.0
	}
}

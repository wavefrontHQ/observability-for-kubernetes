package kstate

import (
	"fmt"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
)

const (
	workloadStatusMetric = "workload.status"

	workloadNameTag      = "workload_name"
	workloadKindTag      = "workload_kind"
	workloadAvailableTag = "available"
	workloadDesiredTag   = "desired"

	workloadReady    = float64(1.0)
	workloadNotReady = float64(0.0)

	workloadKindPod         = "Pod"
	workloadKindCronJob     = "CronJob"
	workloadKindJob         = "Job"
	workloadKindDaemonSet   = "DaemonSet"
	workloadKindStatefulSet = "StatefulSet"
	workloadKindReplicaSet  = "ReplicaSet"
	workloadKindDeployment  = "Deployment"
)

func buildWorkloadStatusMetric(prefix string, status float64, ts int64, source string, tags map[string]string) wf.Metric {
	return metricPoint(prefix, workloadStatusMetric, status, ts, source, tags)
}

func buildWorkloadTags(kind string, name string, namespace string, desired int32, available int32, customTags map[string]string) map[string]string {
	tags := buildTags(workloadNameTag, name, namespace, customTags)
	tags[workloadKindTag] = kind
	tags[workloadAvailableTag] = fmt.Sprintf("%d", available)
	tags[workloadDesiredTag] = fmt.Sprintf("%d", desired)
	return tags
}

func getWorkloadStatus(desired, available int32) float64 {
	if available == desired {
		return workloadReady
	}
	return workloadNotReady
}

package kstate

import (
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
)

const (
	workloadStatusMetric = "workload.status"
	workloadNameTag      = "workload_name"
	workloadKindTag      = "workload_kind"

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

func buildWorkloadTags(kind string, name string, namespace string, customTags map[string]string) map[string]string {
	tags := buildTags(workloadNameTag, name, namespace, customTags)
	tags[workloadKindTag] = kind
	return tags
}

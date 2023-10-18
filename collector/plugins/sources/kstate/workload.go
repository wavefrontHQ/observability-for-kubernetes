package kstate

import (
	"fmt"
	s "strings"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/metrics"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
)

const (
	workloadStatusMetric = "workload.status"

	workloadNameTag = "workload_name"
	workloadKindTag = "workload_kind"

	workloadAvailableTag     = "available"
	workloadDesiredTag       = "desired"
	workloadFailedReasonTag  = "reason"
	workloadFailedMessageTag = "message"
	workloadTypeTag          = "type"

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

func buildWorkloadTags(kind string, name string, namespace string, desired int32, available int32, reason string, message string, customTags map[string]string) map[string]string {
	tags := buildTags(workloadNameTag, name, namespace, customTags)
	tags[workloadKindTag] = kind
	tags[workloadTypeTag] = metrics.MetricSetTypeWorkloadKind
	tags[workloadAvailableTag] = fmt.Sprint(available)
	tags[workloadDesiredTag] = fmt.Sprint(desired)
	if len(reason) > 0 {
		tags[workloadFailedReasonTag] = reason
	}
	if len(message) > 0 {
		tags[workloadFailedMessageTag] = sanitizeMessage(message)
	}
	return tags
}

func getWorkloadStatus(desired, available int32) float64 {
	if available == desired {
		return workloadReady
	}
	return workloadNotReady
}

func sanitizeMessage(message string) string {
	if s.ContainsAny(message, "\"") {
		message = s.ReplaceAll(message, "\"", "")
	}
	return message
}

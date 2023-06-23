package kstate

import (
	"fmt"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
)

const (
	workloadStatusMetric = "workload.status"
	workloadNameTag      = "workload_name"
	workloadTypeTag      = "workload_type"

	workloadReady    = float64(1.0)
	workloadNotReady = float64(0.0)
)

func buildWorkloadStatusMetric(prefix string, numberDesired float64, numberReady float64, ts int64, source string, tags map[string]string) wf.Metric {
	status := workloadNotReady
	if numberReady == numberDesired {
		status = workloadReady
	}

	readyTagValue := fmt.Sprintf("%d/%d", int32(numberReady), int32(numberDesired))
	tags["ready"] = readyTagValue

	return metricPoint(prefix, workloadStatusMetric, status, ts, source, tags)
}

func buildWorkloadTags(kind string, name string, namespace string, customTags map[string]string) map[string]string {
	tags := buildTags(workloadNameTag, name, namespace, customTags)
	tags[workloadTypeTag] = kind
	return tags
}

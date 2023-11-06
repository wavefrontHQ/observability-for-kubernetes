// Copyright 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package events

import (
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"
	v1 "k8s.io/api/core/v1"
)

const (
	CREATION = "Creation"
	RUNTIME  = "Runtime"
)

func annotateEvent(event *v1.Event, workloadCache util.WorkloadCache, clusterName, clusterUUID string) {
	if event.ObjectMeta.Annotations == nil {
		event.ObjectMeta.Annotations = map[string]string{}
	}
	event.ObjectMeta.Annotations["aria/cluster-name"] = clusterName
	event.ObjectMeta.Annotations["aria/cluster-uuid"] = clusterUUID

	if event.InvolvedObject.Kind == "Pod" {
		workloadName, workloadKind, nodeName := workloadCache.GetWorkloadForPodName(event.InvolvedObject.Name, event.InvolvedObject.Namespace)
		event.ObjectMeta.Annotations["aria/workload-name"] = workloadName
		event.ObjectMeta.Annotations["aria/workload-kind"] = workloadKind
		if len(nodeName) > 0 {
			event.ObjectMeta.Annotations["aria/node-name"] = nodeName
		}
		event.ObjectMeta.Annotations["aria/category"] = CREATION
	}
}

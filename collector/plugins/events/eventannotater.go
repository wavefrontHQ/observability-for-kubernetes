// Copyright 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package events

import v1 "k8s.io/api/core/v1"

func annotateEvent(e *v1.Event, er *EventRouter) {
	if e.ObjectMeta.Annotations == nil {
		e.ObjectMeta.Annotations = map[string]string{}
	}
	e.ObjectMeta.Annotations["aria/cluster-name"] = er.clusterName
	e.ObjectMeta.Annotations["aria/cluster-uuid"] = er.clusterUUID

	if e.InvolvedObject.Kind == "Pod" {
		workloadName, workloadKind, nodeName := er.workloadCache.GetWorkloadForPodName(e.InvolvedObject.Name, e.InvolvedObject.Namespace)
		e.ObjectMeta.Annotations["aria/workload-name"] = workloadName
		e.ObjectMeta.Annotations["aria/workload-kind"] = workloadKind
		if len(nodeName) > 0 {
			e.ObjectMeta.Annotations["aria/node-name"] = nodeName
		}
	}
}
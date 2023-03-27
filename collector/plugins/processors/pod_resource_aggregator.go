// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Copyright 2018-2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package processors

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/metrics"
	kube_api "k8s.io/api/core/v1"
	v1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/kubectl/pkg/util/resource"
)

type PodResourceAggregator struct {
	podLister v1listers.PodLister
}

func (aggregator *PodResourceAggregator) Name() string {
	return "resource_aggregator"
}

func (aggregator *PodResourceAggregator) Process(batch *metrics.Batch) (*metrics.Batch, error) {
	for batchKey, metricSet := range batch.Sets {
		if metricSetType, found := metricSet.Labels[metrics.LabelMetricSetType.Key]; !found || metricSetType != metrics.MetricSetTypePod {
			continue
		}

		podName, foundPodName := metricSet.Labels[metrics.LabelPodName.Key]
		ns, foundNs := metricSet.Labels[metrics.LabelNamespaceName.Key]
		if !foundPodName || !foundNs {
			log.Errorf("No namespace and/or pod name for resource %s in pod resource aggregator: %v", batchKey, metricSet.Labels)
			continue
		}

		pod, err := aggregator.getPod(ns, podName)
		if err != nil {
			delete(batch.Sets, batchKey)
			log.Debugf("Failed to get pod %s from cache for resource aggregator: %v", metrics.PodKey(ns, podName), err)
			continue
		}

		reqs, limits := resource.PodRequestsAndLimits(pod)

		for resourceName, req := range reqs {
			if resourceName == kube_api.ResourceCPU {
				metricSet.Values[resourceName.String()+"/request"] = intValue(req.MilliValue())
			} else if resourceName == kube_api.ResourceEphemeralStorage {
				metricSet.Values[metrics.MetricEphemeralStorageRequest.Name] = intValue(req.Value())
			} else {
				metricSet.Values[resourceName.String()+"/request"] = intValue(req.Value())
			}
		}

		for resourceName, limit := range limits {
			if resourceName == kube_api.ResourceCPU {
				metricSet.Values[resourceName.String()+"/limit"] = intValue(limit.MilliValue())
			} else if resourceName == kube_api.ResourceEphemeralStorage {
				metricSet.Values[metrics.MetricEphemeralStorageLimit.Name] = intValue(limit.Value())
			} else {
				metricSet.Values[resourceName.String()+"/limit"] = intValue(limit.Value())
			}
		}
	}

	return batch, nil
}

func (aggregator *PodResourceAggregator) getPod(namespace, name string) (*kube_api.Pod, error) {
	pod, err := aggregator.podLister.Pods(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	if pod == nil {
		return nil, fmt.Errorf("cannot find pod definition")
	}

	return pod, nil
}

func NewPodResourceAggregator(podLister v1listers.PodLister) *PodResourceAggregator {
	return &PodResourceAggregator{
		podLister: podLister,
	}
}

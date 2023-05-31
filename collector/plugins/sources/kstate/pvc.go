// Copyright 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package kstate

import (
	corev1 "k8s.io/api/core/v1"
	"reflect"
	"time"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"

	log "github.com/sirupsen/logrus"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
)

func pointsForPVC(item interface{}, transforms configuration.Transforms) []wf.Metric {
	pvc, ok := item.(*corev1.PersistentVolumeClaim)
	if !ok {
		log.Errorf("invalid type: %s", reflect.TypeOf(item).String())
		return nil
	}

	tags := buildTags("pvc", pvc.Name, pvc.Namespace, transforms.Tags)
	now := time.Now().Unix()

	var resourceStorage = pvc.Spec.Resources.Requests[corev1.ResourceStorage]

	return []wf.Metric{
		metricPoint(transforms.Prefix, "pvc.request.storage_bytes", float64(resourceStorage.Value()), now, transforms.Source, tags),
	}
}

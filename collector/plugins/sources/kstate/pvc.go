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
	persistentVolumeClaim, ok := item.(*corev1.PersistentVolumeClaim)
	if !ok {
		log.Errorf("invalid type: %s", reflect.TypeOf(item).String())
		return nil
	}

	tags := buildTags("pvc", persistentVolumeClaim.Name, persistentVolumeClaim.Namespace, transforms.Tags)
	now := time.Now().Unix()

	var resourceStorage = persistentVolumeClaim.Spec.Resources.Requests[corev1.ResourceStorage]
	rsValue := float64(resourceStorage.Value())

	log.Println("Resource storage: " + resourceStorage.String())
	log.Printf("Resource storage value: %x\n", rsValue)
	log.Println("PVC Phase: " + persistentVolumeClaim.Status.Phase)

	return []wf.Metric{
		metricPoint(transforms.Prefix, "pvc.request.storage_bytes", rsValue, now, transforms.Source, tags),
	}
}

// TODO write tests

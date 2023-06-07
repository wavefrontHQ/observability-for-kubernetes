// Copyright 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package events

import (
	"time"

	"github.com/wavefronthq/wavefront-sdk-go/event"
	v1 "k8s.io/api/core/v1"
)

type EventSink interface {
	ExportEvent(*Event)
}

type Event struct {
	Message     string            `json:"-"`
	Ts          time.Time         `json:"-"`
	Host        string            `json:"-"`
	Tags        map[string]string `json:"-"`
	Options     []event.Option    `json:"-"`
	ClusterName string            `json:"clusterName,omitempty"`
	ClusterUUID string            `json:"clusterUUID,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	v1.Event    `json:",inline"`
}

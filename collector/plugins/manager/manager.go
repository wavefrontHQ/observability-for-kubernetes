// Based on https://github.com/kubernetes-retired/heapster/blob/master/metrics/manager/manager.go
// Diff against master for changes to the original code.

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

package manager // TODO can we rename this to flush_manager?

import (
	"time"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/metrics"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/sinks"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/sources"

	log "github.com/sirupsen/logrus"
)

// FlushManager deals with data push
type FlushManager interface {
	Start()
	Stop()
}

type flushManagerImpl struct {
	processors    []metrics.Processor
	sink          sinks.Sink
	flushInterval time.Duration
	ticker        *time.Ticker
	stopChan      chan struct{}
}

// NewFlushManager crates a new PushManager
func NewFlushManager(processors []metrics.Processor,
	sink sinks.Sink, flushInterval time.Duration) (FlushManager, error) {
	manager := flushManagerImpl{
		processors:    processors,
		sink:          sink,
		flushInterval: flushInterval,
		stopChan:      make(chan struct{}),
	}

	return &manager, nil
}

func (rm *flushManagerImpl) Start() {
	rm.ticker = time.NewTicker(rm.flushInterval)
	go rm.run()
}

func (rm *flushManagerImpl) run() {
	for {
		select {
		case <-rm.ticker.C:
			go rm.push()
		case <-rm.stopChan:
			rm.ticker.Stop()
			rm.sink.Stop()
			return
		}
	}
}

func (rm *flushManagerImpl) Stop() {
	rm.stopChan <- struct{}{}
}

func (rm *flushManagerImpl) push() {
	dataBatches := sources.Manager().GetPendingMetrics()
	combinedBatch := &metrics.Batch{}

	for _, data := range dataBatches {
		combineMetricSets(data, combinedBatch)
	}

	// process the combined metric sets
	for _, p := range rm.processors {
		processedBatch, err := p.Process(combinedBatch)
		if err == nil {
			combinedBatch = processedBatch
		} else {
			log.Errorf("Error in processor: %v", err)
			return
		}
	}

	rm.sink.Export(combinedBatch)
}

func combineMetricSets(src, dst *metrics.Batch) {
	// use the most recent timestamp for the shared batch
	dst.Timestamp = src.Timestamp
	if dst.Sets == nil {
		dst.Sets = make(map[metrics.ResourceKey]*metrics.Set)
	}
	for k, v := range src.Sets {
		dst.Sets[k] = v
	}
	dst.Metrics = append(dst.Metrics, src.Metrics...)
}

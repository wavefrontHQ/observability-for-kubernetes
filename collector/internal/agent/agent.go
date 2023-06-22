// Copyright 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	log "github.com/sirupsen/logrus"

	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/discovery"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/events"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/manager"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/sources"
)

type Agent struct {
	flushManager     manager.FlushManager
	discoveryManager *discovery.Manager
	eventRouter      *events.EventRouter
}

func NewAgent(fm manager.FlushManager, dm *discovery.Manager, er *events.EventRouter) *Agent {
	return &Agent{
		flushManager:     fm,
		discoveryManager: dm,
		eventRouter:      er,
	}
}

func (a *Agent) Start() {
	log.Infof("Starting agent")
	a.flushManager.Start()
	if a.discoveryManager != nil {
		a.discoveryManager.Start()
	}

	if a.eventRouter != nil {
		log.Infof("Starting Events collector")
		a.eventRouter.Start()
		log.Infof("Done Starting Events collector")
	}
}

func (a *Agent) Stop() {
	log.Infof("Stopping agent")
	a.flushManager.Stop()
	if a.discoveryManager != nil {
		a.discoveryManager.Stop()
	}

	if a.eventRouter != nil {
		log.Infof("Stopping Events collector")
		a.eventRouter.Stop()
		log.Infof("Done Stopping Events collector")
	}

	sources.Manager().StopProviders()
	log.Infof("Agent stopped")
}

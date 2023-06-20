package sinks

import (
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/events"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/metrics"
)

type Sink interface {
	Name() string
	Stop()
	metrics.Sink
	events.EventSink
}

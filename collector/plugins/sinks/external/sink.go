package external

import (
	"bytes"
	"encoding/json"
	"net/http"

	gm "github.com/rcrowley/go-metrics"

	log "github.com/sirupsen/logrus"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/events"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/metrics"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/sinks"
)

type ExternalSink struct {
	ClusterName         string
	ClusterUUID         string
	externalEndpointURL string
	eventsEnabled       bool
}

func (sink *ExternalSink) Name() string {
	return "k8s_events_sink"
}

func (sink *ExternalSink) Stop() {}

func (sink *ExternalSink) Export(_ *metrics.Batch) {}

func (sink *ExternalSink) ExportEvent(ev *events.Event) {
	if !sink.eventsEnabled {
		return
	}
	ev.ClusterName = sink.ClusterName
	ev.ClusterUUID = sink.ClusterUUID

	b, _ := json.Marshal(ev)
	req, err := http.NewRequest("POST", sink.externalEndpointURL, bytes.NewBuffer(b))
	if err != nil {
		gm.GetOrRegisterCounter("wavefront.events.errors.count", gm.DefaultRegistry).Inc(1)
		log.WithFields(log.Fields{
			"message": ev.Message,
			"error":   err,
		}).Error("[sampled error] error creating request to external event endpoint")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	_, err = http.DefaultClient.Do(req)
	if err != nil {
		gm.GetOrRegisterCounter("wavefront.events.errors.count", gm.DefaultRegistry).Inc(1)
		log.WithFields(log.Fields{
			"message": ev.Message,
			"error":   err,
		}).Error("[sampled error] error sending event to external event endpoint")
	} else {
		log.WithField("name", sink.Name()).Debug("Events push complete")
	}
}

func NewExternalSink(cfg configuration.SinkConfig) (sinks.Sink, error) {
	return &ExternalSink{
		ClusterName:         cfg.ClusterName,
		ClusterUUID:         cfg.ClusterUUID,
		externalEndpointURL: cfg.ExternalEndpointURL,
		eventsEnabled:       *cfg.EnableEvents,
	}, nil
}

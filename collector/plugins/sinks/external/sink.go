package external

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/events"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/metrics"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/sinks"
)

type ExternalSink struct {
	eventsEnabled             bool
	externalEndpointURL       string
	externalEndpointAccessKey string
	logger                    *log.Entry
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

	b, err := json.Marshal(ev)
	if err != nil {
		sink.logError("error marshalling event JSON", err)
		return
	}

	req, err := http.NewRequest("POST", sink.externalEndpointURL, bytes.NewBuffer(b))
	if err != nil {
		sink.logError("error creating HTTP request", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sink.externalEndpointAccessKey))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		sink.logError("error sending event", err)
		return
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode > 299 {
		sink.logError("error sending event", fmt.Errorf("endpoint returned status code %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode)))
		return
	}
	sink.logger.Debug("Events push complete")
}

func (sink *ExternalSink) logError(message string, err error) {
	sink.logger.WithField("error", err).Error(message)
}

func NewExternalSink(cfg configuration.SinkConfig) (sinks.Sink, error) {
	logger := log.WithField("sink", "k8s_events")
	return &ExternalSink{
		eventsEnabled:             *cfg.EnableEvents,
		externalEndpointURL:       cfg.ExternalEndpointURL,
		externalEndpointAccessKey: cfg.ExternalEndpointAccessKey,
		logger:                    logger,
	}, nil
}

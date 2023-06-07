package events

import (
	"bytes"
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/events"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/metrics"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/sinks"
)

type k8sEventSink struct {
	ClusterName               string
	ClusterUUID               string
	eventsExternalEndpointURL string
}

func (sink *k8sEventSink) Name() string {
	return "k8s_events_sink"
}

func (sink *k8sEventSink) Stop() {
}

func (sink *k8sEventSink) Export(batch *metrics.Batch) {
}

func (sink *k8sEventSink) ExportEvent(ev *events.Event) {
	ev.ClusterName = sink.ClusterName
	ev.ClusterUUID = sink.ClusterUUID

	// TODO: don't ignore errors. Log them.
	b, _ := json.Marshal(ev)
	req, _ := http.NewRequest("POST", sink.eventsExternalEndpointURL, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "text/plain")
	//fmt.Printf("TEST:: token: " + util.GetToken())
	//req.Header.Set("Authorization", "Bearer " + util.GetToken())

	client := &http.Client{}
	_, err := client.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"message": ev.Message,
			"error":   err,
		}).Errorf("%s %s", "[sampled error]", "error sending event to external event endpoint")
	}
}

func NewK8sEventsOnlySink(cfg configuration.SinkConfig) (sinks.Sink, error) {
	return &k8sEventSink{
		ClusterName:               cfg.ClusterName,
		ClusterUUID:               cfg.ClusterUUID,
		eventsExternalEndpointURL: cfg.EventsExternalEndpointURL,
	}, nil
}

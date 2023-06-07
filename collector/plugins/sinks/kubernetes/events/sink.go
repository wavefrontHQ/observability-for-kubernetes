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
	"github.com/wavefronthq/wavefront-sdk-go/event"
)

type k8sEventsink struct {
	ClusterName               string
	eventsExternalEndpointURL string
}

func (sink *k8sEventsink) Name() string {
	return "k8s_events_sink"
}

func (sink *k8sEventsink) Stop() {
}

func (sink *k8sEventsink) Export(batch *metrics.Batch) {
}

func (sink *k8sEventsink) ExportEvent(ev *events.Event) {
	ev.Options = append(ev.Options, event.Annotate("cluster", sink.ClusterName))
	ev.ClusterName = sink.ClusterName

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

func NewK8sEventsOnlySink(cfg configuration.WavefrontSinkConfig) (sinks.Sink, error) {
	return &k8sEventsink{ClusterName: cfg.ClusterName, eventsExternalEndpointURL: cfg.EventsExternalEndpointURL}, nil
}

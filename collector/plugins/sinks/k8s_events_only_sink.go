package sinks

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/events"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/metrics"
	"github.com/wavefronthq/wavefront-sdk-go/event"
)

type k8sEventsOnlySink struct {
	ClusterName               string
	eventsExternalEndpointURL string
}

func (sink *k8sEventsOnlySink) Name() string {
	return "k8s_events_only_sink"
}

func (sink *k8sEventsOnlySink) Stop() {
}

func (sink *k8sEventsOnlySink) Export(batch *metrics.Batch) {
}

func (sink *k8sEventsOnlySink) ExportEvent(ev *events.Event) {
	ev.Options = append(ev.Options, event.Annotate("cluster", sink.ClusterName))
	ev.ClusterName = sink.ClusterName

	b, _ := json.Marshal(ev)
	req, _ := http.NewRequest("POST", sink.eventsExternalEndpointURL, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "text/plain")
	//fmt.Printf("TEST:: token: " + util.GetToken())
	//req.Header.Set("Authorization", "Bearer " + util.GetToken())

	client := &http.Client{}
	client.Do(req)
	//if err != nil {
	//	sink.logVerboseError(log.Fields{
	//		"message": ev.Message,
	//		"error":   err,
	//	}, "error sending event to external event endpoint")
	//}
}

func NewK8sEventsOnlySink(cfg configuration.WavefrontSinkConfig) (Sink, error) {
	return &k8sEventsOnlySink{ClusterName: cfg.ClusterName, eventsExternalEndpointURL: cfg.EventsExternalEndpointURL}, nil
}

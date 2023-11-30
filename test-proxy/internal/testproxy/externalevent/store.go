package externalevent

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/wavefronthq/observability-for-kubernetes/test-proxy/internal/broadcaster"
	"github.com/wavefronthq/wavefront-sdk-go/event"
	v1 "k8s.io/api/core/v1"
)

var (
	expectedEventCategories = []string{
		"Creation.ImagePullBackOff",
	}
)

type Event struct {
	Message  string            `json:"-"`
	Ts       time.Time         `json:"-"`
	Host     string            `json:"-"`
	Tags     map[string]string `json:"-"`
	Options  []event.Option    `json:"-"`
	v1.Event `json:",inline"`
}

type AnnotationName = string
type TagName = string

type Store struct {
	mu      sync.Mutex
	results results
}

type results struct {
	EventCount int

	BadEventJSONs            []string
	ReceivedEventsByCategory map[string][]*Event
	MissingEventCategories   []string
	MissingFields            map[string][]*Event
	FirstTimestampsMissing   []*Event
	LastTimestampsInvalid    []*Event
}

func NewStore() *Store {
	return &Store{
		results: results{
			MissingFields:            map[string][]*Event{},
			ReceivedEventsByCategory: map[string][]*Event{},
		},
	}
}

func (s *Store) MarshalJSON() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.results.MissingEventCategories = make([]string, 0)
	for _, expectedCategory := range expectedEventCategories {
		if _, ok := s.results.ReceivedEventsByCategory[expectedCategory]; !ok {
			s.results.MissingEventCategories = append(s.results.MissingEventCategories, expectedCategory)
		}
	}

	return json.Marshal(s.results)
}

func (s *Store) Subscribe(externalEvents *broadcaster.Broadcaster[[]byte]) {
	eventJSONs, _ := externalEvents.Subscribe()
	go func() {
		for eventJSON := range eventJSONs {
			var event Event
			if err := json.Unmarshal(eventJSON, &event); err != nil {
				log.Error(err.Error())
				log.Error(eventJSON)
				s.RecordBadJSON(eventJSON)
				continue
			}
			s.Record(&event)
		}
	}()
}

func (s *Store) RecordBadJSON(eventJSON []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results.RecordBadLine(eventJSON)
}

func (s *Store) Record(event *Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results.Record(event)
}

func (r *results) Record(event *Event) {
	r.EventCount++

	if event.Event.ObjectMeta.Annotations["aria/cluster-name"] == "" {
		r.MissingFields["aria/cluster-name"] = append(r.MissingFields["aria/cluster-name"], event)
	}

	if event.Event.ObjectMeta.Annotations["aria/cluster-uuid"] == "" {
		r.MissingFields["aria/cluster-uuid"] = append(r.MissingFields["aria/cluster-uuid"], event)
	}

	if event.Event.ObjectMeta.Annotations["aria/category"] == "" {
		r.MissingFields["aria/category"] = append(r.MissingFields["aria/category"], event)
	}

	if event.Event.ObjectMeta.Annotations["aria/subcategory"] == "" {
		r.MissingFields["aria/subcategory"] = append(r.MissingFields["aria/subcategory"], event)
	}

	r.recordEventCategory(event)

	if event.Event.ObjectMeta.Name == "" {
		r.MissingFields["metadata.name"] = append(r.MissingFields["metadata.name"], event)
	}

	if event.Event.ObjectMeta.UID == "" {
		r.MissingFields["metadata.uid"] = append(r.MissingFields["metadata.uid"], event)
	}

	if event.Event.InvolvedObject.UID == "" {
		r.MissingFields["involvedObject.uid"] = append(r.MissingFields["involvedObject.uid"], event)
	}

	if event.Event.InvolvedObject.Name == "" {
		r.MissingFields["involvedObject.name"] = append(r.MissingFields["involvedObject.name"], event)
	}

	if event.Event.InvolvedObject.Kind == "" {
		r.MissingFields["involvedObject.kind"] = append(r.MissingFields["involvedObject.kind"], event)
	} else if event.Event.InvolvedObject.Kind == "Pod" {
		if event.Event.ObjectMeta.Annotations["aria/workload-kind"] == "" {
			r.MissingFields["aria/workload-kind"] = append(r.MissingFields["aria/workload-kind"], event)
		}
		if event.Event.ObjectMeta.Annotations["aria/workload-name"] == "" {
			r.MissingFields["aria/workload-name"] = append(r.MissingFields["aria/workload-name"], event)
		}
	}
	if event.Event.Reason == "" {
		r.MissingFields["reason"] = append(r.MissingFields["reason"], event)
	} else if event.Event.Reason != "FailedScheduling" && event.Event.InvolvedObject.Kind == "Pod" {
		if event.Event.ObjectMeta.Annotations["aria/node-name"] == "" {
			r.MissingFields["aria/node-name"] = append(r.MissingFields["aria/node-name"], event)
		}
	}

	if event.Type == "Normal" && event.Reason != "BackOff" {
		r.MissingFields["unexpected-field"] = append(r.MissingFields["unexpected-field"], event)
	}

	if event.Event.Message == "" {
		r.MissingFields["message"] = append(r.MissingFields["message"], event)
	}

	if event.Event.Source.Component == "" {
		r.MissingFields["source.component"] = append(r.MissingFields["source.component"], event)
	}

	if event.Event.FirstTimestamp.IsZero() {
		r.FirstTimestampsMissing = append(r.FirstTimestampsMissing, event)
	}

	if !lastTimestampIsValid(event) {
		r.LastTimestampsInvalid = append(r.LastTimestampsInvalid, event)
	}
}

func (r *results) recordEventCategory(event *Event) {
	eventCategory := fmt.Sprintf("%s.%s", event.ObjectMeta.Annotations["aria/category"], event.ObjectMeta.Annotations["aria/subcategory"])
	r.ReceivedEventsByCategory[eventCategory] = append(r.ReceivedEventsByCategory[eventCategory], event)
}

func lastTimestampIsValid(event *Event) bool {
	return event.Event.LastTimestamp.Compare(event.Event.FirstTimestamp.Time) >= 0 || event.Event.LastTimestamp.IsZero()
}

func (r *results) RecordBadLine(eventJSON []byte) {
	r.BadEventJSONs = append(r.BadEventJSONs, string(eventJSON))
}

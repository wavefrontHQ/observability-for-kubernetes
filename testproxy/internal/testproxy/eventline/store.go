package eventline

import (
	"encoding/json"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/wavefronthq/observability-for-kubernetes/testproxy/internal/broadcaster"
)

type AnnotationName = string
type TagName = string

type Store struct {
	mu      sync.Mutex
	results results
}

func NewStore() *Store {
	return &Store{
		results: results{
			MissingAnnotations:    map[AnnotationName][]*Event{},
			UnexpectedAnnotations: map[AnnotationName][]*Event{},
			MissingTags:           map[TagName][]*Event{},
		},
	}
}

func (s *Store) MarshalJSON() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return json.Marshal(s.results)
}

func (s *Store) Subscribe(proxylines *broadcaster.Broadcaster[string]) {
	lines, _ := proxylines.Subscribe()
	go func() {
		for line := range lines {
			if !strings.HasPrefix(line, "@Event") {
				continue
			}
			event, err := Parse(line)
			if err != nil {
				log.Error(err.Error())
				log.Error(line)
				s.RecordBadLine(line)
				continue
			}
			s.Record(event)
		}
	}()
}

func (s *Store) RecordBadLine(line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results.RecordBadLine(line)
}

func (s *Store) Record(event *Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results.Record(event)
}

type results struct {
	EventCount int

	BadEventLines []string

	ZeroStartMillis []*Event

	MissingAnnotations map[AnnotationName][]*Event

	UnexpectedAnnotations map[AnnotationName][]*Event

	MissingTags map[TagName][]*Event
}

func (r *results) Record(event *Event) {
	r.EventCount++

	if event.Start == "0" || event.Start == "" {
		r.ZeroStartMillis = append(r.ZeroStartMillis, event)
	}

	r.expectAnnotationNotMissing("host", event)
	r.expectAnnotationNotMissing("cluster", event)
	if event.Annotations["type"] == "Normal" {
		r.UnexpectedAnnotations["type"] = append(r.UnexpectedAnnotations["type"], event)
	}

	r.expectTagNotMissing("namespace", event)
	r.expectTagNotMissing("kind", event)
	r.expectTagNotMissing("reason", event)
	r.expectTagNotMissing("component", event)

	kind := strings.ToLower(event.Tags["kind"])
	if kind == "pod" && event.Tags["pod_name"] == "" {
		r.MissingTags["pod_name"] = append(r.MissingTags["pod_name"], event)
	} else if event.Tags["resource_name"] == "" {
		r.MissingTags["resource_name"] = append(r.MissingTags["resource_name"], event)
	}
}

func (r *results) expectAnnotationNotMissing(key string, event *Event) {
	if event.Annotations[key] == "" {
		r.MissingAnnotations[key] = append(r.MissingAnnotations[key], event)
	}
}

func (r *results) expectTagNotMissing(key string, event *Event) {
	if event.Tags[key] == "" {
		r.MissingTags[key] = append(r.MissingTags[key], event)
	}
}

func (r *results) RecordBadLine(line string) {
	r.BadEventLines = append(r.BadEventLines, line)
}

package metricline

import (
	"fmt"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/broadcaster"
)

type Store struct {
	metrics     []*Metric
	metricsMu   *sync.Mutex
	badMetrics  []string
	badMetricMu *sync.Mutex
}

func NewStore() *Store {
	return &Store{
		metrics:     make([]*Metric, 0, 1024),
		metricsMu:   &sync.Mutex{},
		badMetrics:  make([]string, 0, 1024),
		badMetricMu: &sync.Mutex{},
	}
}

func (s *Store) Metrics() []*Metric {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()
	cpy := make([]*Metric, len(s.metrics))
	copy(cpy, s.metrics)
	return cpy
}

func (s *Store) BadMetrics() []string {
	s.badMetricMu.Lock()
	defer s.badMetricMu.Unlock()
	cpy := make([]string, len(s.badMetrics))
	copy(cpy, s.badMetrics)
	return cpy
}

func (s *Store) LogMetric(metric *Metric) {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()
	s.metrics = append(s.metrics, metric)
}

func (s *Store) LogBadMetric(metric string) {
	s.badMetricMu.Lock()
	defer s.badMetricMu.Unlock()
	s.badMetrics = append(s.badMetrics, metric)
}

func (s *Store) Subscribe(b *broadcaster.Broadcaster[string]) {
	lines, _ := b.Subscribe()
	go func() {
		for line := range lines {
			if line == "" {
				continue
			}
			if isEvent(line) {
				continue
			}
			metric, err := Parse(line)
			if err != nil {
				log.Error(err.Error())
				log.Error(line)
				s.LogBadMetric(line)
				continue
			}
			if len(metric.Tags) > 20 {
				log.Error(fmt.Sprintf("[WF-410: Too many point tags (%d, max 20): %s %#v", len(metric.Tags), metric.Name, metric.Tags))
				continue
			}
			log.Debugf("%#v", metric)
			s.LogMetric(metric)
		}
	}()
}

func isEvent(line string) bool {
	return strings.HasPrefix(line, "@Event")
}

func isHistogram(line string) bool {
	return strings.HasPrefix(line, "!M") || strings.HasPrefix(line, "!H") || strings.HasPrefix(line, "!D")
}

package metricline

import "sync"

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

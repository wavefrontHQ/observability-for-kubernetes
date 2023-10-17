package main

import (
	"sync"
	"time"

	prom "github.com/prometheus/client_model/go"
)

const UnknownTable = "unknown"

type StatsStore struct {
	samplePeriodMinutes int
	knownTables         []string
	tableRates          map[string]*NumberStore[BytesPerSecond]
	tableBytes          map[string]*NumberStore[Bytes]
	tableRowBatches     map[string]*NumberStore[int]
	tableCountStore     *NumberStore[int]
}

func NewStatsStore(measureMinutes int, knownTables []string) *StatsStore {
	return &StatsStore{
		samplePeriodMinutes: measureMinutes,
		knownTables:         knownTables,
		tableRates:          map[string]*NumberStore[BytesPerSecond]{},
		tableBytes:          map[string]*NumberStore[Bytes]{},
		tableRowBatches:     map[string]*NumberStore[int]{},
		tableCountStore:     NewNumberStore[int](measureMinutes),
	}
}

func (s *StatsStore) normalizedTableName(tableName string) string {
	for _, knownTable := range s.knownTables {
		if knownTable == tableName {
			return tableName
		}
	}
	return UnknownTable
}

func (s *StatsStore) Record(now time.Time, metricsFamilies map[string]*prom.MetricFamily) {
	metricsFamily := metricsFamilies[retentionTimeNsMetricName]
	s.tableCountStore.Record(now, len(metricsFamily.GetMetric()))
	for _, retentionNs := range metricsFamily.GetMetric() {
		tableName := LabelValue(retentionNs, "name")
		retention := retentionNs.GetGauge().GetValue() / float64(time.Second)
		if retention == 0.0 {
			continue
		}
		coldBytes := FindMetricByTableName(metricsFamilies[colBytesMetricName].GetMetric(), tableName).GetGauge().GetValue()
		hotBytes := FindMetricByTableName(metricsFamilies[hotBytesMetricName].GetMetric(), tableName).GetGauge().GetValue()
		rowBatches := FindMetricByTableName(metricsFamilies[tableNumBatchesMetricName].GetMetric(), tableName).GetGauge().GetValue()
		bytesPerSecond := (hotBytes + coldBytes) / retention
		s.TableRate(tableName).Record(now, bytesPerSecond)
		s.TableBytes(tableName).Record(now, Bytes(hotBytes+coldBytes))
		s.TableRowBatches(tableName).Record(now, int(rowBatches))
	}
}

func (s *StatsStore) TableRate(tableName string) *NumberStore[BytesPerSecond] {
	tableName = s.normalizedTableName(tableName)
	if s.tableRates[tableName] == nil {
		s.tableRates[tableName] = NewNumberStore[BytesPerSecond](s.samplePeriodMinutes)
	}
	return s.tableRates[tableName]
}

func (s *StatsStore) TableCount() *NumberStore[int] {
	return s.tableCountStore
}

func (s *StatsStore) TableBytes(tableName string) *NumberStore[Bytes] {
	tableName = s.normalizedTableName(tableName)
	if s.tableBytes[tableName] == nil {
		s.tableBytes[tableName] = NewNumberStore[Bytes](s.samplePeriodMinutes)
	}
	return s.tableBytes[tableName]
}

func (s *StatsStore) TableRowBatches(tableName string) *NumberStore[int] {
	tableName = s.normalizedTableName(tableName)
	if s.tableRowBatches[tableName] == nil {
		s.tableRowBatches[tableName] = NewNumberStore[Bytes](s.samplePeriodMinutes)
	}
	return s.tableRowBatches[tableName]
}

type Number interface {
	~int | ~uint | ~float64
}

type NumberStore[V Number] struct {
	mu               sync.Mutex
	retentionMinutes int
	index            int
	observations     [][]observation[V]
}

func NewNumberStore[V Number](retentionMinutes int) *NumberStore[V] {
	return &NumberStore[V]{
		retentionMinutes: retentionMinutes,
		index:            0,
		observations:     make([][]observation[V], retentionMinutes),
	}
}

type observation[V Number] struct {
	At    time.Time
	Value V
}

func (s *NumberStore[V]) Record(at time.Time, value V) {
	s.mu.Lock()
	defer s.mu.Unlock()
	minute := at.Unix() / 60
	if minute != s.lastMinute() {
		s.index = (s.index + 1) % len(s.observations)
		s.observations[s.index] = s.observations[s.index][:0]
	}
	s.observations[s.index] = append(s.observations[s.index], observation[V]{
		At:    at,
		Value: value,
	})
}

func (s *NumberStore[V]) each(visit func(time.Time, V)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, obs := range s.observations {
		for _, ob := range obs {
			visit(ob.At, ob.Value)
		}
	}
}

func (s *NumberStore[V]) lastMinute() int64 {
	if len(s.observations[s.index]) == 0 {
		return 0
	}
	return s.observations[s.index][0].At.Unix() / 60
}

func (s *NumberStore[V]) MaxValue() V {
	var maxValue V
	s.each(func(_ time.Time, value V) {
		if value > maxValue {
			maxValue = value
		}
	})
	return maxValue
}

func (s *NumberStore[V]) Sum() V {
	var sum V
	s.each(func(_ time.Time, value V) {
		sum += value
	})
	return sum
}

func MaxNumber[T Number](a, b T) T {
	if b > a {
		return b
	}
	return a
}

func ScaleDuration(s float64, d time.Duration) time.Duration {
	return time.Duration(s * float64(d))
}

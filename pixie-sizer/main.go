package main

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/nats-io/nats.go"
	prom "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"px.dev/pixie/src/vizier/messages/messagespb"
)

var natsServer = "pl-nats"
var clientTLSCertFile = ""
var clientTLSKeyFile = ""
var clientTLSCAFile = ""

func init() {
	SetFromEnv("PS_NATS_SERVER", &natsServer)
	SetFromEnv("PL_CLIENT_TLS_CERT", &clientTLSCertFile)
	SetFromEnv("PL_CLIENT_TLS_KEY", &clientTLSKeyFile)
	SetFromEnv("PL_TLS_CA_CERT", &clientTLSCAFile)
}

func main() {
	natsConn, err := nats.Connect(natsServer, nats.ClientCert(clientTLSCertFile, clientTLSKeyFile), nats.RootCAs(clientTLSCAFile))
	if err != nil {
		log.Fatalf("error creating nats connection: %s", err)
	}
	natsMsgs := make(chan *nats.Msg)
	subscription, err := natsConn.ChanSubscribe("Metrics", natsMsgs)
	if err != nil {
		log.Fatalf("error creating nats subscriptions to Metrics topic: %s", err)
	}
	defer subscription.Unsubscribe()
	const retentionMinutes = 5
	httpTableSizeStore := NewValueStore[Bytes](retentionMinutes)
	otherTableSizeStore := NewValueStore[Bytes](retentionMinutes)
	tableCountStore := NewValueStore[uint](retentionMinutes)
	go func() {
		const percentile = 0.90
		for {
			time.Sleep(retentionMinutes * time.Minute)
			tableCount := tableCountStore.LatestValue()
			tableStoreLimit := CalculatePEMTableStoreLimit(
				tableCount,
				otherTableSizeStore.PercentileValue(percentile),
				0,
				0,
				httpTableSizeStore.PercentileValue(percentile),
			)
			log.Printf("recommended table size for %d tables: %s", tableCount, tableStoreLimit)
		}
	}()
	for natsMsg := range natsMsgs {
		var metricsMsg messagespb.MetricsMessage
		err := proto.Unmarshal(natsMsg.Data, &metricsMsg)
		if err != nil {
			log.Fatalf("invalid metrics message: %s", err)
		}
		if !strings.HasPrefix(metricsMsg.GetPodName(), "vizier-pem") {
			continue
		}
		log.Println("received metrics from pod", metricsMsg.GetPodName())
		buf := bytes.NewReader([]byte(metricsMsg.GetPromMetricsText()))
		metricsFamilies, err := (&expfmt.TextParser{}).TextToMetricFamilies(buf)
		if err != nil {
			log.Fatalf("invalid prometheus metrics: %s", err)
		}
		metricFamily := metricsFamilies["min_time"]
		maxOtherBytesPerSecond := 0.0
		maxHTTPBytesPerSecond := 0.0
		tableCount := uint(len(metricFamily.GetMetric()))
		for _, retentionNs := range metricFamily.GetMetric() {
			tableName := LabelValue(retentionNs, "name")
			retention := retentionNs.GetGauge().GetValue() / float64(time.Second)
			if retention == 0.0 {
				continue
			}
			coldBytes := FindMetricByTableName(metricsFamilies["table_cold_bytes"].GetMetric(), tableName).GetGauge().GetValue()
			hotBytes := FindMetricByTableName(metricsFamilies["table_hot_bytes"].GetMetric(), tableName).GetGauge().GetValue()
			bytesPerSecond := (hotBytes + coldBytes) / retention
			if tableName == "http_events" {
				maxHTTPBytesPerSecond = math.Max(maxHTTPBytesPerSecond, bytesPerSecond)
			} else {
				maxOtherBytesPerSecond = math.Max(maxOtherBytesPerSecond, bytesPerSecond)
			}
		}
		now := time.Now()
		httpTableSizeStore.Record(now, CalculatePerTableSize(maxHTTPBytesPerSecond, 180*time.Second))
		otherTableSizeStore.Record(now, CalculatePerTableSize(maxOtherBytesPerSecond, 180*time.Second))
		tableCountStore.Record(now, tableCount)
	}
}

func SetFromEnv(envName string, value *string) {
	envValue := os.Getenv(envName)
	if envValue == "" {
		return
	}
	*value = envValue
}

func LabelValue(m *prom.Metric, name string) string {
	for _, label := range m.GetLabel() {
		if label.GetName() == name {
			return label.GetValue()
		}
	}
	return ""
}

func FindMetricByTableName(ms []*prom.Metric, tableName string) *prom.Metric {
	for _, m := range ms {
		metricTableName := LabelValue(m, "name")
		if metricTableName == tableName {
			return m
		}
	}
	return nil
}

type BytesPerSecond = float64

type MiB = uint

type Bytes = uint

type Percent = uint

func CalculatePerTableSize(maxBytesPerSecond BytesPerSecond, targetRetention time.Duration) Bytes {
	return Bytes(math.Ceil(maxBytesPerSecond * targetRetention.Seconds()))
}

type PEMTableStoreLimit struct {
	Total          MiB
	HTTP           Percent
	StirlingError  Bytes
	ProcExitEvents Bytes
}

func (l PEMTableStoreLimit) String() string {
	return fmt.Sprintf("http=%d%%, stirlingError=%d bytes, procExitEvents=%d bytes, total=%d MiB", l.HTTP, l.StirlingError, l.ProcExitEvents, l.Total)
}

func CalculatePEMTableStoreLimit(numTables uint, otherTableSize Bytes, stirlingErrorsSize Bytes, procExitEventsSize Bytes, httpSize Bytes) PEMTableStoreLimit {
	// from pem_manager.cc:101
	// GIVEN: otherTableSize = (memoryLimit - httpSize - stirlingErrorsSize - procExitEventsSize) / (numTables - 4)
	memoryLimit := MiB(math.Ceil(
		float64(otherTableSize*(numTables-4)+httpSize+stirlingErrorsSize+procExitEventsSize) / (1024.0 * 1024.0),
	))
	return PEMTableStoreLimit{
		Total:          memoryLimit,
		HTTP:           Percent(math.Ceil(float64(httpSize) / float64(memoryLimit*1024.0*1024.0) * 100)),
		StirlingError:  stirlingErrorsSize,
		ProcExitEvents: procExitEventsSize,
	}
}

type Number interface {
	~uint
}

type NumberStore[V Number] struct {
	mu               sync.Mutex
	retentionMinutes int
	index            int
	observations     [][]observation[V]
}

func NewValueStore[V Number](retentionMinutes int) NumberStore[V] {
	return NumberStore[V]{
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

func (s *NumberStore[V]) lastMinute() int64 {
	if len(s.observations[s.index]) == 0 {
		return 0
	}
	return s.observations[s.index][0].At.Unix() / 60
}

func (s *NumberStore[V]) LatestValue() V {
	s.mu.Lock()
	defer s.mu.Unlock()
	var value V
	if len(s.observations[s.index]) > 0 {
		value = s.observations[s.index][len(s.observations[s.index])-1].Value
	}
	return value
}

func (s *NumberStore[V]) MaxValue() V {
	s.mu.Lock()
	defer s.mu.Unlock()
	var maxValue V
	for _, obs := range s.observations {
		for _, ob := range obs {
			if ob.Value > maxValue {
				maxValue = ob.Value
			}
		}
	}
	return maxValue
}

func (s *NumberStore[V]) PercentileValue(percentile float64) V {
	s.mu.Lock()
	defer s.mu.Unlock()
	var values []V
	for _, obs := range s.observations {
		for _, ob := range obs {
			values = append(values, ob.Value)
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i] < values[j]
	})
	low := int(math.Floor(float64(len(values)) * percentile))
	high := int(math.Ceil(float64(len(values)) * percentile))
	return (values[low] + values[high]) / V(2)
}

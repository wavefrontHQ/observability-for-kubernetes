package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/nats-io/nats.go"
	prom "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"px.dev/pixie/src/vizier/messages/messagespb"
)

const NatsMetricsChannel = "Metrics"
const retentionTimeNsMetricName = "min_time"
const colBytesMetricName = "table_cold_bytes"
const hotBytesMetricName = "table_hot_bytes"

var namespace = "observability-system"
var natsServer = "pl-nats"
var clientTLSCertFile = ""
var clientTLSKeyFile = ""
var tlsCAFile = ""
var targetMinRetentionTime = 90 * time.Second
var targetMaxRetentionTime = 180 * time.Second
var measureMinutes = 5

func init() {
	OptionalFromEnv("PS_NAMESPACE", ParseString, &namespace)
	OptionalFromEnv("PS_NATS_SERVER", ParseString, &natsServer)
	RequireFromEnv("PL_CLIENT_TLS_CERT", ParseString, &clientTLSCertFile)
	RequireFromEnv("PL_CLIENT_TLS_KEY", ParseString, &clientTLSKeyFile)
	RequireFromEnv("PL_TLS_CA_CERT", ParseString, &tlsCAFile)
	OptionalFromEnv("PS_TARGET_MIN_RETENTION_TIME", time.ParseDuration, &targetMinRetentionTime)
	OptionalFromEnv("PS_TARGET_MAX_RETENTION_TIME", time.ParseDuration, &targetMaxRetentionTime)
	OptionalFromEnv("PS_MEASURE_MINUTES", ParseInt, &measureMinutes)
}

func main() {
	client, err := getClient()
	if err != nil {
		log.Fatalf("error getting kuberntes client: %s", err.Error())
	}
	natsConn, err := nats.Connect(natsServer, nats.ClientCert(clientTLSCertFile, clientTLSKeyFile), nats.RootCAs(tlsCAFile))
	if err != nil {
		log.Fatalf("error creating nats connection: %s", err)
	}
	natsMsgs := make(chan *nats.Msg)
	subscription, err := natsConn.ChanSubscribe(NatsMetricsChannel, natsMsgs)
	if err != nil {
		log.Fatalf("error creating nats subscriptions to Metrics topic: %s", err)
	}
	defer subscription.Unsubscribe()
	httpTableSizeStore := NewValueStore[BytesPerSecond](measureMinutes)
	otherTableSizeStore := NewValueStore[BytesPerSecond](measureMinutes)
	tableCountStore := NewValueStore[int](measureMinutes)
	timer := time.NewTimer(time.Duration(measureMinutes) * time.Minute)
MainLoop:
	for {
		select {
		case <-timer.C:
			break MainLoop
		case natsMsg := <-natsMsgs:
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
			metricFamily := metricsFamilies[retentionTimeNsMetricName]
			now := time.Now()
			tableCountStore.Record(now, len(metricFamily.GetMetric()))
			for _, retentionNs := range metricFamily.GetMetric() {
				tableName := LabelValue(retentionNs, "name")
				retention := retentionNs.GetGauge().GetValue() / float64(time.Second)
				if retention == 0.0 {
					continue
				}
				coldBytes := FindMetricByTableName(metricsFamilies[colBytesMetricName].GetMetric(), tableName).GetGauge().GetValue()
				hotBytes := FindMetricByTableName(metricsFamilies[hotBytesMetricName].GetMetric(), tableName).GetGauge().GetValue()
				bytesPerSecond := (hotBytes + coldBytes) / retention
				if tableName == "http_events" {
					httpTableSizeStore.Record(now, bytesPerSecond)
				} else {
					otherTableSizeStore.Record(now, bytesPerSecond)
				}
			}
		}
	}
	pemPods, err := getPodsByLabel(client, namespace, "name=vizier-pem")
	if err != nil {
		log.Fatalf("error listing PEMs: %s", err.Error())
	}
	var maxMissedRowBatchSize Bytes
	for _, pemPod := range pemPods {
		rowBatchSizeErrors, err := extractFromPodLogs(client, pemPod, ExtractRowBatchSizeError)
		if err != nil {
			log.Fatalf("error extracting row batch size errors: %s", err.Error())
		}
		for _, rowBatchSizeError := range rowBatchSizeErrors {
			maxMissedRowBatchSize = MaxNumber(rowBatchSizeError.RowBatchSize, maxMissedRowBatchSize)
		}
	}
	log.Printf("max missed row batch size: %d", maxMissedRowBatchSize)
	tableCount := tableCountStore.MaxValue()
	tableStoreLimitMin := CalculatePEMTableStoreLimit(
		tableCount,
		MaxNumber(maxMissedRowBatchSize, CalculatePerTableSize(otherTableSizeStore.MaxValue(), targetMinRetentionTime)),
		0,
		0,
		MaxNumber(maxMissedRowBatchSize, CalculatePerTableSize(httpTableSizeStore.MaxValue(), targetMinRetentionTime)),
	)
	tableStoreLimitMax := CalculatePEMTableStoreLimit(
		tableCount,
		MaxNumber(maxMissedRowBatchSize, CalculatePerTableSize(otherTableSizeStore.MaxValue(), targetMaxRetentionTime)),
		0,
		0,
		MaxNumber(maxMissedRowBatchSize, CalculatePerTableSize(httpTableSizeStore.MaxValue(), targetMaxRetentionTime)),
	)
	log.Printf("smallest recommended table store size for %d tables: %s", tableCount, tableStoreLimitMin)
	log.Printf("largest recommended table store size for %d tables: %s", tableCount, tableStoreLimitMax)
}

func RequireFromEnv[T any](envName string, parse func(string) (T, error), value *T) {
	envValue := os.Getenv(envName)
	if envValue == "" {
		panic(fmt.Sprintf("%s is required, but was not set", envName))
	}
	v, err := parse(envValue)
	if err != nil {
		panic(err)
	}
	*value = v
}

func OptionalFromEnv[T any](envName string, parse func(string) (T, error), value *T) {
	envValue := os.Getenv(envName)
	if envValue == "" {
		return
	}
	v, err := parse(envValue)
	if err != nil {
		panic(err)
	}
	*value = v
}

func ParseInt(s string) (int, error) {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return int(i), nil
}

func ParseString(s string) (string, error) {
	return s, nil
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

type MiB = int

type Bytes = int

type Percent = int

func CalculatePerTableSize(maxBytesPerSecond BytesPerSecond, targetRetention time.Duration) Bytes {
	return Bytes(math.Ceil(maxBytesPerSecond * targetRetention.Seconds()))
}

type PEMTableStoreLimit struct {
	Total MiB
	HTTP  Percent
}

func (l PEMTableStoreLimit) String() string {
	return fmt.Sprintf("http=%d%%, total=%d MiB", l.HTTP, l.Total)
}

func CalculatePEMTableStoreLimit(numTables int, otherTableSize Bytes, stirlingErrorsSize Bytes, procExitEventsSize Bytes, httpSize Bytes) PEMTableStoreLimit {
	// from pem_manager.cc:101
	// GIVEN: otherTableSize = (memoryLimit - httpSize - stirlingErrorsSize - procExitEventsSize) / (numTables - 4)
	memoryLimit := MiB(math.Ceil(
		float64(otherTableSize*(numTables-4)+httpSize+stirlingErrorsSize+procExitEventsSize) / (1024.0 * 1024.0),
	))
	memoryLimit = MiB(math.Ceil(float64(memoryLimit)/10.0) * 10.0)
	return PEMTableStoreLimit{
		Total: memoryLimit,
		HTTP:  Percent(5.0 * math.Ceil(float64(httpSize)/float64(memoryLimit*1024.0*1024.0)*100/5.0)),
	}
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

func getClient() (kubernetes.Interface, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("error in getting config: %s", err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error in getting access to K8S: %s", err.Error())
	}
	return clientset, nil
}

func getPodsByLabel(client kubernetes.Interface, namespace string, labelSelector string) ([]corev1.Pod, error) {
	podList, err := client.CoreV1().Pods(namespace).List(context.Background(), v1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching pods by label (%s) in ns %s: %s", labelSelector, namespace, err.Error())
	}
	return podList.Items, nil
}

func extractFromPodLogs[T any](client kubernetes.Interface, pod corev1.Pod, extract func(string) (T, bool)) ([]T, error) {
	podLogOpts := corev1.PodLogOptions{}
	req := client.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error in opening stream: %s", err.Error())
	}
	defer podLogs.Close()
	var matched []T
	lines := bufio.NewScanner(podLogs)
	for lines.Scan() {
		line := lines.Text()
		data, matches := extract(line)
		if matches {
			matched = append(matched, data)
		}
	}
	if lines.Err() != nil {
		return nil, fmt.Errorf("error reading lines: %s", err.Error())
	}
	return matched, nil
}

type RowBatchSizeError struct {
	RowBatchSize Bytes
	MaxTableSize Bytes
}

var rowBatchSizeRegex = regexp.MustCompile(`RowBatch size \((?P<RowBatchSize>\d+)\).+\((?P<MaxTableSize>\d+)\).$`)

func ExtractRowBatchSizeError(line string) (RowBatchSizeError, bool) {
	match := rowBatchSizeRegex.FindStringSubmatch(line)
	if len(match) == 0 {
		return RowBatchSizeError{}, false
	}
	rowBatchSize, err := ParseInt(match[1])
	if err != nil {
		panic(err)
	}
	maxTableSize, err := ParseInt(match[2])
	if err != nil {
		panic(err)
	}
	return RowBatchSizeError{RowBatchSize: rowBatchSize, MaxTableSize: maxTableSize}, true
}

func MaxNumber[T Number](a, b T) T {
	if b > a {
		return b
	}
	return a
}

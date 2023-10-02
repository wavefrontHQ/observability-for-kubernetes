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
const tableNumBatchesMetricName = "table_num_batches"

const HTTPEventsTable = "http_events"
const UnknownTable = "unknown"

var namespace = "observability-system"
var natsServer = "pl-nats"
var clientTLSCertFile = ""
var clientTLSKeyFile = ""
var tlsCAFile = ""
var trafficScaleFactor = 2.0
var measureMinutes = 10
var reportMinutes int

func init() {
	OptionalFromEnv("PS_NAMESPACE", ParseString, &namespace)
	OptionalFromEnv("PS_NATS_SERVER", ParseString, &natsServer)
	RequireFromEnv("PL_CLIENT_TLS_CERT", ParseString, &clientTLSCertFile)
	RequireFromEnv("PL_CLIENT_TLS_KEY", ParseString, &clientTLSKeyFile)
	RequireFromEnv("PL_TLS_CA_CERT", ParseString, &tlsCAFile)
	OptionalFromEnv("PS_TRAFFIC_SCALE_FACTOR", ParseFloat64, &trafficScaleFactor)
	OptionalFromEnv("PS_MEASURE_MINUTES", ParseInt, &measureMinutes)
	reportMinutes = measureMinutes / 2
	OptionalFromEnv("PS_REPORT_MINUTES", ParseInt, &reportMinutes)
}

func main() {
	log.Printf("traffic scale factor: %f", trafficScaleFactor)
	client, err := GetClient()
	if err != nil {
		log.Fatalf("error getting kuberntes client: %s", err.Error())
	}
	cronScripts, err := GetConfigMapsByLabel(client, namespace, "purpose=cron-script")
	if err != nil {
		log.Fatalf("error fetching cron scripts: %s", err.Error())
	}
	targetRetentionTime := GetMaxFrequencyFromCronScripts(cronScripts)
	natsConn, err := nats.Connect(natsServer, nats.ClientCert(clientTLSCertFile, clientTLSKeyFile), nats.RootCAs(tlsCAFile))
	if err != nil {
		log.Fatalf("error creating nats connection: %s", err)
	}
	log.Printf("minimum target retention time: %s", targetRetentionTime.String())
	natsMsgs := make(chan *nats.Msg, 1024)
	subscription, err := natsConn.ChanSubscribe(NatsMetricsChannel, natsMsgs)
	if err != nil {
		log.Fatalf("error creating nats subscriptions to Metrics topic: %s", err)
	}
	defer subscription.Unsubscribe()
	statsStore := NewStatsStore(measureMinutes, []string{HTTPEventsTable})
	go func() {
		for {
			time.Sleep(time.Duration(reportMinutes) * time.Minute)
			pemPods, err := GetPodsByLabel(client, namespace, "name=vizier-pem")
			if err != nil {
				log.Fatalf("error listing PEMs: %s", err.Error())
			}
			maxMissedRowBatchSize := GetMaxMissedRowBatchSize(pemPods, int64(measureMinutes*60), client)

			httpAvgRowBatchSize := MiB(math.Ceil(float64(statsStore.TableBytes(HTTPEventsTable).Sum()) / float64(statsStore.TableRowBatches(HTTPEventsTable).Sum()) / (1024.0 * 1024.0)))
			otherAvgRowBatchSize := MiB(math.Ceil(float64(statsStore.TableBytes(UnknownTable).Sum()) / float64(statsStore.TableRowBatches(UnknownTable).Sum()) / (1024.0 * 1024.0)))

			httpMinTableSize := MaxNumber(maxMissedRowBatchSize, httpAvgRowBatchSize)
			otherMinTableSize := MaxNumber(maxMissedRowBatchSize, otherAvgRowBatchSize)

			log.Printf("minimum http table size: %d MiB", httpMinTableSize)
			log.Printf("minimum other table size: %d MiB", otherMinTableSize)
			scaledRetentionTime := ScaleDuration(trafficScaleFactor, targetRetentionTime)
			tableCount := statsStore.TableCount().MaxValue()
			smallestTableStoreLimit := CalculatePEMTableStoreLimit(
				tableCount,
				MaxNumber(otherMinTableSize, CalculatePerTableSize(statsStore.TableRate(UnknownTable).MinValue(), scaledRetentionTime)),
				0,
				0,
				MaxNumber(httpMinTableSize, CalculatePerTableSize(statsStore.TableRate(HTTPEventsTable).MinValue(), scaledRetentionTime)),
			)
			tableStoreLimit := CalculatePEMTableStoreLimit(
				tableCount,
				MaxNumber(otherMinTableSize, CalculatePerTableSize(statsStore.TableRate(UnknownTable).PercentileValue(0.5), scaledRetentionTime)),
				0,
				0,
				MaxNumber(httpMinTableSize, CalculatePerTableSize(statsStore.TableRate(HTTPEventsTable).PercentileValue(0.5), scaledRetentionTime)),
			)
			largestTableStoreLimit := CalculatePEMTableStoreLimit(
				tableCount,
				MaxNumber(otherMinTableSize, CalculatePerTableSize(statsStore.TableRate(UnknownTable).MaxValue(), scaledRetentionTime)),
				0,
				0,
				MaxNumber(httpMinTableSize, CalculatePerTableSize(statsStore.TableRate(HTTPEventsTable).MaxValue(), scaledRetentionTime)),
			)
			log.Printf("smallest table store size for %d tables: %s", tableCount, smallestTableStoreLimit)
			log.Printf("largest table store size for %d tables:%s", tableCount, largestTableStoreLimit)
			log.Printf("recommended table store size for %d tables: %s", tableCount, tableStoreLimit)
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
		buf := bytes.NewReader([]byte(metricsMsg.GetPromMetricsText()))
		metricsFamilies, err := (&expfmt.TextParser{}).TextToMetricFamilies(buf)
		if err != nil {
			log.Fatalf("invalid prometheus metrics: %s", err)
		}
		statsStore.Record(time.Now(), metricsFamilies)
	}
}

type StatsStore struct {
	measureMinutes  int
	knownTables     []string
	tableRates      map[string]*NumberStore[BytesPerSecond]
	tableBytes      map[string]*NumberStore[Bytes]
	tableRowBatches map[string]*NumberStore[int]
	tableCountStore *NumberStore[int]
}

func NewStatsStore(measureMinutes int, knownTables []string) *StatsStore {
	return &StatsStore{
		measureMinutes:  measureMinutes,
		knownTables:     knownTables,
		tableRates:      map[string]*NumberStore[BytesPerSecond]{},
		tableBytes:      map[string]*NumberStore[Bytes]{},
		tableRowBatches: map[string]*NumberStore[int]{},
		tableCountStore: NewValueStore[int](measureMinutes),
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
		s.tableRates[tableName] = NewValueStore[BytesPerSecond](s.measureMinutes)
	}
	return s.tableRates[tableName]
}

func (s *StatsStore) TableCount() *NumberStore[int] {
	return s.tableCountStore
}

func (s *StatsStore) TableBytes(tableName string) *NumberStore[Bytes] {
	tableName = s.normalizedTableName(tableName)
	if s.tableBytes[tableName] == nil {
		s.tableBytes[tableName] = NewValueStore[Bytes](s.measureMinutes)
	}
	return s.tableBytes[tableName]
}

func (s *StatsStore) TableRowBatches(tableName string) *NumberStore[int] {
	tableName = s.normalizedTableName(tableName)
	if s.tableRowBatches[tableName] == nil {
		s.tableRowBatches[tableName] = NewValueStore[Bytes](s.measureMinutes)
	}
	return s.tableRowBatches[tableName]
}

func GetMaxMissedRowBatchSize(pemPods []corev1.Pod, sinceSeconds int64, client kubernetes.Interface) MiB {
	var maxMissedRowBatchSize Bytes
	for _, pemPod := range pemPods {
		rowBatchSizeErrors, err := ExtractFromPodLogs(client, pemPod, sinceSeconds, ExtractRowBatchSizeError)
		if err != nil {
			log.Fatalf("error extracting row batch size errors: %s", err.Error())
		}
		if len(rowBatchSizeErrors) <= 1 { // you get one missed row batch at startup, but nothing else
			continue
		}
		for _, rowBatchSizeError := range rowBatchSizeErrors[1:] {
			maxMissedRowBatchSize = MaxNumber(rowBatchSizeError.RowBatchSize, maxMissedRowBatchSize)
		}
	}
	return MiB(math.Ceil(float64(maxMissedRowBatchSize) / (1024.0 * 1024.0)))
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

func ParseFloat64(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
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
	return FindMetricByLabels(ms, map[string]string{"name": tableName})
}

func FindMetricByLabels(ms []*prom.Metric, labels map[string]string) *prom.Metric {
	for _, m := range ms {
		allMatched := true
		for name, expectedValue := range labels {
			actualValue := LabelValue(m, name)
			allMatched = allMatched && actualValue == expectedValue
		}
		if allMatched {
			return m
		}
	}
	return nil
}

type BytesPerSecond = float64

type MiB = int

type Bytes = int

type Percent = int

func CalculatePerTableSize(maxBytesPerSecond BytesPerSecond, targetRetention time.Duration) MiB {
	return MiB(math.Ceil(maxBytesPerSecond * targetRetention.Seconds() / (1024.0 * 1024.0)))
}

type PEMTableStoreLimit struct {
	Total MiB
	HTTP  Percent
}

func (l PEMTableStoreLimit) String() string {
	return fmt.Sprintf("http=%d%%, total=%d MiB", l.HTTP, l.Total)
}

func CalculatePEMTableStoreLimit(numTables int, otherTableSize MiB, stirlingErrorsSize Bytes, procExitEventsSize Bytes, httpSize MiB) PEMTableStoreLimit {
	// from pem_manager.cc:101
	// GIVEN: otherTableSize = (memoryLimit - httpSize - stirlingErrorsSize - procExitEventsSize) / (numTables - 4)
	memoryLimit := MiB(math.Ceil(
		float64(otherTableSize)*float64(numTables-4) + float64(httpSize) + float64(stirlingErrorsSize+procExitEventsSize)/(1024.0*1024.0),
	))
	//memoryLimit = MiB(math.Ceil(float64(memoryLimit)/10.0) * 10.0)
	return PEMTableStoreLimit{
		Total: memoryLimit,
		HTTP:  Percent(math.Ceil(float64(httpSize) / float64(memoryLimit) * 100.0)),
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

func NewValueStore[V Number](retentionMinutes int) *NumberStore[V] {
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
	var maxValue V
	s.each(func(_ time.Time, value V) {
		if value > maxValue {
			maxValue = value
		}
	})
	return maxValue
}

func (s *NumberStore[V]) MinValue() V {
	var minValue V
	first := true
	s.each(func(_ time.Time, value V) {
		if first {
			minValue = value
			first = false
		} else if value < minValue {
			minValue = value
		}
	})
	return minValue
}

func (s *NumberStore[V]) PercentileValue(percentile float64) V {
	var values []V
	s.each(func(_ time.Time, value V) {
		values = append(values, value)
	})
	sort.Slice(values, func(i, j int) bool {
		return values[i] < values[j]
	})
	low := int(math.Floor(float64(len(values)) * percentile))
	high := int(math.Ceil(float64(len(values)) * percentile))
	return (values[low] + values[high]) / V(2)
}

func (s *NumberStore[V]) Sum() V {
	var sum V
	s.each(func(_ time.Time, value V) {
		sum += value
	})
	return sum
}

func GetClient() (kubernetes.Interface, error) {
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

func GetPodsByLabel(client kubernetes.Interface, namespace string, labelSelector string) ([]corev1.Pod, error) {
	podList, err := client.CoreV1().Pods(namespace).List(context.Background(), v1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching pods by label (%s) in ns %s: %s", labelSelector, namespace, err.Error())
	}
	return podList.Items, nil
}

func GetConfigMapsByLabel(client kubernetes.Interface, namespace string, labelSelector string) ([]corev1.ConfigMap, error) {
	cmList, err := client.CoreV1().ConfigMaps(namespace).List(context.Background(), v1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching pods by label (%s) in ns %s: %s", labelSelector, namespace, err.Error())
	}
	return cmList.Items, nil
}

var cronYAMLFrequencyRegex = regexp.MustCompile(`(?m)frequency_s:\s*(\d+)$`)

func GetMaxFrequencyFromCronScripts(configMaps []corev1.ConfigMap) time.Duration {
	maxFrequencyS := 0
	for _, configMap := range configMaps {
		cronYAML := configMap.Data["cron.yaml"]
		match := cronYAMLFrequencyRegex.FindStringSubmatch(cronYAML)
		if len(match) == 0 {
			continue
		}
		cronFrequencyS, err := ParseInt(match[1])
		if err != nil {
			panic(err)
		}
		maxFrequencyS = MaxNumber(maxFrequencyS, cronFrequencyS)
	}
	return time.Duration(maxFrequencyS) * time.Second
}

func ExtractFromPodLogs[T any](client kubernetes.Interface, pod corev1.Pod, sinceSeconds int64, extract func(string) (T, bool)) ([]T, error) {
	podLogOpts := corev1.PodLogOptions{SinceSeconds: &sinceSeconds}
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

func ScaleDuration(s float64, d time.Duration) time.Duration {
	return time.Duration(s * float64(d))
}

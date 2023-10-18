package main

import (
	"log"
	"strings"
	"time"

	prom "github.com/prometheus/client_model/go"
)

const retentionTimeNsMetricName = "min_time"
const colBytesMetricName = "table_cold_bytes"
const hotBytesMetricName = "table_hot_bytes"
const tableNumBatchesMetricName = "table_num_batches"

var namespace = "observability-system"
var natsServer = "pl-nats"
var clientTLSCertFile = ""
var clientTLSKeyFile = ""
var tlsCAFile = ""
var trafficScaleFactor = 2.0
var samplePeriodMinutes = 10
var reportMinutes int

func init() {
	OptionalFromEnv("PS_NAMESPACE", ParseString, &namespace)
	OptionalFromEnv("PS_NATS_SERVER", ParseString, &natsServer)
	RequireFromEnv("PL_CLIENT_TLS_CERT", ParseString, &clientTLSCertFile)
	RequireFromEnv("PL_CLIENT_TLS_KEY", ParseString, &clientTLSKeyFile)
	RequireFromEnv("PL_TLS_CA_CERT", ParseString, &tlsCAFile)
	OptionalFromEnv("PS_TRAFFIC_SCALE_FACTOR", ParseFloat64, &trafficScaleFactor)
	OptionalFromEnv("PS_SAMPLE_PERIOD_MINUTES", ParseInt, &samplePeriodMinutes)
	reportMinutes = samplePeriodMinutes
	OptionalFromEnv("PS_REPORT_MINUTES", ParseInt, &reportMinutes)
}

func main() {
	log.Printf("traffic scale factor: %f", trafficScaleFactor)
	statsStore := NewStatsStore(samplePeriodMinutes, KnownTables)
	metricStream, err := NewMetricStream(natsServer, clientTLSCertFile, clientTLSKeyFile, tlsCAFile)
	if err != nil {
		log.Fatalf("error creating metrics stream: %s", err.Error())
	}
	unsubscribe, err := metricStream.Subscribe(func(podName string, metricsFamilies map[string]*prom.MetricFamily) {
		if !strings.HasPrefix(podName, "vizier-pem") {
			return
		}
		statsStore.Record(time.Now(), metricsFamilies)
	})
	if err != nil {
		log.Fatalf("error subscribing to metrics stream: %s", err.Error())
	}
	defer unsubscribe()
	client, err := GetClient()
	if err != nil {
		log.Fatalf("error getting kuberntes typed client: %s", err.Error())
	}
	dynamicClient, err := GetDynamicClient()
	if err != nil {
		log.Fatalf("error getting kuberntes dynamic client: %s", err.Error())
	}
	(&Recommender{
		ReportMinutes:       reportMinutes,
		SamplePeriodMinutes: samplePeriodMinutes,
		TrafficScaleFactor:  trafficScaleFactor,
		Client:              client,
		DynamicClient:       dynamicClient,
		Namespace:           namespace,
		StatsStore:          statsStore,
	}).Run()

}

type BytesPerSecond = float64

type MiB = int

type Bytes = int

type Percent = int

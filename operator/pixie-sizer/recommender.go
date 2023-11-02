package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math"
	"strings"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

const HTTPEventsTable = "http_events"

var KnownTables = []string{HTTPEventsTable}

type Recommender struct {
	ReportMinutes       int
	SamplePeriodMinutes int
	TrafficScaleFactor  float64

	Client        kubernetes.Interface
	DynamicClient dynamic.Interface
	Namespace     string

	StatsStore *StatsStore
}

func (r *Recommender) Run() {
	for {
		time.Sleep(time.Duration(r.ReportMinutes) * time.Minute)
		cronScripts, err := GetConfigMapsByLabel(r.Client, r.Namespace, "purpose=cron-script")
		if err != nil {
			log.Fatalf("error fetching cron scripts: %s", err.Error())
		}
		targetRetentionTime := GetMaxFrequencyFromCronScripts(cronScripts)
		log.Printf("minimum target retention time: %s", targetRetentionTime.String())
		pemPods, err := GetPodsByLabel(r.Client, r.Namespace, "name=vizier-pem")
		if err != nil {
			log.Fatalf("error listing PEMs: %s", err.Error())
		}
		maxMissedRowBatchSize := GetMaxMissedRowBatchSize(pemPods, int64(r.SamplePeriodMinutes*60), r.Client)

		httpAvgRowBatchSize := MiB(math.Ceil(float64(r.StatsStore.TableBytes(HTTPEventsTable).Sum()) / float64(r.StatsStore.TableRowBatches(HTTPEventsTable).Sum()) / (1024.0 * 1024.0)))
		otherAvgRowBatchSize := MiB(math.Ceil(float64(r.StatsStore.TableBytes(UnknownTable).Sum()) / float64(r.StatsStore.TableRowBatches(UnknownTable).Sum()) / (1024.0 * 1024.0)))

		httpMinTableSize := MaxNumber(maxMissedRowBatchSize, httpAvgRowBatchSize)
		otherMinTableSize := MaxNumber(maxMissedRowBatchSize, otherAvgRowBatchSize)

		log.Printf("max missed row batch size: %d", maxMissedRowBatchSize)
		log.Printf("minimum http table size: %d MiB", httpMinTableSize)
		log.Printf("minimum other table size: %d MiB", otherMinTableSize)
		scaledRetentionTime := ScaleDuration(r.TrafficScaleFactor, targetRetentionTime)
		tableCount := r.StatsStore.TableCount().MaxValue()
		log.Printf("table count: %d", tableCount)
		tableStoreLimit := CalculatePEMLimits(
			tableCount,
			MaxNumber(otherMinTableSize, CalculatePerTableSize(r.StatsStore.TableRate(UnknownTable).MaxValue(), scaledRetentionTime)),
			0,
			0,
			MaxNumber(httpMinTableSize, CalculatePerTableSize(r.StatsStore.TableRate(HTTPEventsTable).MaxValue(), scaledRetentionTime)),
		)
		log.Printf("recommended settings:\n%s", tableStoreLimit.YAMLSnippet())
	}
}

func CalculatePerTableSize(maxBytesPerSecond BytesPerSecond, targetRetention time.Duration) MiB {
	return MiB(math.Ceil(maxBytesPerSecond * targetRetention.Seconds() / (1024.0 * 1024.0)))
}

type PEMLimits struct {
	Total MiB
	HTTP  Percent
}

func CalculatePEMLimits(numTables int, otherTableSize MiB, stirlingErrorsSize Bytes, procExitEventsSize Bytes, httpSize MiB) PEMLimits {
	// from pem_manager.cc:101
	// GIVEN: otherTableSize = (memoryLimit - httpSize - stirlingErrorsSize - procExitEventsSize) / (numTables - 4)
	memoryLimit := MiB(math.Ceil(
		float64(otherTableSize)*float64(numTables-4) + float64(httpSize) + float64(stirlingErrorsSize+procExitEventsSize)/(1024.0*1024.0),
	))
	return PEMLimits{
		Total: memoryLimit,
		HTTP:  Percent(math.Ceil(float64(httpSize) / float64(memoryLimit) * 100.0)),
	}
}

func (l PEMLimits) MemoryLimit() MiB {
	return MaxNumber(0, l.Total-200) + 750
}

func (l PEMLimits) YAMLSnippet() string {
	buf := bytes.NewBuffer(nil)

	FprintlnWithIndent(buf, 0, "spec:")
	FprintlnWithIndent(buf, 1, "experimental:")
	FprintlnWithIndent(buf, 2, "pixie:")
	FprintlnWithIndent(buf, 3, "table_store_limits:")
	FprintlnWithIndent(buf, 4, "total_mib: %d", l.Total)
	FprintlnWithIndent(buf, 4, "http_events_percent: %d", l.HTTP)

	return buf.String()
}

func FprintlnWithIndent(w io.Writer, tabs int, line string, args ...any) {
	_, _ = fmt.Fprintf(w, strings.Repeat("  ", tabs))
	_, _ = fmt.Fprintf(w, line+"\n", args...)
}

package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

const HTTPEventsTable = "http_events"

var KnownTables = []string{HTTPEventsTable}

type Recommender struct {
	ReportMinutes      int
	MeasureMinutes     int
	TrafficScaleFactor float64

	Client        kubernetes.Interface
	DynamicClient dynamic.Interface
	Namespace     string

	StatsStore *StatsStore
}

func (r *Recommender) Run() {
	for {
		time.Sleep(time.Duration(r.ReportMinutes) * time.Minute)
		settingsPrefix, err := GetPrefixForPEMSettings(r.DynamicClient, r.Namespace)
		if err != nil {
			log.Fatalf("error determining settings prefix: %s", err)
		}
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
		maxMissedRowBatchSize := GetMaxMissedRowBatchSize(pemPods, int64(r.MeasureMinutes*60), r.Client)

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
		smallestTableStoreLimit := CalculatePEMLimits(
			tableCount,
			MaxNumber(otherMinTableSize, CalculatePerTableSize(r.StatsStore.TableRate(UnknownTable).MinValue(), scaledRetentionTime)),
			0,
			0,
			MaxNumber(httpMinTableSize, CalculatePerTableSize(r.StatsStore.TableRate(HTTPEventsTable).MinValue(), scaledRetentionTime)),
		)
		tableStoreLimit := CalculatePEMLimits(
			tableCount,
			MaxNumber(otherMinTableSize, CalculatePerTableSize(r.StatsStore.TableRate(UnknownTable).PercentileValue(0.5), scaledRetentionTime)),
			0,
			0,
			MaxNumber(httpMinTableSize, CalculatePerTableSize(r.StatsStore.TableRate(HTTPEventsTable).PercentileValue(0.5), scaledRetentionTime)),
		)
		largestTableStoreLimit := CalculatePEMLimits(
			tableCount,
			MaxNumber(otherMinTableSize, CalculatePerTableSize(r.StatsStore.TableRate(UnknownTable).MaxValue(), scaledRetentionTime)),
			0,
			0,
			MaxNumber(httpMinTableSize, CalculatePerTableSize(r.StatsStore.TableRate(HTTPEventsTable).MaxValue(), scaledRetentionTime)),
		)
		log.Printf("smallest settings (under %s): %s", settingsPrefix, smallestTableStoreLimit)
		log.Printf("largest settings (under %s): %s", settingsPrefix, largestTableStoreLimit)
		log.Printf("recommended settings (under %s): %s", settingsPrefix, tableStoreLimit)
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

func (l PEMLimits) String() string {
	return fmt.Sprintf("table_store_limits.http_events_percent=%d, table_store_limits.total_mib=%d, resources.limits.memory=%dMi", l.HTTP, l.Total, l.MemoryLimit())
}

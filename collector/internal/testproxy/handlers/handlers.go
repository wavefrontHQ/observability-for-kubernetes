package handlers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/broadcaster"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/testproxy/eventline"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/testproxy/externalevent"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/testproxy/logs"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/testproxy/metricline"
)

func LogJsonArrayHandler(logVerifier *logs.LogVerifier) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Fatal(err)
			return
		}
		defer req.Body.Close()

		// Validate log format
		logLines := logVerifier.VerifyJsonArrayFormat(b)
		if len(logLines) == 0 {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Validate expected tags
		logVerifier.ValidateExpectedTags(logLines)

		// Validate filtering allowed tags
		logVerifier.ValidateAllowedTags(logLines)

		// Validate filtering denied tags
		logVerifier.ValidateDeniedTags(logLines)

		// Validate expected optional tags
		logVerifier.ValidateExpectedOptionalTags(logLines)

		w.WriteHeader(http.StatusOK)
	}
}

func LogJsonLinesHandler(logVerifier *logs.LogVerifier) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Fatal(err)
		}
		defer req.Body.Close()

		// Validate log format
		logLines := logVerifier.VerifyJsonLinesFormat(b)
		if len(logLines) == 0 {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Validate expected tags
		logVerifier.ValidateExpectedTags(logLines)

		// Validate filtering allowed tags
		logVerifier.ValidateAllowedTags(logLines)

		// Validate filtering denied tags
		logVerifier.ValidateDeniedTags(logLines)

		// Validate expected optional tags
		logVerifier.ValidateExpectedOptionalTags(logLines)

		w.WriteHeader(http.StatusOK)
	}
}

func LogAssertionHandler(store *logs.Results) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			log.Errorf("expected method %s but got %s", http.MethodGet, req.Method)
			return
		}

		if store.ReceivedLogCount == 0 {
			w.WriteHeader(http.StatusNoContent)
			_, _ = w.Write([]byte("No logs have been received"))
			return
		}

		output, err := store.ToJSON()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(fmt.Sprintf("Unable to marshal log test store object: %s", err.Error())))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(output)
	}
}

func HandleIncomingMetrics(proxylines *broadcaster.Broadcaster[string], conn net.Conn) {
	defer conn.Close()
	lines := bufio.NewScanner(conn)

	for lines.Scan() {
		if len(lines.Text()) == 0 {
			continue
		}
		proxylines.Publish(1*time.Second, lines.Text())
	}

	if err := lines.Err(); err != nil {
		log.Error(err.Error())
	}

	return
}

func DumpMetricsHandler(store *metricline.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		log.Infof("******* req received in DumpMetricsHandler: '%+v'", req)
		if req.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			log.Errorf("expected method %s but got %s", http.MethodGet, req.Method)
			return
		}

		badMetrics := store.BadMetrics()
		if len(badMetrics) > 0 {
			w.WriteHeader(http.StatusUnprocessableEntity)
			err := json.NewEncoder(w).Encode(badMetrics)
			if err != nil {
				log.Error(err.Error())
			}
			return
		}

		w.WriteHeader(http.StatusOK)

		err := json.NewEncoder(w).Encode(store.Metrics())
		if err != nil {
			log.Error(err.Error())
		}
	}
}

func DiffMetricsHandler(store *metricline.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			log.Errorf("expected method %s but got %s", http.MethodPost, req.Method)
			return
		}

		badMetrics := store.BadMetrics()
		if len(badMetrics) > 0 {
			w.WriteHeader(http.StatusUnprocessableEntity)
			err := json.NewEncoder(w).Encode(badMetrics)
			if err != nil {
				log.Error(err.Error())
			}
			return
		}

		var expectedMetrics []*metricline.Metric
		var excludedMetrics []*metricline.Metric
		lines := bufio.NewScanner(req.Body)
		defer req.Body.Close()

		for lines.Scan() {
			if len(lines.Bytes()) == 0 {
				continue
			}
			var err error
			if lines.Bytes()[0] == '~' {
				var excludedMetric *metricline.Metric
				excludedMetric, err = decodeMetric(lines.Bytes()[1:])
				excludedMetrics = append(excludedMetrics, excludedMetric)
			} else {
				var expectedMetric *metricline.Metric
				expectedMetric, err = decodeMetric(lines.Bytes())
				expectedMetrics = append(expectedMetrics, expectedMetric)
			}
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				err = json.NewEncoder(w).Encode(err.Error())
				if err != nil {
					log.Error(err.Error())
				}
				return
			}
		}

		linesErr := lines.Err()
		if linesErr != nil {
			w.WriteHeader(http.StatusBadRequest)
			ioErr := json.NewEncoder(w).Encode(linesErr.Error())
			if ioErr != nil {
				log.Error(ioErr.Error())
			}
			return
		}

		w.WriteHeader(http.StatusOK)

		linesErr = json.NewEncoder(w).Encode(metricline.DiffMetrics(expectedMetrics, excludedMetrics, store.Metrics()))
		if linesErr != nil {
			log.Error(linesErr.Error())
		}
	}
}

func decodeMetric(bytes []byte) (*metricline.Metric, error) {
	var metric *metricline.Metric
	err := json.Unmarshal(bytes, &metric)
	return metric, err
}

func JSONEncodeHandler[D any](data D) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Errorf("error encoding event results: %s", err.Error())
		}
	}
}

func EventAssertionHandler(eventStore *eventline.Store) http.HandlerFunc {
	return JSONEncodeHandler(eventStore)
}

func ExternalEventAssertionHandler(externalEventStore *externalevent.Store) http.HandlerFunc {
	return JSONEncodeHandler(externalEventStore)
}

func ExternalEventsHandler(externalEvents *broadcaster.Broadcaster[[]byte]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		buf := bytes.NewBuffer(make([]byte, 0, r.ContentLength))
		_, err := io.Copy(buf, r.Body)
		log.Debug(buf.String())
		if err != nil {
			log.Errorf("error reading external event body: %s", err.Error())
		}
		externalEvents.Publish(time.Second, buf.Bytes())
	}
}

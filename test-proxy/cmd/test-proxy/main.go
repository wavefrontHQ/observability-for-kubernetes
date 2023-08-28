package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/wavefronthq/observability-for-kubernetes/test-proxy/internal/broadcaster"
	"github.com/wavefronthq/observability-for-kubernetes/test-proxy/internal/testproxy/eventline"
	"github.com/wavefronthq/observability-for-kubernetes/test-proxy/internal/testproxy/externalevent"
	"github.com/wavefronthq/observability-for-kubernetes/test-proxy/internal/testproxy/handlers"
	"github.com/wavefronthq/observability-for-kubernetes/test-proxy/internal/testproxy/logs"
	"github.com/wavefronthq/observability-for-kubernetes/test-proxy/internal/testproxy/metricline"
)

var (
	externalEventAddr = ":9999"
	proxyAddr         = ":7777"
	controlAddr       = ":8888"
	runMode           = "metrics"
	logFilePath       string
	logLevel          = log.InfoLevel.String()

	// Needs to match what is set up in log sender
	expectedTags = []string{
		"user_defined_tag",
		"service",
		"application",
		"source",
		"cluster",
		"timestamp",
		"pod_name",
		"container_name",
		"namespace_name",
		"pod_id",
		"container_id",
		"integration",
		"log",
	}
	// Needs to match what is set up in log sender
	optionalTags = map[string]string{
		"level": "ERROR",
	}

	// Needs to match what is set up in log sender
	allowListFilteredTags = map[string][]string{
		"namespace_name": {"kube-system", "observability-system"},
	}
	// Needs to match what is set up in log sender
	denyListFilteredTags = map[string][]string{
		"container_name": {"kube-apiserver"},
	}
)

func init() {
	flag.StringVar(&proxyAddr, "proxy", proxyAddr, "host and port for the test \"wavefront proxy\" to listen on")
	flag.StringVar(&externalEventAddr, "external-events", externalEventAddr, "host and port for the http external event server to listen on")
	flag.StringVar(&controlAddr, "control", controlAddr, "host and port for the http control server to listen on")
	flag.StringVar(&runMode, "mode", runMode, "which mode to run in. Valid options are \"metrics\", and \"logs\"")
	flag.StringVar(&logFilePath, "logFilePath", logFilePath, "the full path to output logs to instead of using stdout")
	flag.StringVar(&logLevel, "logLevel", logLevel, "change log level. Default is \"info\", use \"debug\" for metric logging")
}

func main() {
	flag.Parse()

	log.SetFormatter(&log.TextFormatter{})
	if level, err := log.ParseLevel(logLevel); err == nil {
		log.SetLevel(level)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	proxylines := broadcaster.New[string]()

	metricStore := metricline.NewStore()
	metricStore.Subscribe(proxylines)

	eventStore := eventline.NewStore()
	eventStore.Subscribe(proxylines)

	logStore := logs.NewLogResults(copyStringMap(optionalTags))

	externalEvents := broadcaster.New[[]byte]()
	externalEventsStore := externalevent.NewStore()
	externalEventsStore.Subscribe(externalEvents)

	if logFilePath != "" {
		// Set log output to file to prevent our logging component from picking up stdout/stderr logs
		// and sending them back to us over and over.
		file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			fmt.Println("Could not create log output file: ", err)
			os.Exit(1)
		}

		log.SetOutput(file)
	}

	go serveExternalEvents(externalEvents)

	switch runMode {
	case "metrics":
		go serveMetrics(proxylines)
	case "logs":
		go serveLogs(logStore)
	default:
		log.Error("\"mode\" flag must be set to: \"metrics\" or \"logs\"")
		os.Exit(1)
	}

	// Blocking call to start up the control server
	serveControl(metricStore, eventStore, externalEventsStore, logStore)
	log.Println("Server gracefully shutdown")
}

func serveExternalEvents(externalEvents *broadcaster.Broadcaster[[]byte]) {
	routes := http.NewServeMux()
	routes.HandleFunc("/events", handlers.ExternalEventsHandler(externalEvents))

	log.Infof("http external events server listening on %s", externalEventAddr)
	if err := http.ListenAndServe(externalEventAddr, routes); err != nil {
		log.Fatal(err.Error())
	}
}

func serveMetrics(proxylines *broadcaster.Broadcaster[string]) {
	log.Infof("tcp metrics server listening on %s", proxyAddr)
	listener, err := net.Listen("tcp", proxyAddr)
	if err != nil {
		log.Fatal(err.Error())
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Error(err.Error())
			continue
		}
		go handlers.HandleIncomingMetrics(proxylines, conn)
	}
}

func serveLogs(store *logs.Results) {
	logVerifier := logs.NewLogVerifier(store, expectedTags, copyStringMap(optionalTags), allowListFilteredTags, denyListFilteredTags)

	logsServeMux := http.NewServeMux()
	logsServeMux.HandleFunc("/logs/json_array", handlers.LogJsonArrayHandler(logVerifier))
	logsServeMux.HandleFunc("/logs/json_lines", handlers.LogJsonLinesHandler(logVerifier))

	log.Infof("http logs server listening on %s", proxyAddr)
	if err := http.ListenAndServe(proxyAddr, logsServeMux); err != nil {
		log.Fatal(err.Error())
	}
}

func serveControl(metricStore *metricline.Store, eventStore *eventline.Store, externalEventStore *externalevent.Store, logStore *logs.Results) {
	controlServeMux := http.NewServeMux()

	controlServeMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	controlServeMux.HandleFunc("/metrics", handlers.DumpMetricsHandler(metricStore))
	controlServeMux.HandleFunc("/metrics/diff", handlers.DiffMetricsHandler(metricStore))
	// Based on logs already sent, perform checks on store logs
	// Start by supporting POST parameter expected_log_format
	controlServeMux.HandleFunc("/logs/assert", handlers.LogAssertionHandler(logStore))
	controlServeMux.HandleFunc("/events/assert", handlers.EventAssertionHandler(eventStore))
	controlServeMux.HandleFunc("/events/external/assert", handlers.ExternalEventAssertionHandler(externalEventStore))
	// NOTE: these handler functions attach to the control HTTP server, NOT the TCP server that actually receives data

	log.Infof("http control server listening on %s", controlAddr)
	if err := http.ListenAndServe(controlAddr, controlServeMux); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func copyStringMap(input map[string]string) map[string]string {
	copy := make(map[string]string)
	for k, v := range input {
		copy[k] = v
	}
	return copy
}

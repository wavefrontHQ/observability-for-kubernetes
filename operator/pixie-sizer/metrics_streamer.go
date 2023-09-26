package main

import (
	"bytes"
	"fmt"
	"log"

	"github.com/gogo/protobuf/proto"
	"github.com/nats-io/nats.go"
	prom "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"px.dev/pixie/src/vizier/messages/messagespb"
)

const NatsMetricsChannel = "Metrics"

type MetricsStreamer struct {
	natsConn *nats.Conn
}

func NewMetricStream(natsServer string, clientTLSCertFile string, clientTLSKeyFile string, tlsCAFile string) (*MetricsStreamer, error) {
	natsConn, err := nats.Connect(natsServer, nats.ClientCert(clientTLSCertFile, clientTLSKeyFile), nats.RootCAs(tlsCAFile))
	if err != nil {
		return nil, fmt.Errorf("error creating nats connection: %s", err)
	}
	return &MetricsStreamer{
		natsConn: natsConn,
	}, err
}

func (s *MetricsStreamer) Subscribe(handle func(podName string, metricsFamilies map[string]*prom.MetricFamily)) (func(), error) {
	subscription, err := s.natsConn.Subscribe(NatsMetricsChannel, func(msg *nats.Msg) {
		var metricsMsg messagespb.MetricsMessage
		err := proto.Unmarshal(msg.Data, &metricsMsg)
		if err != nil {
			log.Printf("invalid metrics message: %s", err)
			return
		}
		buf := bytes.NewReader([]byte(metricsMsg.GetPromMetricsText()))
		metricsFamilies, err := (&expfmt.TextParser{}).TextToMetricFamilies(buf)
		if err != nil {
			log.Printf("invalid prometheus metrics: %s", err)
			return
		}
		handle(metricsMsg.GetPodName(), metricsFamilies)
	})
	if err != nil {
		return nil, fmt.Errorf("error creating nats subscriptions to Metrics topic: %s", err)
	}
	return func() {
		subscription.Unsubscribe()
	}, nil
}

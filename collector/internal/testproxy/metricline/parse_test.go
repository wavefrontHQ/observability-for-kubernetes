package metricline_test

import (
	"testing"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/testproxy/metricline"

	"github.com/stretchr/testify/assert"
)

func TestParseMetric(t *testing.T) {
	t.Run("can parse histograms", func(t *testing.T) {
		metric, err := metricline.Parse("!M 1493773500 #20 30 #10 5 request.latency source=\"appServer1\" region=\"us-west\"")
		assert.NoError(t, err)
		assert.Equal(t, "request.latency", metric.Name)
		assert.Equal(t, "1493773500", metric.Timestamp)
		assert.Equal(t, map[string]string{"region": "us-west", "source": "appServer1"}, metric.Tags)
		assert.Equal(t, map[string]string{"30": "20", "5": "10"}, metric.Buckets)
	})

	t.Run("can parse metrics", func(t *testing.T) {
		metric, err := metricline.Parse("system.cpu.loadavg.1m 0.03 1382754475 source=\"test1.wavefront.com\"")
		assert.NoError(t, err)
		assert.Equal(t, "system.cpu.loadavg.1m", metric.Name)
		assert.Equal(t, "0.03", metric.Value)
		assert.Equal(t, "1382754475", metric.Timestamp)
		assert.Equal(t, map[string]string{"source": "test1.wavefront.com"}, metric.Tags)
	})
}

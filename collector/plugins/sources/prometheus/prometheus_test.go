// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package prometheus

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/leadership"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/options"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"

	gm "github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/filter"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/httputil"
)

func TestNoFilters(t *testing.T) {
	src := &prometheusMetricsSource{}

	points, err := src.parseMetrics(testMetricReader())
	require.NoError(t, err, "parsing metrics")
	assert.Equal(t, 8, len(points), "wrong number of points")
}

func TestMetricAllowList(t *testing.T) {
	cfg := filter.Config{
		MetricAllowList: []string{"*seconds.count*"},
	}
	f := filter.FromConfig(cfg)

	src := &prometheusMetricsSource{
		filters: f,
	}

	points, err := src.parseMetrics(testMetricReader())
	require.NoError(t, err, "parsing metrics")
	assert.Equal(t, 1, len(points), "wrong number of points")
}

func TestMetricDenyList(t *testing.T) {
	cfg := filter.Config{
		MetricDenyList: []string{"*seconds.count*"},
	}
	f := filter.FromConfig(cfg)

	src := &prometheusMetricsSource{
		filters: f,
	}

	points, err := src.parseMetrics(testMetricReader())
	require.NoError(t, err, "parsing metrics")
	assert.Equal(t, 7, len(points), "wrong number of points")
}

func TestMetricTagAllowList(t *testing.T) {
	cfg := filter.Config{
		MetricTagAllowList: map[string][]string{"label": {"good"}},
	}
	f := filter.FromConfig(cfg)

	src := &prometheusMetricsSource{
		filters: f,
	}

	points, err := src.parseMetrics(testMetricReader())
	require.NoError(t, err, "parsing metrics")
	assert.Equal(t, 1, len(points), "wrong number of points")
}

func TestMetricTagDenyList(t *testing.T) {
	cfg := filter.Config{
		MetricTagDenyList: map[string][]string{"label": {"ba*"}},
	}
	f := filter.FromConfig(cfg)

	src := &prometheusMetricsSource{
		filters: f,
	}

	points, err := src.parseMetrics(testMetricReader())
	require.NoError(t, err, "parsing metrics")
	assert.Equal(t, 7, len(points), "wrong number of points")
}

func TestTagInclude(t *testing.T) {
	src := &prometheusMetricsSource{
		filters: filter.FromConfig(filter.Config{
			TagInclude: []string{"label"},
		}),
	}

	points, err := src.parseMetrics(testMetricReader())
	require.NoError(t, err, "parsing metrics")
	assert.Equal(t, 8, len(points), "wrong number of points")

	tagCounts := map[string]int{}
	for _, point := range points {
		tags := point.Tags()
		for tagName := range tags {
			tagCounts[tagName] += 1
		}
	}
	assert.Equal(t, 1, len(tagCounts), "the only tags left are 'label'")
	assert.Equal(t, 2, tagCounts["label"], "two metrics have a tag named 'label'")
}

func TestTagExclude(t *testing.T) {
	src := &prometheusMetricsSource{
		filters: filter.FromConfig(filter.Config{
			TagExclude: []string{"label"},
		}),
	}

	points, err := src.parseMetrics(testMetricReader())
	require.NoError(t, err, "parsing metrics")
	assert.Equal(t, 8, len(points), "wrong number of points")

	for _, point := range points {
		_, ok := point.Tags()["label"]
		assert.False(t, ok, point.Tags())
	}
}

func BenchmarkMetricPoint(b *testing.B) {
	filtered := gm.GetOrRegisterCounter("filtered", gm.DefaultRegistry)
	tempTags := map[string]string{"pod_name": "prometheus_pod_xyz", "namespace_name": "default"}
	src := &prometheusMetricsSource{prefix: "prefix."}
	pointBuilder := NewPointBuilder(src, filtered)
	for i := 0; i < b.N; i++ {
		_ = pointBuilder.point("http.requests.total.count", 1.0, 0, "", tempTags)
	}
}

func testMetricReader() *bytes.Reader {
	metricsStr := `
http_request_duration_seconds_bucket{le="0.5"} 0
http_request_duration_seconds_bucket{le="1"} 1
http_request_duration_seconds_bucket{le="2"} 2
http_request_duration_seconds_bucket{le="3"} 3
http_request_duration_seconds_bucket{le="5"} 3
http_request_duration_seconds_bucket{le="+Inf"} 3
http_request_duration_seconds_sum{label="bad"} 6
http_request_duration_seconds_count{label="good"} 3
`
	return bytes.NewReader([]byte(metricsStr))
}

func TestDiscoveredPrometheusMetricSource(t *testing.T) {
	t.Run("static source", func(t *testing.T) {
		ms, err := NewPrometheusMetricsSource("", "", "", "", map[string]string{}, nil, false, httputil.ClientConfig{})

		assert.Nil(t, err)
		assert.False(t, ms.AutoDiscovered(), "prometheus auto-discovery")
	})

	t.Run("discovered source", func(t *testing.T) {
		ms, err := NewPrometheusMetricsSource("", "", "", "some-discovery-method", map[string]string{}, nil, false, httputil.ClientConfig{})

		assert.Nil(t, err)
		assert.True(t, ms.AutoDiscovered(), "prometheus auto-discovery")
	})
}

func Test_prometheusMetricsSource_Scrape(t *testing.T) {
	t.Run("returns a result with current timestamp", func(t *testing.T) {
		nowTime := time.Now()
		// https://medium.com/zus-health/mocking-outbound-http-requests-in-go-youre-probably-doing-it-wrong-60373a38d2aa
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		promMetSource := prometheusMetricsSource{
			metricsURL: fmt.Sprintf("%s/fake/metrics/path", server.URL),
			client:     &http.Client{},
			pps:        gm.NewCounter(),
		}

		result, err := promMetSource.Scrape()
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, result.Timestamp, nowTime)
	})

	t.Run("return an error and increments error counters if client fails to get metrics URL", func(t *testing.T) {
		promMetSource := &prometheusMetricsSource{
			metricsURL: "fake metrics URL",
			client:     &http.Client{},
			eps:        gm.NewCounter(),
		}
		collectErrors.Clear()

		_, scrapeError := promMetSource.Scrape()

		assert.NotNil(t, scrapeError)
		assert.Equal(t, int64(1), collectErrors.Count())
		assert.Equal(t, int64(1), promMetSource.eps.Count())
	})

	t.Run("gets the metrics URL", func(t *testing.T) {
		requestedPath := ""
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			requestedPath = request.URL.Path
			writer.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		promMetSource := prometheusMetricsSource{
			metricsURL: fmt.Sprintf("%s/fake/metrics/path", server.URL),
			client:     &http.Client{},
			pps:        gm.NewCounter(),
		}

		_, err := promMetSource.Scrape()
		assert.NoError(t, err)

		assert.Equal(t, "/fake/metrics/path", requestedPath)
	})

	t.Run("returns an HTTPError and increments error counters on resp error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusBadRequest)
		}))
		defer server.Close()
		promMetSource := &prometheusMetricsSource{
			metricsURL: fmt.Sprintf("%s/fake/metrics/path", server.URL),
			client:     &http.Client{},
			eps:        gm.NewCounter(),
		}
		expectedErr := &HTTPError{
			MetricsURL: fmt.Sprintf("%s/fake/metrics/path", server.URL),
			Status:     "400 Bad Request",
			StatusCode: http.StatusBadRequest,
		}
		collectErrors.Clear()

		_, scrapeError := promMetSource.Scrape()

		assert.Equal(t, expectedErr, scrapeError)
		assert.Equal(t, int64(1), collectErrors.Count())
		assert.Equal(t, int64(1), promMetSource.eps.Count())
	})

	t.Run("returns metrics based on response body and counts number of points", func(t *testing.T) {
		startTimestamp := time.Now().Unix()
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
			writer.Write([]byte(`
fake_metric{} 1
fake_metric{} 1
`))
		}))
		defer server.Close()

		promMetSource := prometheusMetricsSource{
			metricsURL: fmt.Sprintf("%s/fake/metrics/path", server.URL),
			client:     &http.Client{},
			pps:        gm.NewCounter(),
		}

		expectedMetric := wf.NewPoint(
			"fake.metric.value",
			1.0,
			startTimestamp, // not really though
			"",
			nil,
		)
		expectedMetric.SetLabelPairs([]wf.LabelPair{})

		collectedPointsBefore := collectedPoints.Count()
		result, err := promMetSource.Scrape()
		assert.NoError(t, err)
		collectedPointsAfter := collectedPoints.Count()
		assert.Len(t, result.Metrics, 2)
		assert.Equal(t, expectedMetric.Metric, result.Metrics[0].(*wf.Point).Metric)
		assert.Equal(t, expectedMetric.Value, result.Metrics[0].(*wf.Point).Value)
		assert.LessOrEqual(t, expectedMetric.Timestamp, result.Metrics[0].(*wf.Point).Timestamp)
		assert.Equal(t, expectedMetric.Source, result.Metrics[0].(*wf.Point).Source)
		assert.Equal(t, expectedMetric.Tags(), result.Metrics[0].(*wf.Point).Tags())

		assert.Equal(t, int64(2), collectedPointsAfter-collectedPointsBefore)
		assert.Equal(t, int64(2), promMetSource.pps.Count())
	})
}

func Test_prometheusProvider_GetMetricsSources(t *testing.T) {
	t.Run("when use leader election is enabled", func(t *testing.T) {
		t.Run("returns sources when we are the leader", func(t *testing.T) {
			promProvider, _ := NewPrometheusProvider(configuration.PrometheusSourceConfig{
				UseLeaderElection: true,
				URL:               "https://example.local/metrics",
			}, InstanceFromHost)
			util.SetAgentType(options.AllAgentType)
			leadership.SetLeading(true)
			defer leadership.SetLeading(false)

			sources := promProvider.GetMetricsSources()

			assert.Len(t, sources, 1)
		})

		t.Run("does not return sources when we are not the leader", func(t *testing.T) {
			promProvider, _ := NewPrometheusProvider(configuration.PrometheusSourceConfig{
				UseLeaderElection: true,
				URL:               "https://example.local/metrics",
			}, InstanceFromHost)
			util.SetAgentType(options.AllAgentType)

			sources := promProvider.GetMetricsSources()

			assert.Empty(t, sources)
		})
	})

	t.Run("returns sources when use leader election is disabled", func(t *testing.T) {
		promProvider, _ := NewPrometheusProvider(configuration.PrometheusSourceConfig{
			UseLeaderElection: false,
			Discovered:        "something",
			URL:               "https://example.local/metrics",
		}, InstanceFromHost)

		util.SetAgentType(options.AllAgentType)

		sources := promProvider.GetMetricsSources()

		assert.Len(t, sources, 1)
	})

	t.Run("returns one source per instance", func(t *testing.T) {
		promProvider, _ := NewPrometheusProvider(configuration.PrometheusSourceConfig{
			URL:        "http://example.local:2222/metrics",
			Discovered: "something",
		}, func(_ string) ([]Instance, error) {
			return []Instance{{"127.0.0.1:2222", nil}, {"127.0.0.2:2222", nil}}, nil
		})
		util.SetAgentType(options.AllAgentType)

		sources := promProvider.GetMetricsSources()

		assert.Len(t, sources, 2)
		assert.Equal(t, "http://127.0.0.1:2222/metrics", sources[0].(*prometheusMetricsSource).metricsURL)
		assert.Equal(t, "http://127.0.0.2:2222/metrics", sources[1].(*prometheusMetricsSource).metricsURL)
	})

	t.Run("recomputes the sources every time", func(t *testing.T) {
		firstCall := true
		promProvider, _ := NewPrometheusProvider(configuration.PrometheusSourceConfig{
			URL:        "http://example.local:2222/metrics",
			Discovered: "something",
		}, func(_ string) ([]Instance, error) {
			if firstCall {
				firstCall = false
				return []Instance{{"127.0.0.1:2222", nil}}, nil
			} else {
				return []Instance{{"127.0.0.2:2222", nil}}, nil
			}

		})
		util.SetAgentType(options.AllAgentType)

		promProvider.GetMetricsSources()
		sources := promProvider.GetMetricsSources()

		assert.Len(t, sources, 1)
		assert.Contains(t, sources[0].Name(), "http://127.0.0.2:2222/metrics")
	})

	t.Run("sends url.Host into LookupInstances", func(t *testing.T) {
		actualHost := ""
		promProvider, _ := NewPrometheusProvider(configuration.PrometheusSourceConfig{
			URL:        "http://example.local:2222/metrics",
			Discovered: "something",
		}, func(host string) ([]Instance, error) {
			actualHost = host
			return []Instance{{"127.0.0.1:2222", nil}, {"127.0.0.2:2222", nil}}, nil
		})
		util.SetAgentType(options.AllAgentType)

		promProvider.GetMetricsSources()

		assert.Equal(t, "example.local:2222", actualHost)
	})

	t.Run("does not include port if the instances doesn't specify it", func(t *testing.T) {
		promProvider, _ := NewPrometheusProvider(configuration.PrometheusSourceConfig{
			URL:        "https://example.local:8443/metrics",
			Discovered: "something",
		}, func(host string) ([]Instance, error) {
			return []Instance{{"127.0.0.1", nil}}, nil
		})
		util.SetAgentType(options.AllAgentType)

		sources := promProvider.GetMetricsSources()

		assert.Len(t, sources, 1)
		assert.Equal(t, "https://127.0.0.1/metrics", sources[0].(*prometheusMetricsSource).metricsURL)
	})

	t.Run("returns no sources when there is an error looking up instances", func(t *testing.T) {
		promProvider, _ := NewPrometheusProvider(configuration.PrometheusSourceConfig{
			URL:        "https://example.local/metrics",
			Discovered: "something",
		}, func(host string) ([]Instance, error) {
			return nil, errors.New("some error")
		})
		util.SetAgentType(options.AllAgentType)

		sources := promProvider.GetMetricsSources()

		assert.Empty(t, sources)
	})
}

func TestNewPrometheusProvider(t *testing.T) {
	t.Run("errors if prometheus URL is missing", func(t *testing.T) {
		_, err := NewPrometheusProvider(
			configuration.PrometheusSourceConfig{},
			InstanceFromHost,
		)

		assert.Error(t, err)
	})

	t.Run("errors if prometheus URL is invalid", func(t *testing.T) {
		_, err := NewPrometheusProvider(configuration.PrometheusSourceConfig{
			URL: "\x00https://example.local/metrics",
		}, InstanceFromHost)

		assert.Error(t, err)
	})

	t.Run("use configured source as source tag", func(t *testing.T) {
		cfg := configuration.PrometheusSourceConfig{
			URL: "https://example.local",
			Transforms: configuration.Transforms{
				Source: "fake source",
			},
			UseLeaderElection: true,
		}

		leadership.SetLeading(true)
		util.SetAgentType(options.AllAgentType)

		promProvider, err := NewPrometheusProvider(cfg, InstanceFromHost)
		assert.NoError(t, err)
		source := promProvider.GetMetricsSources()[0].(*prometheusMetricsSource)
		assert.Equal(t, "fake source", source.source)
	})

	t.Run("use node name as source tag", func(t *testing.T) {
		cfg := configuration.PrometheusSourceConfig{URL: "https://example.local"}
		_ = os.Setenv(util.NodeNameEnvVar, "fake node name")
		defer os.Unsetenv(util.NodeNameEnvVar)
		leadership.SetLeading(true)
		util.SetAgentType(options.AllAgentType)
		promProvider, err := NewPrometheusProvider(cfg, InstanceFromHost)

		assert.NoError(t, err)
		source := promProvider.GetMetricsSources()[0].(*prometheusMetricsSource)
		assert.Equal(t, "fake node name", source.source)
	})

	t.Run("use 'prom_source' as source tag", func(t *testing.T) {
		cfg := configuration.PrometheusSourceConfig{URL: "https://example.local"}

		leadership.SetLeading(true)
		util.SetAgentType(options.AllAgentType)

		promProvider, err := NewPrometheusProvider(cfg, InstanceFromHost)
		assert.NoError(t, err)
		source := promProvider.GetMetricsSources()[0].(*prometheusMetricsSource)
		assert.Equal(t, "prom_source", source.source)
	})

	t.Run("default name to URL if not configured", func(t *testing.T) {
		cfg := configuration.PrometheusSourceConfig{URL: "http://test-prometheus-url.com"}

		prometheusProvider, err := NewPrometheusProvider(cfg, InstanceFromHost)

		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("%s: http://test-prometheus-url.com", providerName), prometheusProvider.Name())
	})

	t.Run("uses configured provider name", func(t *testing.T) {
		cfg := configuration.PrometheusSourceConfig{Name: "fake name", URL: "http://test-prometheus-url.com"}

		prometheusProvider, err := NewPrometheusProvider(cfg, InstanceFromHost)

		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("%s: fake name", providerName), prometheusProvider.Name())
	})

	t.Run("sources default to not auto discovered", func(t *testing.T) {
		cfg := configuration.PrometheusSourceConfig{
			URL: "http://test-prometheus-url.com",
		}
		leadership.SetLeading(true)
		util.SetAgentType(options.AllAgentType)
		promProvider, _ := NewPrometheusProvider(cfg, InstanceFromHost)

		sources := promProvider.GetMetricsSources()

		assert.False(t, sources[0].AutoDiscovered())
	})

	t.Run("configures sources to auto discovered", func(t *testing.T) {
		cfg := configuration.PrometheusSourceConfig{
			URL:        "http://test-prometheus-url.com",
			Discovered: "fake discovered",
		}

		promProvider, err := NewPrometheusProvider(cfg, InstanceFromHost)

		assert.NoError(t, err)
		source := promProvider.GetMetricsSources()[0].(*prometheusMetricsSource)
		assert.True(t, source.AutoDiscovered())
	})

	t.Run("metrics source defaults with minimal configuration", func(t *testing.T) {
		cfg := configuration.PrometheusSourceConfig{
			URL: "http://test-prometheus-url.com",
		}
		leadership.SetLeading(true)
		util.SetAgentType(options.AllAgentType)

		promProvider, _ := NewPrometheusProvider(cfg, InstanceFromHost)

		source := promProvider.GetMetricsSources()[0].(*prometheusMetricsSource)
		assert.Equal(t, time.Second*30, source.client.Timeout)
		assert.NotNil(t, source.client.Transport)
		assert.Equal(t, "", source.prefix)
		assert.Empty(t, source.tags, "when lookup host is nil, does not add instance tag")
		assert.Equal(t, nil, source.filters)
		assert.Equal(t, "http://test-prometheus-url.com", source.metricsURL)
	})

	t.Run("when lookup host is present, adds host:port as instance tag", func(t *testing.T) {
		cfg := configuration.PrometheusSourceConfig{
			URL: "http://test-prometheus-url.com",
		}
		leadership.SetLeading(true)
		util.SetAgentType(options.AllAgentType)
		promProvider, err := NewPrometheusProvider(cfg, func(host string) ([]Instance, error) {
			return []Instance{{"127.0.0.1:2222", map[string]string{"instance": "127.0.0.1:2222"}}}, nil
		})

		assert.NoError(t, err)
		source := promProvider.GetMetricsSources()[0].(*prometheusMetricsSource)
		assert.Equal(t, 1, len(source.tags))
		assert.Contains(t, source.tags, "instance")
		assert.Equal(t, "127.0.0.1:2222", source.tags["instance"])
	})

	t.Run("when Discovered is present", func(t *testing.T) {
		t.Run("does not use leader election UseLeaderElection is false", func(t *testing.T) {
			cfg := configuration.PrometheusSourceConfig{
				URL:               "http://test-prometheus-url.com",
				UseLeaderElection: false,
				Discovered:        "fake discovered",
			}

			promProvider, _ := NewPrometheusProvider(cfg, InstanceFromHost)

			assert.False(t, promProvider.(*prometheusProvider).useLeaderElection)
		})

		t.Run("uses leader election when UseLeaderElection is true", func(t *testing.T) {
			cfg := configuration.PrometheusSourceConfig{
				URL:               "http://test-prometheus-url.com",
				UseLeaderElection: true,
				Discovered:        "fake discovered",
			}

			promProvider, err := NewPrometheusProvider(cfg, InstanceFromHost)

			assert.NoError(t, err)
			assert.True(t, promProvider.(*prometheusProvider).useLeaderElection)
		})
	})

	t.Run("when Discovered is empty", func(t *testing.T) {
		t.Run("uses leader election even when UseLeaderElection is false", func(t *testing.T) {
			cfg := configuration.PrometheusSourceConfig{
				URL:               "http://test-prometheus-url.com",
				UseLeaderElection: false,
				Discovered:        "",
			}

			promProvider, _ := NewPrometheusProvider(cfg, InstanceFromHost)

			assert.True(t, promProvider.(*prometheusProvider).useLeaderElection)
		})

		t.Run("uses leader election even when UseLeaderElection is true", func(t *testing.T) {
			cfg := configuration.PrometheusSourceConfig{
				URL:               "http://test-prometheus-url.com",
				UseLeaderElection: true,
				Discovered:        "",
			}

			promProvider, _ := NewPrometheusProvider(cfg, InstanceFromHost)

			assert.True(t, promProvider.(*prometheusProvider).useLeaderElection)
		})
	})
}

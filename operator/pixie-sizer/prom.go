package main

import prom "github.com/prometheus/client_model/go"

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

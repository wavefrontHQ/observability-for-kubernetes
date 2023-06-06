package options

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAgentType(t *testing.T) {
	t.Run("ScrapeCluster", func(t *testing.T) {
		assert.True(t, AllAgentType.ScrapeCluster())
		assert.True(t, LegacyAgentType.ScrapeCluster())
		assert.False(t, NodeAgentType.ScrapeCluster())
		assert.True(t, ClusterAgentType.ScrapeCluster())
		assert.False(t, K8sEventsAgentType.ScrapeCluster())
	})

	t.Run("ScrapeAnyNodes", func(t *testing.T) {
		assert.True(t, AllAgentType.ScrapeAnyNodes())
		assert.True(t, LegacyAgentType.ScrapeAnyNodes())
		assert.True(t, NodeAgentType.ScrapeAnyNodes())
		assert.False(t, ClusterAgentType.ScrapeAnyNodes())
		assert.False(t, K8sEventsAgentType.ScrapeAnyNodes())
	})

	t.Run("ScrapeOnlyOwnNode", func(t *testing.T) {
		assert.False(t, AllAgentType.ScrapeOnlyOwnNode())
		assert.True(t, LegacyAgentType.ScrapeOnlyOwnNode())
		assert.True(t, NodeAgentType.ScrapeOnlyOwnNode())
		assert.False(t, ClusterAgentType.ScrapeOnlyOwnNode())
		assert.False(t, K8sEventsAgentType.ScrapeOnlyOwnNode())
	})

	t.Run("ExportKubernetesEvents", func(t *testing.T) {
		assert.False(t, AllAgentType.OnlyExportKubernetesEvents())
		assert.False(t, LegacyAgentType.OnlyExportKubernetesEvents())
		assert.False(t, NodeAgentType.OnlyExportKubernetesEvents())
		assert.False(t, ClusterAgentType.OnlyExportKubernetesEvents())
		assert.True(t, K8sEventsAgentType.OnlyExportKubernetesEvents())
	})

	t.Run("ClusterCollector", func(t *testing.T) {
		assert.True(t, AllAgentType.ClusterCollector())
		assert.True(t, LegacyAgentType.ClusterCollector())
		assert.False(t, NodeAgentType.ClusterCollector())
		assert.True(t, ClusterAgentType.ClusterCollector())
		assert.True(t, K8sEventsAgentType.ClusterCollector())
	})

}

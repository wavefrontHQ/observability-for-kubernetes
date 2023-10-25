package pixie

import (
	"github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
)

func defaultResources(clusterSize string, config Config) Config {
	config.PEMResources = PEMResources[clusterSize]
	config.TableStoreLimits = TableStoreLimits[clusterSize]
	config.KelvinResources = KelvinResources[clusterSize]
	config.QueryBrokerResources = QueryBrokerResources[clusterSize]
	config.NATSResources = NATSResources[clusterSize]
	config.MetadataResources = MetadataResources[clusterSize]
	return config
}

var PEMResources = map[string]v1alpha1.Resources{
	v1alpha1.ClusterSizeSmall: {
		Requests: v1alpha1.Resource{
			CPU:    "50m",
			Memory: "300Mi",
		},
		Limits: v1alpha1.Resource{
			CPU:    "300m",
			Memory: "500Mi",
		},
	},
	v1alpha1.ClusterSizeMedium: {
		Requests: v1alpha1.Resource{
			CPU:    "300m",
			Memory: "500Mi",
		},
		Limits: v1alpha1.Resource{
			CPU:    "750m",
			Memory: "600Mi",
		},
	},
	v1alpha1.ClusterSizeLarge: {
		Requests: v1alpha1.Resource{
			CPU:    "750m",
			Memory: "600Mi",
		},
		Limits: v1alpha1.Resource{
			CPU:    "1",
			Memory: "750Mi",
		},
	},
}

var TableStoreLimits = map[string]v1alpha1.TableStoreLimits{
	v1alpha1.ClusterSizeSmall: {
		TotalMiB:          50,
		HttpEventsPercent: 20,
	},
	v1alpha1.ClusterSizeMedium: {
		TotalMiB:          150,
		HttpEventsPercent: 20,
	},
	v1alpha1.ClusterSizeLarge: {
		TotalMiB:          200,
		HttpEventsPercent: 20,
	},
}

var KelvinResources = map[string]v1alpha1.Resources{
	v1alpha1.ClusterSizeSmall: {
		Requests: v1alpha1.Resource{
			CPU:    "50m",
			Memory: "50Mi",
		},
		Limits: v1alpha1.Resource{
			CPU:    "100m",
			Memory: "100Mi",
		},
	},
	v1alpha1.ClusterSizeMedium: {
		Requests: v1alpha1.Resource{
			CPU:    "100m",
			Memory: "100Mi",
		},
		Limits: v1alpha1.Resource{
			CPU:    "1",
			Memory: "1Gi",
		},
	},
	v1alpha1.ClusterSizeLarge: {
		Requests: v1alpha1.Resource{
			CPU:    "1",
			Memory: "1Gi",
		},
		Limits: v1alpha1.Resource{
			CPU:    "2",
			Memory: "10Gi",
		},
	},
}

var QueryBrokerResources = map[string]v1alpha1.Resources{
	v1alpha1.ClusterSizeSmall: {
		Requests: v1alpha1.Resource{
			CPU:    "50m",
			Memory: "50Mi",
		},
		Limits: v1alpha1.Resource{
			CPU:    "250m",
			Memory: "128Mi",
		},
	},
	v1alpha1.ClusterSizeMedium: {
		Requests: v1alpha1.Resource{
			CPU:    "250m",
			Memory: "128Mi",
		},
		Limits: v1alpha1.Resource{
			CPU:    "500m",
			Memory: "256Mi",
		},
	},
	v1alpha1.ClusterSizeLarge: {
		Requests: v1alpha1.Resource{
			CPU:    "500m",
			Memory: "256Mi",
		},
		Limits: v1alpha1.Resource{
			CPU:    "1",
			Memory: "512Mi",
		},
	},
}

var NATSResources = map[string]v1alpha1.Resources{
	v1alpha1.ClusterSizeSmall: {
		Requests: v1alpha1.Resource{
			CPU:    "5m",
			Memory: "5Mi",
		},
		Limits: v1alpha1.Resource{
			CPU:    "25m",
			Memory: "25Mi",
		},
	},
	v1alpha1.ClusterSizeMedium: {
		Requests: v1alpha1.Resource{
			CPU:    "10m",
			Memory: "10Mi",
		},
		Limits: v1alpha1.Resource{
			CPU:    "50m",
			Memory: "250Mi",
		},
	},
	v1alpha1.ClusterSizeLarge: {
		Requests: v1alpha1.Resource{
			CPU:    "50m",
			Memory: "250Mi",
		},
		Limits: v1alpha1.Resource{
			CPU:    "100m",
			Memory: "500Mi",
		},
	},
}

var MetadataResources = map[string]v1alpha1.Resources{
	v1alpha1.ClusterSizeSmall: {
		Requests: v1alpha1.Resource{
			CPU:    "25m",
			Memory: "128Mi",
		},
		Limits: v1alpha1.Resource{
			CPU:    "250m",
			Memory: "250Mi",
		},
	},
	v1alpha1.ClusterSizeMedium: {
		Requests: v1alpha1.Resource{
			CPU:    "250m",
			Memory: "250Mi",
		},
		Limits: v1alpha1.Resource{
			CPU:    "500m",
			Memory: "1Gi",
		},
	},
	v1alpha1.ClusterSizeLarge: {
		Requests: v1alpha1.Resource{
			CPU:    "500m",
			Memory: "1Gi",
		},
		Limits: v1alpha1.Resource{
			CPU:    "1",
			Memory: "2Gi",
		},
	},
}

var CertProvisionerJobResources = map[string]v1alpha1.Resources{
	v1alpha1.ClusterSizeSmall: {
		Requests: v1alpha1.Resource{
			CPU:    "50m",
			Memory: "10Mi",
		},
		Limits: v1alpha1.Resource{
			CPU:    "100m",
			Memory: "100Mi",
		},
	},
	v1alpha1.ClusterSizeMedium: {
		Requests: v1alpha1.Resource{
			CPU:    "50m",
			Memory: "10Mi",
		},
		Limits: v1alpha1.Resource{
			CPU:    "100m",
			Memory: "100Mi",
		},
	},
	v1alpha1.ClusterSizeLarge: {
		Requests: v1alpha1.Resource{
			CPU:    "50m",
			Memory: "10Mi",
		},
		Limits: v1alpha1.Resource{
			CPU:    "100m",
			Memory: "100Mi",
		},
	},
}

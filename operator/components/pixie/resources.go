package pixie

import (
	"github.com/wavefronthq/observability-for-kubernetes/operator/api/common"
	"github.com/wavefronthq/observability-for-kubernetes/operator/api/wavefront/v1alpha1"
)

func defaultResources(clusterSize string, config Config) Config {
	config.PEMResources = PEMResources[clusterSize]
	config.TableStoreLimits = TableStoreLimits[clusterSize]
	config.KelvinResources = KelvinResources[clusterSize]
	config.QueryBrokerResources = QueryBrokerResources[clusterSize]
	config.NATSResources = NATSResources[clusterSize]
	config.MetadataResources = MetadataResources[clusterSize]
	config.CertProvisionerJobResources = CertProvisionerJobResources[clusterSize]
	return config
}

var PEMResources = map[string]common.ContainerResources{
	v1alpha1.ClusterSizeSmall: {
		Requests: common.ContainerResource{
			CPU:    "100m",
			Memory: "300Mi",
		},
		Limits: common.ContainerResource{
			CPU:    "500m",
			Memory: "500Mi",
		},
	},
	v1alpha1.ClusterSizeMedium: {
		Requests: common.ContainerResource{
			CPU:    "500m",
			Memory: "500Mi",
		},
		Limits: common.ContainerResource{
			CPU:    "1",
			Memory: "600Mi",
		},
	},
	v1alpha1.ClusterSizeLarge: {
		Requests: common.ContainerResource{
			CPU:    "1",
			Memory: "600Mi",
		},
		Limits: common.ContainerResource{
			CPU:    "2",
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

var KelvinResources = map[string]common.ContainerResources{
	v1alpha1.ClusterSizeSmall: {
		Requests: common.ContainerResource{
			CPU:    "50m",
			Memory: "50Mi",
		},
		Limits: common.ContainerResource{
			CPU:    "100m",
			Memory: "100Mi",
		},
	},
	v1alpha1.ClusterSizeMedium: {
		Requests: common.ContainerResource{
			CPU:    "100m",
			Memory: "100Mi",
		},
		Limits: common.ContainerResource{
			CPU:    "1",
			Memory: "1Gi",
		},
	},
	v1alpha1.ClusterSizeLarge: {
		Requests: common.ContainerResource{
			CPU:    "1",
			Memory: "1Gi",
		},
		Limits: common.ContainerResource{
			CPU:    "2",
			Memory: "10Gi",
		},
	},
}

var QueryBrokerResources = map[string]common.ContainerResources{
	v1alpha1.ClusterSizeSmall: {
		Requests: common.ContainerResource{
			CPU:    "50m",
			Memory: "50Mi",
		},
		Limits: common.ContainerResource{
			CPU:    "500m",
			Memory: "128Mi",
		},
	},
	v1alpha1.ClusterSizeMedium: { // TODO update
		Requests: common.ContainerResource{
			CPU:    "500m",
			Memory: "128Mi",
		},
		Limits: common.ContainerResource{
			CPU:    "1",
			Memory: "256Mi",
		},
	},
	v1alpha1.ClusterSizeLarge: { // TODO update
		Requests: common.ContainerResource{
			CPU:    "1",
			Memory: "256Mi",
		},
		Limits: common.ContainerResource{
			CPU:    "2",
			Memory: "512Mi",
		},
	},
}

var NATSResources = map[string]common.ContainerResources{
	v1alpha1.ClusterSizeSmall: {
		Requests: common.ContainerResource{
			CPU:    "100m",
			Memory: "5Mi",
		},
		Limits: common.ContainerResource{
			CPU:    "500m",
			Memory: "25Mi",
		},
	},
	v1alpha1.ClusterSizeMedium: {
		Requests: common.ContainerResource{
			CPU:    "500m",
			Memory: "10Mi",
		},
		Limits: common.ContainerResource{
			CPU:    "1",
			Memory: "250Mi",
		},
	},
	v1alpha1.ClusterSizeLarge: {
		Requests: common.ContainerResource{
			CPU:    "1",
			Memory: "250Mi",
		},
		Limits: common.ContainerResource{
			CPU:    "2",
			Memory: "500Mi",
		},
	},
}

var MetadataResources = map[string]common.ContainerResources{
	v1alpha1.ClusterSizeSmall: {
		Requests: common.ContainerResource{
			CPU:    "25m",
			Memory: "128Mi",
		},
		Limits: common.ContainerResource{
			CPU:    "250m",
			Memory: "250Mi",
		},
	},
	v1alpha1.ClusterSizeMedium: {
		Requests: common.ContainerResource{
			CPU:    "250m",
			Memory: "250Mi",
		},
		Limits: common.ContainerResource{
			CPU:    "500m",
			Memory: "1Gi",
		},
	},
	v1alpha1.ClusterSizeLarge: {
		Requests: common.ContainerResource{
			CPU:    "500m",
			Memory: "1Gi",
		},
		Limits: common.ContainerResource{
			CPU:    "1",
			Memory: "2Gi",
		},
	},
}

var CertProvisionerJobResources = map[string]common.ContainerResources{
	v1alpha1.ClusterSizeSmall: {
		Requests: common.ContainerResource{
			CPU:    "50m",
			Memory: "10Mi",
		},
		Limits: common.ContainerResource{
			CPU:    "100m",
			Memory: "100Mi",
		},
	},
	v1alpha1.ClusterSizeMedium: {
		Requests: common.ContainerResource{
			CPU:    "50m",
			Memory: "10Mi",
		},
		Limits: common.ContainerResource{
			CPU:    "100m",
			Memory: "100Mi",
		},
	},
	v1alpha1.ClusterSizeLarge: {
		Requests: common.ContainerResource{
			CPU:    "50m",
			Memory: "10Mi",
		},
		Limits: common.ContainerResource{
			CPU:    "100m",
			Memory: "100Mi",
		},
	},
}

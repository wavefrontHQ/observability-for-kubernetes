package sinks

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestCleanTags(t *testing.T) {
	t.Run("excludes tags in the exclude tag list", func(t *testing.T) {
		for _, excludedTagName := range excludeTagList {
			actual := map[string]string{excludedTagName: "some-value"}
			cleanTags(actual, []string{}, maxWavefrontTags)
			assert.Equal(t, map[string]string{}, actual)
		}
	})

	t.Run("excludes empty tags", func(t *testing.T) {
		actual := map[string]string{"good-tag": ""}
		cleanTags(actual, []string{}, maxWavefrontTags)
		assert.Equal(t, map[string]string{}, actual)
	})

	t.Run("de-duplicates tag values >= min dedupe value length characters when over capacity", func(t *testing.T) {
		tagGreaterThanMinLen := "some.hostname"
		assert.True(t, len(tagGreaterThanMinLen) >= minDedupeTagValueLen)

		tagEqualMinLen := "host1"
		assert.True(t, len(tagEqualMinLen) == minDedupeTagValueLen)

		tagLessThanMinLen := "host"
		assert.True(t, len(tagLessThanMinLen) < minDedupeTagValueLen)

		t.Run("when the tag names are different lengths", func(t *testing.T) {
			actual := map[string]string{"label.long-tag-name": tagGreaterThanMinLen, "label.shrt-tg": tagGreaterThanMinLen}
			cleanTags(actual, []string{}, 1)
			assert.Equal(t, map[string]string{"label.shrt-tg": tagGreaterThanMinLen}, actual)
		})

		t.Run("when the tag names of the same length", func(t *testing.T) {
			actual := map[string]string{"label.dup2": tagGreaterThanMinLen, "label.dup1": tagGreaterThanMinLen}
			cleanTags(actual, []string{}, 1)
			assert.Equal(t, map[string]string{"label.dup1": tagGreaterThanMinLen}, actual)
		})

		t.Run("when the duplicated values are < min len characters", func(t *testing.T) {
			actual := map[string]string{"a-tag": tagLessThanMinLen, "b-tag": tagLessThanMinLen}
			cleanTags(actual, []string{}, 1)
			assert.Equal(t, map[string]string{"a-tag": tagLessThanMinLen, "b-tag": tagLessThanMinLen}, actual)
		})

		t.Run("when the duplicated values are equal min len characters", func(t *testing.T) {
			actual := map[string]string{"label.a-tag": tagEqualMinLen, "label.b-tag": tagEqualMinLen}
			cleanTags(actual, []string{}, 1)
			assert.Equal(t, map[string]string{"label.a-tag": tagEqualMinLen}, actual)
		})

		t.Run("when one of the duplicated values is in the tag include list", func(t *testing.T) {
			actual := map[string]string{"nodename": "same-value", "hostname": "same-value"}
			cleanTags(actual, []string{}, 1)
			assert.Equal(t, map[string]string{"nodename": "same-value"}, actual)
		})

		t.Run("when duplicated values have non label.* tag names", func(t *testing.T) {
			actual := map[string]string{"test-tag-name": "same-value", "label.test-tag-name": "same-value"}
			cleanTags(actual, []string{}, 1)
			assert.Equal(t, map[string]string{"test-tag-name": "same-value"}, actual)

			actual = map[string]string{"label.test-tag-name": "same-value", "test-tag-name": "same-value"}
			cleanTags(actual, []string{}, 1)
			assert.Equal(t, map[string]string{"test-tag-name": "same-value"}, actual)

			actual = map[string]string{
				"test-tag-name":         "same-value",
				"another-test-tag-name": "same-value",
				"label.test-tag-name":   "same-value",
			}
			cleanTags(actual, []string{}, 2)
			assert.Equal(t, map[string]string{
				"test-tag-name":         "same-value",
				"another-test-tag-name": "same-value",
			}, actual)
		})

		t.Run("when under the max capacity", func(t *testing.T) {
			actual := map[string]string{"a-tag": tagLessThanMinLen, "b-tag": tagLessThanMinLen}
			cleanTags(actual, []string{}, 2)
			assert.Equal(t, map[string]string{"a-tag": tagLessThanMinLen, "b-tag": tagLessThanMinLen}, actual)
		})
	})

	t.Run("limits example IaaS node info metric tags to max capacity ", func(t *testing.T) {
		t.Run("GKE example", func(t *testing.T) {
			actual := map[string]string{
				"nodename":                               "gke-mamichael-cluster-5-default-pool-5592f664-mkrr",
				"node_role":                              "worker",
				"os_image":                               "Container-Optimized OS from Google",
				"kubelet_version":                        "v1.23.8-gke.1900",
				"pod_cidr":                               "10.96.2.0/24",
				"internal_ip":                            "10.40.56.17",
				"kernel_version":                         "5.10.127+",
				"provider_id":                            "gce://wavefront-gcp-dev/us-central1-c/gke-mamichael-cluster-5-default-pool-5592f664-mkrr",
				"label.beta.kubernetes.io/arch":          "amd64",
				"label.beta.kubernetes.io/instance-type": "e2-standard-2",
				"label.beta.kubernetes.io/os":            "linux",
				"label.cloud.google.com/gke-boot-disk":   "pd-standard",
				"label.cloud.google.com/gke-container-runtime":   "containerd",
				"label.cloud.google.com/gke-cpu-scaling-level":   "2",
				"label.cloud.google.com/gke-max-pods-per-node":   "110",
				"label.cloud.google.com/gke-nodepool":            "default-pool",
				"label.cloud.google.com/gke-os-distribution":     "cos",
				"label.cloud.google.com/machine-family":          "e2",
				"label.failure-domain.beta.kubernetes.io/region": "us-central1",
				"label.failure-domain.beta.kubernetes.io/zone":   "us-central1-c",
				"label.kubernetes.io/arch":                       "amd64",
				"label.kubernetes.io/hostname":                   "gke-mamichael-cluster-5-default-pool-5592f664-3op5",
				"label.kubernetes.io/os":                         "linux",
				"label.node.kubernetes.io/instance-type":         "e2-standard-2",
				"label.topology.gke.io/zone":                     "us-central1-c",
				"label.topology.kubernetes.io/region":            "us-central1",
				"label.topology.kubernetes.io/zone":              "us-central1-c"}

			expectedCleanedTags := map[string]string{
				"nodename":                            "gke-mamichael-cluster-5-default-pool-5592f664-mkrr",
				"node_role":                           "worker",
				"os_image":                            "Container-Optimized OS from Google",
				"kubelet_version":                     "v1.23.8-gke.1900",
				"pod_cidr":                            "10.96.2.0/24",
				"internal_ip":                         "10.40.56.17",
				"kernel_version":                      "5.10.127+",
				"provider_id":                         "gce://wavefront-gcp-dev/us-central1-c/gke-mamichael-cluster-5-default-pool-5592f664-mkrr",
				"label.cloud.google.com/gke-nodepool": "default-pool",
				"label.cloud.google.com/gke-os-distribution": "cos",
				"label.cloud.google.com/machine-family":      "e2",
				"label.kubernetes.io/arch":                   "amd64",
				"label.kubernetes.io/hostname":               "gke-mamichael-cluster-5-default-pool-5592f664-3op5",
				"label.kubernetes.io/os":                     "linux",
				"label.node.kubernetes.io/instance-type":     "e2-standard-2",
				"label.topology.gke.io/zone":                 "us-central1-c",
				"label.topology.kubernetes.io/region":        "us-central1",
			}

			cleanTags(actual, []string{}, maxWavefrontTags)
			require.Equal(t, expectedCleanedTags, actual)
			require.Equal(t, maxWavefrontTags, len(actual))
		})

		t.Run("AKS example", func(t *testing.T) {
			actual := map[string]string{
				"host_id":                              "",
				"hostname":                             "aks-agentpool-18535100-vmss000000",
				"node_role":                            "worker",
				"nodename":                             "aks-agentpool-18535100-vmss000000",
				"resource_id":                          "/",
				"type":                                 "node",
				"test":                                 "node",
				"label.agentpool":                      "agentpool",
				"label.beta.kubernetes.io/arch":        "amd64",
				"label.beta.kubernetes.io/os":          "linux",
				"label.kubernetes.azure.com/agentpool": "agentpool",
				"label.kubernetes.azure.com/cluster":   "MC_K8sSaaS_k8s-upgrade-test_westus2",
				"label.kubernetes.azure.com/kubelet-identity-client-id": "80e47095-2878-4d98-9758-9dbfab610463",
				"label.kubernetes.azure.com/mode":                       "system",
				"label.kubernetes.azure.com/node-image-version":         "AKSUbuntu-2204gen2containerd-202304.10.0",
				"label.kubernetes.azure.com/os-sku":                     "Ubuntu",
				"label.kubernetes.azure.com/role":                       "agent",
				"label.kubernetes.azure.com/storageprofile":             "managed",
				"label.kubernetes.azure.com/storagetier":                "Premium_LRS",
				"label.kubernetes.io/hostname":                          "aks-agentpool-18535100-vmss000000",
				"label.kubernetes.io/os":                                "linux",
				"label.kubernetes.io/role":                              "agent",
				"label.node-role.kubernetes.io/agent":                   "",
				"label.node.kubernetes.io/instance-type":                "Standard_DS2_v2",
				"label.storageprofile":                                  "managed",
				"label.storagetier":                                     "Premium_LRS",
				"label.topology.disk.csi.azure.com/zone":                "westus2-1",
				"label.topology.kubernetes.io/region":                   "westus2",
				"label.topology.kubernetes.io/zone":                     "westus2-1",
			}

			expected := map[string]string{
				"label.agentpool": "agentpool",
				"label.kubernetes.azure.com/kubelet-identity-client-id": "80e47095-2878-4d98-9758-9dbfab610463",
				"label.kubernetes.azure.com/mode":                       "system",
				"label.kubernetes.azure.com/node-image-version":         "AKSUbuntu-2204gen2containerd-202304.10.0",
				"label.kubernetes.azure.com/os-sku":                     "Ubuntu",
				"label.kubernetes.io/hostname":                          "aks-agentpool-18535100-vmss000000",
				"label.kubernetes.io/os":                                "linux",
				"label.kubernetes.io/role":                              "agent",
				"label.node.kubernetes.io/instance-type":                "Standard_DS2_v2",
				"label.storageprofile":                                  "managed",
				"label.storagetier":                                     "Premium_LRS",
				"label.topology.kubernetes.io/region":                   "westus2",
				"label.topology.kubernetes.io/zone":                     "westus2-1",
				"node_role":                                             "worker",
				"nodename":                                              "aks-agentpool-18535100-vmss000000",
				"test":                                                  "node",
				"type":                                                  "node",
			}

			cleanTags(actual, []string{}, maxWavefrontTags)
			assert.Equal(t, maxWavefrontTags, len(actual))
			assert.Equal(t, expected, actual)
		})

		t.Run("EKS example", func(t *testing.T) {
			actual := map[string]string{
				"node_role":                              "worker",
				"nodename":                               "ip-192-168-12-242.us-west-2.compute.internal",
				"type":                                   "node",
				"pod_cidr":                               "10.96.2.0/24",
				"internal_ip":                            "10.40.56.17",
				"kernel_version":                         "5.10.127+",
				"foo":                                    "bar",
				"test":                                   "tester",
				"czar":                                   "aljkssljfdk",
				"label.alpha.eksctl.io/cluster-name":     "k8s-saas-team-ci",
				"label.alpha.eksctl.io/instance-id":      "i-00ba63d14a98f141d",
				"label.alpha.eksctl.io/nodegroup-name":   "arm-group",
				"label.beta.kubernetes.io/arch":          "arm64",
				"label.beta.kubernetes.io/instance-type": "m6g.medium",
				"label.beta.kubernetes.io/os":            "linux",
				"label.failure-domain.beta.kubernetes.io/region": "us-west-2",
				"label.failure-domain.beta.kubernetes.io/zone":   "us-west-2c",
				"label.kubernetes.io/arch":                       "arm64",
				"label.kubernetes.io/hostname":                   "ip-192-168-12-242.us-west-2.compute.internal",
				"label.kubernetes.io/os":                         "linux",
				"label.node-lifecycle":                           "on-demand",
				"label.node.kubernetes.io/instance-type":         "m6g.medium",
				"label.topology.kubernetes.io/region":            "us-west-2",
				"label.topology.kubernetes.io/zone":              "us-west-2c",
			}

			expected := map[string]string{
				"node_role":                              "worker",
				"nodename":                               "ip-192-168-12-242.us-west-2.compute.internal",
				"type":                                   "node",
				"pod_cidr":                               "10.96.2.0/24",
				"internal_ip":                            "10.40.56.17",
				"kernel_version":                         "5.10.127+",
				"foo":                                    "bar",
				"test":                                   "tester",
				"czar":                                   "aljkssljfdk",
				"label.alpha.eksctl.io/nodegroup-name":   "arm-group",
				"label.kubernetes.io/arch":               "arm64",
				"label.kubernetes.io/hostname":           "ip-192-168-12-242.us-west-2.compute.internal",
				"label.kubernetes.io/os":                 "linux",
				"label.node-lifecycle":                   "on-demand",
				"label.node.kubernetes.io/instance-type": "m6g.medium",
				"label.topology.kubernetes.io/region":    "us-west-2",
				"label.topology.kubernetes.io/zone":      "us-west-2c",
			}

			cleanTags(actual, []string{}, maxWavefrontTags)
			require.Equal(t, maxWavefrontTags, len(actual))
			assert.Equal(t, expected, actual)
		})

		t.Run("EKS example v1.25", func(t *testing.T) {
			actual := map[string]string{
				"node_role":                              "worker",
				"nodename":                               "ip-10-10-0-1.us-west-2.compute.internal",
				"type":                                   "node",
				"pod_cidr":                               "10.96.2.0/24",
				"internal_ip":                            "10.10.0.1",
				"kernel_version":                         "5.10.123-123.333.amzn2.aarch64",
				"foo":                                    "bar",
				"test":                                   "tester",
				"czar":                                   "aljkssljfdk",
				"car":                                    "aljkfdk",
				"sometag":                                "somevalue",
				"label.alpha.eksctl.io/cluster-name":     "k8s-saas-team-ci",
				"label.alpha.eksctl.io/nodegroup-name":   "x86-nodes",
				"label.beta.kubernetes.io/arch":          "amd64",
				"label.beta.kubernetes.io/instance-type": "t3.medium",
				"label.beta.kubernetes.io/os":            "linux",
				"label.eks.amazonaws.com/capacityType":   "ON_DEMAND",
				"label.eks.amazonaws.com/nodegroup":      "x86-nodes",
				"label.eks.amazonaws.com/nodegroup-image":             "ami-07d047ed27809c968",
				"label.eks.amazonaws.com/sourceLaunchTemplateId":      "lt-05481b0402ded0449",
				"label.eks.amazonaws.com/sourceLaunchTemplateVersion": "1",
				"label.failure-domain.beta.kubernetes.io/region":      "us-west-2",
				"label.failure-domain.beta.kubernetes.io/zone":        "us-west-2c",
				"label.k8s.io/cloud-provider-aws":                     "abcdefg123456789abcdefg123456789",
				"label.kubernetes.io/arch":                            "amd64",
				"label.kubernetes.io/hostname":                        "ip-10-10-0-2.us-west-2.compute.internal",
				"label.kubernetes.io/os":                              "linux",
				"label.node.kubernetes.io/instance-type":              "t3.medium",
				"label.topology.kubernetes.io/region":                 "us-west-2",
				"label.topology.kubernetes.io/zone":                   "us-west-2c",
			}

			expected := map[string]string{
				"node_role":                              "worker",
				"nodename":                               "ip-10-10-0-1.us-west-2.compute.internal",
				"type":                                   "node",
				"pod_cidr":                               "10.96.2.0/24",
				"internal_ip":                            "10.10.0.1",
				"kernel_version":                         "5.10.123-123.333.amzn2.aarch64",
				"foo":                                    "bar",
				"test":                                   "tester",
				"czar":                                   "aljkssljfdk",
				"car":                                    "aljkfdk",
				"sometag":                                "somevalue",
				"label.kubernetes.io/arch":               "amd64",
				"label.kubernetes.io/os":                 "linux",
				"label.kubernetes.io/hostname":           "ip-10-10-0-2.us-west-2.compute.internal",
				"label.node.kubernetes.io/instance-type": "t3.medium",
				"label.topology.kubernetes.io/region":    "us-west-2",
				"label.topology.kubernetes.io/zone":      "us-west-2c",
			}

			cleanTags(actual, []string{}, maxWavefrontTags)
			require.Equal(t, maxWavefrontTags, len(actual))
			assert.Equal(t, expected, actual)
		})

		t.Run("GKE example to keep Tags specified in the tagGuaranteeList", func(t *testing.T) {
			actual := map[string]string{
				"nodename":                               "gke-mamichael-cluster-5-default-pool-5592f664-mkrr",
				"node_role":                              "worker",
				"os_image":                               "Container-Optimized OS from Google",
				"kubelet_version":                        "v1.23.8-gke.1900",
				"pod_cidr":                               "10.96.2.0/24",
				"internal_ip":                            "10.40.56.17",
				"kernel_version":                         "5.10.127+",
				"provider_id":                            "gce://wavefront-gcp-dev/us-central1-c/gke-mamichael-cluster-5-default-pool-5592f664-mkrr",
				"label.beta.kubernetes.io/arch":          "amd64",
				"label.beta.kubernetes.io/instance-type": "e2-standard-2",
				"label.beta.kubernetes.io/os":            "linux",
				"label.cloud.google.com/gke-boot-disk":   "pd-standard",
				"label.cloud.google.com/gke-container-runtime":   "containerd",
				"label.cloud.google.com/gke-cpu-scaling-level":   "2",
				"label.cloud.google.com/gke-max-pods-per-node":   "110",
				"label.cloud.google.com/gke-nodepool":            "default-pool",
				"label.cloud.google.com/gke-os-distribution":     "cos",
				"label.cloud.google.com/machine-family":          "e2",
				"label.failure-domain.beta.kubernetes.io/region": "us-central1",
				"label.failure-domain.beta.kubernetes.io/zone":   "us-central1-c",
				"label.kubernetes.io/arch":                       "amd64",
				"label.kubernetes.io/hostname":                   "gke-mamichael-cluster-5-default-pool-5592f664-3op5",
				"label.kubernetes.io/os":                         "linux",
				"label.node.kubernetes.io/instance-type":         "e2-standard-2",
				"label.topology.gke.io/zone":                     "us-central1-c",
				"label.topology.kubernetes.io/region":            "us-central1",
				"label.topology.kubernetes.io/zone":              "us-central1-c"}

			tagGuaranteeList := []string{"label.failure-domain.beta.kubernetes.io/zone"}
			expectedCleanedTags := map[string]string{
				"nodename":        "gke-mamichael-cluster-5-default-pool-5592f664-mkrr",
				"node_role":       "worker",
				"os_image":        "Container-Optimized OS from Google",
				"kubelet_version": "v1.23.8-gke.1900",
				"pod_cidr":        "10.96.2.0/24",
				"internal_ip":     "10.40.56.17",
				"kernel_version":  "5.10.127+",
				"provider_id":     "gce://wavefront-gcp-dev/us-central1-c/gke-mamichael-cluster-5-default-pool-5592f664-mkrr",
				"label.cloud.google.com/gke-os-distribution":   "cos",
				"label.cloud.google.com/machine-family":        "e2",
				"label.kubernetes.io/arch":                     "amd64",
				"label.kubernetes.io/hostname":                 "gke-mamichael-cluster-5-default-pool-5592f664-3op5",
				"label.kubernetes.io/os":                       "linux",
				"label.node.kubernetes.io/instance-type":       "e2-standard-2",
				"label.topology.gke.io/zone":                   "us-central1-c",
				"label.topology.kubernetes.io/region":          "us-central1",
				"label.failure-domain.beta.kubernetes.io/zone": "us-central1-c"}
			cleanTags(actual, tagGuaranteeList, maxWavefrontTags)
			assert.Equal(t, maxWavefrontTags, len(actual))
			assert.Equal(t, expectedCleanedTags, actual)
		})
	})
}

func TestIsAnEmptyTag(t *testing.T) {
	assert.True(t, isAnEmptyTag(""))
	assert.True(t, isAnEmptyTag("/"))
	assert.True(t, isAnEmptyTag("-"))
}

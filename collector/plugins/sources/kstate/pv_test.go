package kstate

import (
	"sort"
	"testing"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setupBasicPV() *v1.PersistentVolume {
	return &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "pv1",
			Labels: nil,
		},
		Spec: v1.PersistentVolumeSpec{
			Capacity: map[v1.ResourceName]resource.Quantity{
				v1.ResourceStorage: *resource.NewQuantity(4, resource.BinarySI),
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{},
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
				v1.ReadOnlyMany,
			},
			ClaimRef:                      nil,
			PersistentVolumeReclaimPolicy: "",
			StorageClassName:              "",
			MountOptions:                  nil,
			VolumeMode:                    nil,
			NodeAffinity:                  nil,
		},
		Status: v1.PersistentVolumeStatus{
			Phase:   v1.VolumeFailed,
			Message: "",
			Reason:  "",
		},
	}
}

func basicPVBuilderInput() (volume *v1.PersistentVolume, transforms configuration.Transforms, timestamp int64, tags map[string]string) {
	return setupBasicPV(),
		configuration.Transforms{Prefix: "kubernetes.", Source: "test-source-for-pv"},
		0,
		map[string]string{
			"pv-tag1": "value1",
			"pv-tag2": "value2",
			"pv-tag3": "value3",
		}
}

func TestPointsForPV(t *testing.T) {
	t.Run("test for basic PV", func(t *testing.T) {
		testPV := setupBasicPV()
		actualWFPoints := pointsForPV(testPV, configuration.Transforms{Prefix: "kubernetes."})
		assert.Equal(t, 5, len(actualWFPoints))

		expectedMetricNames := []string{
			"kubernetes.pv.capacity_bytes",
			"kubernetes.pv.status.phase",
			"kubernetes.pv.info",
			"kubernetes.pv.access_mode",
			"kubernetes.pv.access_mode",
		}

		var actualMetricNames []string

		for _, point := range actualWFPoints {
			actualMetricNames = append(actualMetricNames, point.Name())
		}

		sort.Strings(expectedMetricNames)
		sort.Strings(actualMetricNames)
		assert.Equal(t, expectedMetricNames, actualMetricNames)
	})

	t.Run("metric tags and values", func(t *testing.T) {
		t.Run("buildPVCapacityBytes has value from resource storage in capacity", func(t *testing.T) {
			actualMetric := buildPVCapacityBytes(basicPVBuilderInput())
			expectedMetric := []wf.Metric{
				metricPoint(
					"kubernetes.",
					"pv.capacity_bytes",
					4.0,
					0.0,
					"test-source-for-pv",
					map[string]string{
						"pv-tag1": "value1",
						"pv-tag2": "value2",
						"pv-tag3": "value3",
					},
				),
			}
			assert.Equal(t, expectedMetric, actualMetric)
		})

		t.Run("buildPVInfo has shared tags by default", func(t *testing.T) {
			actualMetric := buildPVInfo(basicPVBuilderInput())
			expectedMetric := metricPoint(
				"kubernetes.",
				"pv.info",
				1.0,
				0.0,
				"test-source-for-pv",
				map[string]string{
					"pv-tag1": "value1",
					"pv-tag2": "value2",
					"pv-tag3": "value3",
				},
			)
			assert.Equal(t, expectedMetric, actualMetric)
		})

		t.Run("buildPVInfo has tags based on what volume source fields are set", func(t *testing.T) {
			t.Run("CephFS", func(t *testing.T) {
				volume, transforms, value, tags := basicPVBuilderInput()
				volume.Spec.PersistentVolumeSource.CephFS = &v1.CephFSPersistentVolumeSource{
					Path: "test-path",
				}
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"cephfs_path": "test-path"})
			})

			t.Run("CSI.Driver", func(t *testing.T) {
				volume, transforms, value, tags := basicPVBuilderInput()
				volume.Spec.PersistentVolumeSource.CSI = &v1.CSIPersistentVolumeSource{
					Driver:       "test-driver",
					VolumeHandle: "test-volume-handle",
				}
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"csi_driver": "test-driver"})
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"csi_volume_handle": "test-volume-handle"})
			})

			t.Run("HostPath, with type only if it is set", func(t *testing.T) {
				volume, transforms, value, tags := basicPVBuilderInput()
				volume.Spec.PersistentVolumeSource.HostPath = &v1.HostPathVolumeSource{
					Path: "test-host-path",
					Type: nil,
				}
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"host_path": "test-host-path"})
				buildPVInfoAndAssertTagsNotPresent(volume, transforms, value, tags)(t, []string{"host_path_type"})

				testHostPath := v1.HostPathDirectoryOrCreate
				volume.Spec.PersistentVolumeSource.HostPath.Type = &testHostPath
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"host_path_type": string(v1.HostPathDirectoryOrCreate)})
			})

			t.Run("ISCSI, with initiator name only if it is set", func(t *testing.T) {
				volume, transforms, value, tags := basicPVBuilderInput()
				volume.Spec.PersistentVolumeSource.ISCSI = &v1.ISCSIPersistentVolumeSource{
					TargetPortal:  "test-target-portal",
					IQN:           "test-iqn",
					Lun:           15,
					InitiatorName: nil,
				}
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"iscsi_target_portal": "test-target-portal"})
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"iscsi_iqn": "test-iqn"})
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"iscsi_lun": "15"})
				buildPVInfoAndAssertTagsNotPresent(volume, transforms, value, tags)(t, []string{"iscsi_initiator_name"})

				testInitiatorName := "test-initiator-name"
				volume.Spec.PersistentVolumeSource.ISCSI.InitiatorName = &testInitiatorName
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"iscsi_initiator_name": "test-initiator-name"})
			})

			t.Run("Local, with fs type only if it is set", func(t *testing.T) {
				volume, transforms, value, tags := basicPVBuilderInput()
				volume.Spec.PersistentVolumeSource.Local = &v1.LocalVolumeSource{
					Path:   "test-local-path",
					FSType: nil,
				}
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"local_path": "test-local-path"})
				buildPVInfoAndAssertTagsNotPresent(volume, transforms, value, tags)(t, []string{"local_fs_type"})

				testLocalFSType := "test-local-fs-type"
				volume.Spec.PersistentVolumeSource.Local.FSType = &testLocalFSType
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"local_fs_type": "test-local-fs-type"})
			})

			t.Run("NFS", func(t *testing.T) {
				volume, transforms, value, tags := basicPVBuilderInput()
				volume.Spec.PersistentVolumeSource.NFS = &v1.NFSVolumeSource{
					Server: "test-nfs-server",
					Path:   "test-nfs-path",
				}
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"nfs_server": "test-nfs-server"})
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"nfs_path": "test-nfs-path"})
			})

			t.Run("RBD", func(t *testing.T) {
				volume, transforms, value, tags := basicPVBuilderInput()
				volume.Spec.PersistentVolumeSource.RBD = &v1.RBDPersistentVolumeSource{
					RBDImage: "test-rbd-image",
					FSType:   "test-rbd-fs-type",
				}
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"rbd_image": "test-rbd-image"})
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"rbd_fs_type": "test-rbd-fs-type"})
			})

			t.Run("PortworxVolume", func(t *testing.T) {
				volume, transforms, value, tags := basicPVBuilderInput()
				volume.Spec.PersistentVolumeSource.PortworxVolume = &v1.PortworxVolumeSource{
					VolumeID: "test-portworx-volume-id",
					FSType:   "test-portworx-fs-type",
				}
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"portworx_volume_id": "test-portworx-volume-id"})
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"portworx_fs_type": "test-portworx-fs-type"})
			})

			t.Run("FlexVolume", func(t *testing.T) {
				volume, transforms, value, tags := basicPVBuilderInput()
				volume.Spec.PersistentVolumeSource.FlexVolume = &v1.FlexPersistentVolumeSource{
					Driver: "test-flex-volume-driver",
					FSType: "test-flex-volume-fs-type",
				}
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"flex_volume_driver": "test-flex-volume-driver"})
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"flex_volume_fs_type": "test-flex-volume-fs-type"})
			})

			t.Run("GCEPersistentDisk", func(t *testing.T) {
				volume, transforms, value, tags := basicPVBuilderInput()
				volume.Spec.PersistentVolumeSource.GCEPersistentDisk = &v1.GCEPersistentDiskVolumeSource{
					PDName: "test-gce-pd-name",
				}
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"gce_persistent_disk_name": "test-gce-pd-name"})
			})

			t.Run("AWSElasticBlockStore", func(t *testing.T) {
				volume, transforms, value, tags := basicPVBuilderInput()
				volume.Spec.PersistentVolumeSource.AWSElasticBlockStore = &v1.AWSElasticBlockStoreVolumeSource{
					VolumeID: "test-aws-ebs-volume-id",
				}
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"ebs_volume_id": "test-aws-ebs-volume-id"})
			})

			t.Run("AzureDisk", func(t *testing.T) {
				volume, transforms, value, tags := basicPVBuilderInput()
				volume.Spec.PersistentVolumeSource.AzureDisk = &v1.AzureDiskVolumeSource{
					DiskName: "test-azure-disk-name",
				}
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"azure_disk_name": "test-azure-disk-name"})
			})

			t.Run("FC, with Lun only if it is set", func(t *testing.T) {
				volume, transforms, value, tags := basicPVBuilderInput()
				volume.Spec.PersistentVolumeSource.FC = &v1.FCVolumeSource{
					TargetWWNs: []string{
						"test-target-wwn-1",
						"test-target-wwn-2",
					},
					WWIDs: []string{
						"test-wwid-1",
						"test-wwid-2",
					},
					Lun: nil,
				}
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"fc_target_wwns": "test-target-wwn-1,test-target-wwn-2"})
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"fc_wwids": "test-wwid-1,test-wwid-2"})
				buildPVInfoAndAssertTagsNotPresent(volume, transforms, value, tags)(t, []string{"fc_lun"})

				testLun := int32(47)
				volume.Spec.PersistentVolumeSource.FC.Lun = &testLun
				buildPVInfoAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"fc_lun": "47"})
			})
		})

		t.Run("buildPVPhase has shared tags, type and phase by default", func(t *testing.T) {
			actualMetric := buildPVPhase(basicPVBuilderInput())
			expectedMetric := metricPoint(
				"kubernetes.",
				"pv.status.phase",
				5.0,
				0.0,
				"test-source-for-pv",
				map[string]string{
					"pv-tag1": "value1",
					"pv-tag2": "value2",
					"pv-tag3": "value3",
					"phase":   "Failed",
					"type":    "persistent_volume",
				},
			)
			assert.Equal(t, expectedMetric, actualMetric)
		})

		t.Run("buildPVPhase has claim ref only if set", func(t *testing.T) {
			volume, transforms, value, tags := basicPVBuilderInput()
			buildPVPhaseAndAssertTagsNotPresent(volume, transforms, value, tags)(t, []string{"claim_ref_name", "claim_ref_namespace"})

			volume.Spec.ClaimRef = &v1.ObjectReference{
				Name:      "test-claim-ref-name",
				Namespace: "test-claim-ref-namespace",
			}
			buildPVPhaseAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"claim_ref_name": "test-claim-ref-name"})
			buildPVPhaseAndAssertTags(volume, transforms, value, tags)(t, map[string]string{"claim_ref_namespace": "test-claim-ref-namespace"})
		})

		t.Run("buildPVAccessModes has a metric with access mode tag for each access mode", func(t *testing.T) {
			actualMetrics := buildPVAccessModes(basicPVBuilderInput())
			expectedRWOMetric := metricPoint(
				"kubernetes.",
				"pv.access_mode",
				1.0,
				0.0,
				"test-source-for-pv",
				map[string]string{
					"pv-tag1":     "value1",
					"pv-tag2":     "value2",
					"pv-tag3":     "value3",
					"access_mode": string(v1.ReadWriteOnce),
				},
			)

			expectedROMMetric := metricPoint(
				"kubernetes.",
				"pv.access_mode",
				1.0,
				0.0,
				"test-source-for-pv",
				map[string]string{
					"pv-tag1":     "value1",
					"pv-tag2":     "value2",
					"pv-tag3":     "value3",
					"access_mode": string(v1.ReadOnlyMany),
				},
			)

			assert.Contains(t, actualMetrics, expectedRWOMetric)
			assert.Contains(t, actualMetrics, expectedROMMetric)
		})
	})
}

func buildPVInfoAndAssertTags(volume *v1.PersistentVolume, transforms configuration.Transforms, timestamp int64, baseTags map[string]string) func(*testing.T, map[string]string) {
	actualMetric := buildPVInfo(volume, transforms, timestamp, baseTags)

	return func(t *testing.T, assertTags map[string]string) {
		for tagKey, tagVal := range assertTags {
			assert.Equal(t, tagVal, actualMetric.Tags()[tagKey])
		}
	}
}

func buildPVInfoAndAssertTagsNotPresent(volume *v1.PersistentVolume, transforms configuration.Transforms, timestamp int64, baseTags map[string]string) func(*testing.T, []string) {
	actualMetric := buildPVInfo(volume, transforms, timestamp, baseTags)

	return func(t *testing.T, assertTagKeys []string) {
		for _, tagKey := range assertTagKeys {
			_, found := actualMetric.Tags()[tagKey]
			assert.False(t, found)
		}
	}
}

func buildPVPhaseAndAssertTags(volume *v1.PersistentVolume, transforms configuration.Transforms, timestamp int64, baseTags map[string]string) func(*testing.T, map[string]string) {
	actualMetric := buildPVPhase(volume, transforms, timestamp, baseTags)

	return func(t *testing.T, assertTags map[string]string) {
		for tagKey, tagVal := range assertTags {
			assert.Equal(t, tagVal, actualMetric.Tags()[tagKey])
		}
	}
}

func buildPVPhaseAndAssertTagsNotPresent(volume *v1.PersistentVolume, transforms configuration.Transforms, timestamp int64, baseTags map[string]string) func(*testing.T, []string) {
	actualMetric := buildPVPhase(volume, transforms, timestamp, baseTags)

	return func(t *testing.T, assertTagKeys []string) {
		for _, tagKey := range assertTagKeys {
			_, found := actualMetric.Tags()[tagKey]
			assert.False(t, found)
		}
	}
}

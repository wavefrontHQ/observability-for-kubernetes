// Copyright 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package kstate

import (
	"reflect"
	"strconv"
	"time"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"

	corev1 "k8s.io/api/core/v1"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"

	log "github.com/sirupsen/logrus"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
)

func pointsForPV(item interface{}, transforms configuration.Transforms) []wf.Metric {
	persistentVolume, ok := item.(*corev1.PersistentVolume)
	if !ok {
		log.Errorf("invalid type: %s", reflect.TypeOf(item).String())
		return nil
	}

	sharedTags := buildTags("pv_name", persistentVolume.Name, persistentVolume.Namespace, transforms.Tags)
	copyLabels(persistentVolume.GetLabels(), sharedTags)

	now := time.Now().Unix()
	points := buildPVCapacityBytes(persistentVolume, transforms, now, sharedTags)
	points = append(points, buildPVInfo(persistentVolume, transforms, now, sharedTags))
	points = append(points, buildPVPhase(persistentVolume, transforms, now, sharedTags))
	points = append(points, buildPVAccessModes(persistentVolume, transforms, now, sharedTags)...)

	return points
}

func buildPVPhase(persistentVolume *corev1.PersistentVolume, transforms configuration.Transforms, now int64, sharedTags map[string]string) wf.Metric {
	tags := make(map[string]string, len(sharedTags))
	copyTags(sharedTags, tags)

	tags["phase"] = string(persistentVolume.Status.Phase)
	phaseValue := util.ConvertPVPhase(persistentVolume.Status.Phase)

	claimRef := persistentVolume.Spec.ClaimRef
	if claimRef != nil {
		tags["claim_ref_name"] = claimRef.Name
		tags["claim_ref_namespace"] = claimRef.Namespace
	}
	return metricPoint(transforms.Prefix, "pv.status.phase", float64(phaseValue), now, transforms.Source, tags)
}

func buildPVCapacityBytes(persistentVolume *corev1.PersistentVolume, transforms configuration.Transforms, now int64, sharedTags map[string]string) []wf.Metric {
	tags := make(map[string]string, len(sharedTags))
	copyTags(sharedTags, tags)

	var capacity = persistentVolume.Spec.Capacity[corev1.ResourceStorage]
	return []wf.Metric{
		metricPoint(transforms.Prefix, "pv.capacity_bytes", float64(capacity.Value()), now, transforms.Source, tags),
	}
}


func buildPVAccessModes(persistentVolume *corev1.PersistentVolume, transforms configuration.Transforms, now int64, sharedTags map[string]string) []wf.Metric {
	points := make([]wf.Metric, len(persistentVolume.Spec.AccessModes))
	for i, accessMode := range persistentVolume.Spec.AccessModes {
		tags := make(map[string]string, len(sharedTags))
		copyTags(sharedTags, tags)

		tags["access_mode"] = string(accessMode)

		points[i] = metricPoint(transforms.Prefix, "pv.access_mode",
			1.0, now, transforms.Source, tags)
	}
	return points
}

func buildPVInfo(persistentVolume *corev1.PersistentVolume, transforms configuration.Transforms, now int64, sharedTags map[string]string) wf.Metric {
	tags := make(map[string]string, len(sharedTags))
	copyTags(sharedTags, tags)

	setVolumeSource(persistentVolume, tags)
	return metricPoint(transforms.Prefix, "pv.info", 1.0, now, transforms.Source, tags)
}

func setVolumeSource(p *corev1.PersistentVolume, tags map[string]string) {
	switch {
	case p.Spec.PersistentVolumeSource.CephFS != nil:
		tags["cephfs_path"] = p.Spec.PersistentVolumeSource.CephFS.Path
	case p.Spec.PersistentVolumeSource.CSI != nil:
		tags["csi_driver"] = p.Spec.PersistentVolumeSource.CSI.Driver
		tags["csi_volume_handle"] = p.Spec.PersistentVolumeSource.CSI.VolumeHandle
	case p.Spec.PersistentVolumeSource.HostPath != nil:
		tags["host_path"] = p.Spec.PersistentVolumeSource.HostPath.Path
		if p.Spec.PersistentVolumeSource.HostPath.Type != nil {
			tags["host_path_type"] = string(*p.Spec.PersistentVolumeSource.HostPath.Type)
		}
	case p.Spec.PersistentVolumeSource.ISCSI != nil:
		tags["iscsi_target_portal"] = p.Spec.PersistentVolumeSource.ISCSI.TargetPortal
		tags["iscsi_iqn"] = p.Spec.PersistentVolumeSource.ISCSI.IQN
		tags["iscsi_lun"] = strconv.FormatInt(int64(p.Spec.PersistentVolumeSource.ISCSI.Lun), 10)
		if p.Spec.PersistentVolumeSource.ISCSI.InitiatorName != nil {
			tags["iscsi_initiator_name"] = *p.Spec.PersistentVolumeSource.ISCSI.InitiatorName
		}
	case p.Spec.PersistentVolumeSource.Local != nil:
		tags["local_path"] = p.Spec.PersistentVolumeSource.Local.Path
		if p.Spec.PersistentVolumeSource.Local.FSType != nil {
			tags["local_fs_type"] = *p.Spec.PersistentVolumeSource.Local.FSType
		}
	case p.Spec.PersistentVolumeSource.NFS != nil:
		tags["nfs_server"] = p.Spec.PersistentVolumeSource.NFS.Server
		tags["nfs_path"] = p.Spec.PersistentVolumeSource.NFS.Path
	case p.Spec.PersistentVolumeSource.RBD != nil:
		tags["rbd_image"] = p.Spec.PersistentVolumeSource.RBD.RBDImage
		tags["rbd_fs_type"] = p.Spec.PersistentVolumeSource.RBD.FSType
	//TODO: Remove the below deprecated(but still supported) volume sources in future.
	case p.Spec.PersistentVolumeSource.PortworxVolume != nil:
		tags["portworx_volume_id"] = p.Spec.PersistentVolumeSource.PortworxVolume.VolumeID
		tags["portworx_fs_type"] = p.Spec.PersistentVolumeSource.PortworxVolume.FSType
	case p.Spec.PersistentVolumeSource.FlexVolume != nil:
		tags["flex_volume_driver"] = p.Spec.PersistentVolumeSource.FlexVolume.Driver
		tags["flex_volume_fs_type"] = p.Spec.PersistentVolumeSource.FlexVolume.FSType
	case p.Spec.PersistentVolumeSource.GCEPersistentDisk != nil:
		tags["gce_persistent_disk_name"] = p.Spec.PersistentVolumeSource.GCEPersistentDisk.PDName
	case p.Spec.PersistentVolumeSource.AWSElasticBlockStore != nil:
		tags["ebs_volume_id"] = p.Spec.PersistentVolumeSource.AWSElasticBlockStore.VolumeID
	case p.Spec.PersistentVolumeSource.AzureDisk != nil:
		tags["azure_disk_name"] = p.Spec.PersistentVolumeSource.AzureDisk.DiskName
	case p.Spec.PersistentVolumeSource.FC != nil:
		if p.Spec.PersistentVolumeSource.FC.Lun != nil {
			tags["fc_lun"] = strconv.FormatInt(int64(*p.Spec.PersistentVolumeSource.FC.Lun), 10)
		}
		var (
			fcTargetWWNs, fcWWIDs string
		)
		for _, wwn := range p.Spec.PersistentVolumeSource.FC.TargetWWNs {
			if len(fcTargetWWNs) != 0 {
				fcTargetWWNs += ","
			}
			fcTargetWWNs += wwn
			tags["fc_target_wwns"] = fcTargetWWNs
		}
		for _, wwid := range p.Spec.PersistentVolumeSource.FC.WWIDs {
			if len(fcWWIDs) != 0 {
				fcWWIDs += ","
			}
			fcWWIDs += wwid
			tags["fc_wwids"] = fcWWIDs
		}
	}
}

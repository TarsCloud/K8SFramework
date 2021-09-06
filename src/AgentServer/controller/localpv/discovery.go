/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package localpv

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"hash/fnv"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"tarsagent/controller/common"
	"time"

	"k8s.io/api/core/v1"
)

// Discoverer finds available volumes and creates PVs for them
// It looks for volumes in the directories specified in the discoveryMap
type Discoverer struct {
	*common.RuntimeConfig
	Labels map[string]string
	// ProcTable is a reference to running processes so that we can prevent PV from being created while it is being cleaned
	CleanupTracker  *CleanupStatusTracker
	nodeAffinityAnn string
	nodeAffinity    *v1.VolumeNodeAffinity
	pvcLabelSelector labels.Selector
}

// NewDiscoverer creates a Discoverer object that will scan through
// the configured directories and create local PVs for any new directories found
func NewDiscoverer(config *common.RuntimeConfig, cleanupTracker *CleanupStatusTracker) (*Discoverer, error) {
	labelMap := make(map[string]string)
	for _, labelName := range config.NodeLabelsForPV {
		labelVal, ok := config.Node.Labels[labelName]
		if ok {
			labelMap[labelName] = labelVal
		}
	}

	volumeNodeAffinity, err := generateVolumeNodeAffinity(config.Node)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate volume node affinity: %v", err)
	}

	return &Discoverer{
		RuntimeConfig:  config,
		Labels:         labelMap,
		CleanupTracker: cleanupTracker,
		nodeAffinity:   volumeNodeAffinity,
		pvcLabelSelector: generatePVCLabelSelector(),
	}, nil
}

func (d *Discoverer) DiscoverVolumes() {
	pvcs, err := d.RuntimeConfig.K8sClient.CoreV1().PersistentVolumeClaims("").List(
		context.TODO(), metav1.ListOptions{LabelSelector: d.pvcLabelSelector.String()})
	if err != nil {
		glog.Errorf("Error list pvcs: %v", err)
		return
	}

	for _, pvc := range pvcs.Items {
		if pvc.Status.Phase == v1.ClaimPending {
			d.pendingPVC(pvc)
		} else if pvc.Status.Phase == v1.ClaimBound {
			d.boundPVC(pvc)
		}
	}
}

func (d *Discoverer) pendingPVC(pvc v1.PersistentVolumeClaim) bool {
	reclaimPolicy, mountOptions, err := d.getReclaimPolicyMountOptions(d.TStorageClass.Name)
	if err != nil {
		glog.Errorf("Failed to get ReclaimPolicy from storage class %q: %v", d.TStorageClass.Name, err)
		return false
	}
	if reclaimPolicy != v1.PersistentVolumeReclaimRetain && reclaimPolicy != v1.PersistentVolumeReclaimDelete {
		glog.Errorf("Unsupported ReclaimPolicy %q from storage class %q, supported policy are Retain and Delete.",
			reclaimPolicy, d.TStorageClass.Name)
		return false
	}

	app := pvc.Labels[common.TServerAppLabel]
	server := pvc.Labels[common.TServerNameLabel]
	directory := pvc.Labels[common.TLocalVolumeLabel]
	if app == "" || server == "" || directory == "" {
		glog.Errorf("pvc: %s has empty param. app: %s, server: %s, dir: %s", pvc.Name, app, server, directory)
		return false
	}

	// Ignore the label match if host-ipc or host-network etc.
	if directory != common.TLocalVolumeFakeName {
		if _, ok := d.RuntimeConfig.Node.Labels[common.TNodeSupportLabel]; !ok {
			return false
		}
	}

	// the rule of pvc name and local path
	prefix, postfix := getPVPrefixPosfix(pvc.Namespace, strings.ToLower(app), strings.ToLower(server), directory)

	pvName := generatePVName(prefix, d.Node.Name, d.TStorageClass.Name)
	pv, exists := d.Cache.GetPV(pvName)
	if exists {
		if pv.Spec.VolumeMode != nil && *pv.Spec.VolumeMode == v1.PersistentVolumeBlock {
			errStr := fmt.Sprintf("UnSupported Volume Mode: %s.%s requires block mode.\n",
				pvc.Namespace, pvName)
			glog.Errorf(errStr)
			d.Recorder.Eventf(pv, v1.EventTypeWarning, common.EventVolumeFailedDelete, errStr)
		}
		return false
	}

	if d.CleanupTracker.InProgress(pvName) {
		glog.Infof("PV %s is still being cleaned, not going to recreate it", pvName)
		return false
	}

	// Create directory in agent container
	insidePath := filepath.Join(d.TStorageClass.MountDir, postfix)
	if err := d.changeHostDir(insidePath, pvc.Annotations, v1.ClaimPending); err != nil {
		glog.Error(fmt.Sprintf("pvc %s.%s chown %s, err: %s.\n", pvc.Namespace, pvc.Name, insidePath, err))
		return false
	}

	// Create PV resource
	localPVConfig := &common.LocalPVConfig{
		Name:            pvName,
		HostPath:        filepath.Join(d.TStorageClass.HostDir, postfix),
		Capacity:        roundDownCapacityPretty(5 * common.GiB),
		StorageClass:    d.TStorageClass.Name,
		ReclaimPolicy:   reclaimPolicy,
		ProvisionerName: d.Name,
		VolumeMode:      v1.PersistentVolumeFilesystem,
		Labels:          d.Labels,
		MountOptions:    mountOptions,
		NodeAffinity: 	 d.nodeAffinity,
	}
	// PV labels must matches to pvc
	localPVConfig.Labels[common.TServerAppLabel] = app
	localPVConfig.Labels[common.TServerNameLabel] = server
	localPVConfig.Labels[common.TLocalVolumeLabel] = directory

	_, err = d.APIUtil.CreatePV(common.CreateLocalPVSpec(localPVConfig))
	if err != nil {
		glog.Errorf("Error creating PV %q for volume at %q: %v", pvName, insidePath, err)
		return false
	}

	// Hack for waiting populator's cache
	time.Sleep(1 * time.Second)
	glog.Infof("Created PV %q for volume at %q", pvName, insidePath)
	return true
}

func (d *Discoverer) boundPVC(pvc v1.PersistentVolumeClaim) bool {
	if pv, exists := d.Cache.GetPV(pvc.Spec.VolumeName); exists {
		if err := d.changeHostDir(pv.Spec.Local.Path, pvc.Annotations, v1.ClaimBound); err != nil {
			glog.Error(fmt.Sprintf("pvc %s.%s chown %s, err: %s.\n", pvc.Namespace, pvc.Name, pv.Spec.Local.Path, err))
			return false
		}
	}
	return true
}

func (d *Discoverer) changeHostDir(dirPath string, annotations map[string]string, phase v1.PersistentVolumeClaimPhase) error {
	if !d.VolUtil.Existed(dirPath) {
		if phase == v1.ClaimBound {
			return fmt.Errorf("phase: %s has no dirPath: %s.\n", phase, dirPath)
		} else {
			if err := d.VolUtil.MakeDir(dirPath); err != nil {
				return err
			}
			glog.Infof("phase: %s create dirPath: %s", dirPath, phase)
		}
	}

	if permAnn, ok := annotations[common.TLocalVolumePermAnn]; ok && permAnn != "" {
		 perm, err := strconv.ParseInt(permAnn, 8, 32)
		 if err != nil {
		 	return err
		 }
		 newPerm := os.FileMode(uint32(perm))

		 oldPerm, err := d.VolUtil.GetFilePerm(dirPath)
		 if err != nil {
		 	return err
		 }

		 if oldPerm != newPerm {
			 glog.Infof("change dirPath: %s mode, from: %o to %o", dirPath, oldPerm, newPerm)
			 if err := os.Chmod(dirPath, newPerm); err != nil {
				 return err
			 }
		 }
	}

	uidAnn, ok1 := annotations[common.TLocalVolumeUIDAnn]
	gidAnn, ok2 := annotations[common.TLocalVolumeGIDAnn]
	if ok1 && ok2 && uidAnn != "" && gidAnn != "" {
		newUid, err := strconv.Atoi(uidAnn)
		if err != nil {
			return err
		}
		newGid, err := strconv.Atoi(gidAnn)
		if err != nil {
			return err
		}

		oldUid, oldGid, err := d.VolUtil.GetFileUidGid(dirPath)
		if err != nil {
			return err
		}

		if oldUid != newUid || oldGid != newGid {
			glog.Infof("change dirPath: %s owner, from: %d.%d to %d.%d", dirPath, oldUid, oldGid, newUid, newGid)
			if err := os.Chown(dirPath, newUid, newGid); err != nil {
				return err
			}
		}
	}

	return nil
}

func (d *Discoverer) getReclaimPolicyMountOptions(name string) (v1.PersistentVolumeReclaimPolicy, []string, error) {
	class, err := d.K8sClient.StorageV1().StorageClasses().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return "", nil, err
	}
	if class.ReclaimPolicy != nil {
		return *class.ReclaimPolicy, class.MountOptions, nil
	}
	return v1.PersistentVolumeReclaimDelete, nil, nil
}

func generatePVName(prefix, node, class string) string {
	h := fnv.New32a()
	h.Write([]byte(node))
	h.Write([]byte(class))
	// This is the FNV-1a 32-bit hash
	return fmt.Sprintf("%s-%x", prefix, h.Sum32())
}

func getPVPrefixPosfix(namespace, app, server, dir string) (string, string) {
	prefix := fmt.Sprintf("%s-%s-%s-%s", namespace, dir, app, server)
	posfix := fmt.Sprintf("%s/%s.%s/%s", namespace, app, server, dir)
	return prefix, posfix
}

func generateVolumeNodeAffinity(node *v1.Node) (*v1.VolumeNodeAffinity, error) {
	if node.Labels == nil {
		return nil, fmt.Errorf("Node does not have labels")
	}
	nodeValue, found := node.Labels[common.NodeLabelKey]
	if !found {
		return nil, fmt.Errorf("Node does not have expected label %s", common.NodeLabelKey)
	}

	return &v1.VolumeNodeAffinity{
		Required: &v1.NodeSelector{
			NodeSelectorTerms: []v1.NodeSelectorTerm{
				{
					MatchExpressions: []v1.NodeSelectorRequirement{
						{
							Key:      common.NodeLabelKey,
							Operator: v1.NodeSelectorOpIn,
							Values:   []string{nodeValue},
						},
					},
				},
			},
		},
	}, nil
}

func generatePVCLabelSelector() labels.Selector {
	tServerAppLabel		, _  := labels.NewRequirement(common.TServerAppLabel, selection.Exists, nil)
	tServerNameLabel	, _  := labels.NewRequirement(common.TServerNameLabel, selection.Exists, nil)
	tLocalVolumeLabel	, _  := labels.NewRequirement(common.TLocalVolumeLabel, selection.Exists, nil)

	tPVCLabelSelector := labels.NewSelector().Add(*tLocalVolumeLabel, *tServerAppLabel, *tServerNameLabel)
	return tPVCLabelSelector
}

// Round down the capacity to an easy to read value.
func roundDownCapacityPretty(capacityBytes int64) int64 {
	easyToReadUnitsBytes := []int64{common.GiB, common.MiB}

	// Round down to the nearest easy to read unit
	// such that there are at least 10 units at that size.
	for _, easyToReadUnitBytes := range easyToReadUnitsBytes {
		// Round down the capacity to the nearest unit.
		size := capacityBytes / easyToReadUnitBytes
		if size >= 10 {
			return size * easyToReadUnitBytes
		}
	}
	return capacityBytes
}

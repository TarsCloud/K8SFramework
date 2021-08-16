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
	"strings"
	"syscall"
	"tarsagent/controller/common"

	"k8s.io/api/core/v1"
	storagev1listers "k8s.io/client-go/listers/storage/v1"
	"k8s.io/client-go/tools/cache"
)

// Discoverer finds available volumes and creates PVs for them
// It looks for volumes in the directories specified in the discoveryMap
type Discoverer struct {
	*common.RuntimeConfig
	Labels map[string]string
	// ProcTable is a reference to running processes so that we can prevent PV from being created while
	// it is being cleaned
	CleanupTracker  *CleanupStatusTracker
	nodeAffinityAnn string
	nodeAffinity    *v1.VolumeNodeAffinity
	classLister     storagev1listers.StorageClassLister
}

// NewDiscoverer creates a Discoverer object that will scan through
// the configured directories and create local PVs for any new directories found
func NewDiscoverer(config *common.RuntimeConfig, cleanupTracker *CleanupStatusTracker) (*Discoverer, error) {
	sharedInformer := config.K8sInformerFactory.Storage().V1().StorageClasses()
	sharedInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		// We don't need an actual event handler for StorageClasses,
		// but we must pass a non-nil one to cache.NewInformer()
		AddFunc:    nil,
		UpdateFunc: nil,
		DeleteFunc: nil,
	})

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
		classLister:    sharedInformer.Lister(),
		nodeAffinity:   volumeNodeAffinity}, nil
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

// DiscoverLocalVolumes reads the configured discovery paths, and creates PVs for the new volumes
func (d *Discoverer) DiscoverLocalVolumes() {
	if _, ok := d.RuntimeConfig.Node.Labels[common.TNodeSupportLabel]; !ok {
		return
	}
	for class, config := range d.DiscoveryMap {
		d.discoverVolumesAtPath(class, config)
	}
}

func (d *Discoverer) getReclaimPolicyFromStorageClass(name string) (v1.PersistentVolumeReclaimPolicy, error) {
	class, err := d.classLister.Get(name)
	if err != nil {
		return "", err
	}
	if class.ReclaimPolicy != nil {
		return *class.ReclaimPolicy, nil
	}
	return v1.PersistentVolumeReclaimDelete, nil
}

func (d *Discoverer) getMountOptionsFromStorageClass(name string) ([]string, error) {
	class, err := d.classLister.Get(name)
	if err != nil {
		return nil, err
	}
	return class.MountOptions, nil
}

func (d *Discoverer) discoverVolumesAtPath(class string, config common.MountConfig) {
	glog.V(7).Infof("Discovering volumes at hostpath %q, mount path %q for storage class %q", config.HostDir, config.MountDir, class)

	reclaimPolicy, err := d.getReclaimPolicyFromStorageClass(class)
	if err != nil {
		glog.Errorf("Failed to get ReclaimPolicy from storage class %q: %v", class, err)
		return
	}

	if reclaimPolicy != v1.PersistentVolumeReclaimRetain && reclaimPolicy != v1.PersistentVolumeReclaimDelete {
		glog.Errorf("Unsupported ReclaimPolicy %q from storage class %q, supported policy are Retain and Delete.", reclaimPolicy, class)
		return
	}

	tServerAppLabel		, _  := labels.NewRequirement(common.TServerAppLabel, selection.Exists, nil)
	tServerNameLabel	, _  := labels.NewRequirement(common.TServerNameLabel, selection.Exists, nil)
	tLocalVolumeLabel	, _  := labels.NewRequirement(common.TLocalVolumeLabel, selection.Exists, nil)
	tPVCLabelSelector := labels.NewSelector().Add(*tLocalVolumeLabel, *tServerAppLabel, *tServerNameLabel)
	tPVCs, err := d.RuntimeConfig.K8sClient.CoreV1().PersistentVolumeClaims("").List(
		context.TODO(), metav1.ListOptions{LabelSelector: tPVCLabelSelector.String()})
	if err != nil {
		glog.Errorf("Error list pvcs: %v", err)
		return
	}

	// Put pv name into set for filter pod seq
	unitMap := make(map[string]bool)

	for _, pvc := range tPVCs.Items {
		if pvc.Status.Phase != v1.ClaimPending {
			continue
		}

		app := pvc.Labels[common.TServerAppLabel]
		server := pvc.Labels[common.TServerNameLabel]
		dirName := pvc.Labels[common.TLocalVolumeLabel]
		if app == "" || server == "" || dirName == "" {
			continue
		}

		// Check if PV already exists for it
		prefix, posfix := getPVPrefixPosfix(pvc.Namespace, strings.ToLower(app), strings.ToLower(server), dirName)
		if _, ok := unitMap[prefix]; ok {
			continue
		}

		pvName := generatePVName(prefix, d.Node.Name, class)
		pv, exists := d.Cache.GetPV(pvName)
		if exists {
			if pv.Spec.VolumeMode != nil && *pv.Spec.VolumeMode == v1.PersistentVolumeBlock {
				errStr := fmt.Sprintf("UnSupported Volume Mode: %s.%s requires block mode.\n",
					pvc.Namespace, pvName)
				glog.Errorf(errStr)
				d.Recorder.Eventf(pv, v1.EventTypeWarning, common.EventVolumeFailedDelete, errStr)
			}
			unitMap[prefix] = true
			continue
		}
		if d.CleanupTracker.InProgress(pvName) {
			glog.Infof("PV %s is still being cleaned, not going to recreate it", pvName)
			continue
		}

		// Create directory
		dirPath := filepath.Join(config.MountDir, posfix)
		if !d.VolUtil.Existed(dirPath) {
			// unix permissions are 'filtered' by whatever umask has been set.
			oldMask := syscall.Umask(0)
			if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
				glog.Error(fmt.Sprintf("pvc %s.%s mkdir %s, err: %s.\n", pvc.Namespace, pvc.Name, dirPath, err))
				continue
			}
			syscall.Umask(oldMask)
		}

		volMode, err := common.GetVolumeMode(d.VolUtil, dirPath)
		if err != nil {
			glog.Error(err)
			continue
		}

		mountOptions, err := d.getMountOptionsFromStorageClass(class)
		if err != nil {
			glog.Errorf("Failed to get mount options from storage class %s: %v", class, err)
			continue
		}

		var capacityByte int64
		desireVolumeMode := v1.PersistentVolumeMode(config.VolumeMode)
		switch volMode {
		case v1.PersistentVolumeFilesystem:
			if desireVolumeMode == v1.PersistentVolumeBlock {
				glog.Errorf("Path %q of filesystem mode cannot be used to create block volume", dirPath)
				continue
			}
			capacityByte, err = d.VolUtil.GetFsCapacityByte(dirPath)
			if err != nil {
				glog.Errorf("Path %q fs stats error: %v", dirPath, err)
				continue
			}
		default:
			glog.Errorf("Path %q has unexpected volume type %q", dirPath, volMode)
			continue
		}

		if err := d.createPV(class, pvc, reclaimPolicy, mountOptions, config, capacityByte, desireVolumeMode); err == nil {
			unitMap[prefix] = true
		}
	}
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

func (d *Discoverer) createPV(class string, pvc v1.PersistentVolumeClaim, reclaimPolicy v1.PersistentVolumeReclaimPolicy,
	mountOptions []string,	config common.MountConfig, capacityByte int64, volMode v1.PersistentVolumeMode) error {
	// Basic Info
	app := pvc.Labels[common.TServerAppLabel]
	server := pvc.Labels[common.TServerNameLabel]
	dirName := pvc.Labels[common.TLocalVolumeLabel]

	prefix, posfix := getPVPrefixPosfix(pvc.Namespace, strings.ToLower(app), strings.ToLower(server), dirName)
	pvName := generatePVName(prefix, d.Node.Name, class)
	outsidePath := filepath.Join(config.HostDir, posfix)
	glog.Infof("Found new volume at host path %q with capacity %d, creating Local PV %q, required volumeMode %q",
		outsidePath, capacityByte, pvName, volMode)

	// Init PV resource
	localPVConfig := &common.LocalPVConfig{
		Name:            pvName,
		HostPath:        outsidePath,
		Capacity:        roundDownCapacityPretty(capacityByte),
		StorageClass:    class,
		ReclaimPolicy:   reclaimPolicy,
		ProvisionerName: d.Name,
		VolumeMode:      volMode,
		Labels:          d.Labels,
		MountOptions:    mountOptions,
		NodeAffinity: 	 d.nodeAffinity,
	}
	// pv labels must matches to pvc
	localPVConfig.Labels[common.TServerAppLabel] = app
	localPVConfig.Labels[common.TServerNameLabel] = server
	localPVConfig.Labels[common.TLocalVolumeLabel] = dirName

	// Create PV resource
	_, err := d.APIUtil.CreatePV(common.CreateLocalPVSpec(localPVConfig))
	if err != nil {
		glog.Errorf("Error creating PV %q for volume at %q: %v", pvName, outsidePath, err)
		return err
	}
	glog.Infof("Created PV %q for volume at %q", pvName, outsidePath)

	return nil
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

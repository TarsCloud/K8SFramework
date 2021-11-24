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

package common

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/mount"
	crdClientSet "k8s.tars.io/client-go/clientset/versioned"
	crdInformers "k8s.tars.io/client-go/informers/externalversions"
	"path/filepath"
)

const (
	KiB int64 = 1024
	MiB int64 = 1024 * KiB
	GiB int64 = 1024 * MiB
	TiB int64 = 1024 * GiB
)

const (
	// TImageBaseType is language-based image
	TImageBaseType 	= "base"
	// TImageNodeType is tarsnode image
	TImageNodeType 	= "node"
	// TImageServerType is business server image
	TImageServerType= "server"

	// TServerAppLabel is App label
	TServerAppLabel  = "tars.io/ServerApp"
	// TServerNameLabel is Server label
	TServerNameLabel = "tars.io/ServerName"
	// TLocalVolumeLabel is LocalPV label
	TLocalVolumeLabel = "tars.io/LocalVolume"

	// NodeServerAppAffinityPrefix is node ability label prefix
	NodeServerAppAffinityPrefix = "tars.io/ability"
	// NodeNamespaceAffinityPrefix is tars node label prefix
	NodeNamespaceAffinityPrefix = "tars.io/node"

	// TNodeSupportLabel is the label which represents the support for local pv
	TNodeSupportLabel = "tars.io/SupportLocalVolume"
	// TStorageClassName is the tars storage class of local pv
	TStorageClassName = "tars-storage-class"
	// TLocalVolumeHostDir is the tars mount dir of local pv
	TLocalVolumeHostDir = "/usr/local/app/tars/host-mount"
	// TLocalVolumeMode is the tars volume mode of local pv
	TLocalVolumeMode = "Filesystem"
	// TLocalVolumeFakeName is specific pvc name for hostipc etc.
	TLocalVolumeFakeName = "delay-bind"

	// TLocalVolumeUIDAnn is pvc uid
	TLocalVolumeUIDAnn  = "tars.io/LocalVolumeUID"
	// TLocalVolumeGIDAnn is pvc gid
	TLocalVolumeGIDAnn = "tars.io/LocalVolumeGID"
	// TLocalVolumePermAnn is pvc perm
	TLocalVolumePermAnn = "tars.io/LocalVolumeMode"
	// AnnProvisionedBy is the external provisioner annotation in PV object
	AnnProvisionedBy = "pv.kubernetes.io/provisioned-by"
	// EventVolumeFailedDelete copied from k8s.io/kubernetes/pkg/controller/volume/events
	EventVolumeFailedDelete = "VolumeFailedDelete"

	// NodeLabelKey is the label key that this provisioner uses for PV node affinity
	// hostname is not the best choice, but it's what pod and node affinity also use
	NodeLabelKey = "kubernetes.io/hostname"
)

// UserConfig stores all the user-defined parameters to the provisioner
type UserConfig struct {
	// Node object for this node
	Node *v1.Node
	// key = storageclass, value = mount configuration for the storageclass
	TStorageClass MountConfig
	// Labels and their values that are added to PVs created by the provisioner
	NodeLabelsForPV []string
	// Namespace of this Pod (optional)
	Namespace string
	// MinResyncPeriod is minimum resync period. Resync period in reflectors
	// will be random between MinResyncPeriod and 2*MinResyncPeriod.
	MinResyncPeriod metav1.Duration
}

// MountConfig stores a configuration for discoverying a specific storageclass
type MountConfig struct {
	// the storageclass name
	Name string `json:"name" yaml:"name"`
	// The hostpath directory
	HostDir string `json:"hostDir" yaml:"hostDir"`
	// The mount point of the hostpath volume
	MountDir string `json:"mountDir" yaml:"mountDir"`
	// The volume mode of created PersistentVolume object,
	// default to Filesystem if not specified.
	VolumeMode string `json:"volumeMode" yaml:"volumeMode"`
}

// RuntimeConfig stores all the objects that the provisioner needs to run
type RuntimeConfig struct {
	*UserConfig
	// Unique name of this provisioner
	Name string
	// Cache to store PVs managed by this provisioner
	Cache *VolumeCache
	// K8s API layer
	APIUtil APIUtil
	// Volume util layer
	VolUtil VolumeUtil
	// K8s API client
	K8sClient kubernetes.Interface
	CrdClient crdClientSet.Interface
	// Recorder is used to record events in the API server
	Recorder record.EventRecorder
	// Disable block device discovery and management if true
	BlockDisabled bool
	// Mounter used to verify mountpoints
	Mounter mount.Interface
	// InformerFactory gives access to informers for the controller.
	K8sInformerFactory informers.SharedInformerFactory
	CrdInformerFactory crdInformers.SharedInformerFactory
}

// LocalPVConfig defines the parameters for creating a local PV
type LocalPVConfig struct {
	Name            string
	HostPath        string
	Capacity        int64
	StorageClass    string
	ReclaimPolicy   v1.PersistentVolumeReclaimPolicy
	ProvisionerName string
	NodeAffinity    *v1.VolumeNodeAffinity
	VolumeMode      v1.PersistentVolumeMode
	MountOptions    []string
	Labels          map[string]string
}

// CreateLocalPVSpec returns a PV spec that can be used for PV creation
func CreateLocalPVSpec(config *LocalPVConfig) *v1.PersistentVolume {
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:   config.Name,
			Labels: config.Labels,
			Annotations: map[string]string{
				AnnProvisionedBy: config.ProvisionerName,
			},
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: config.ReclaimPolicy,
			Capacity: v1.ResourceList{
				v1.ResourceName(v1.ResourceStorage): *resource.NewQuantity(int64(config.Capacity), resource.BinarySI),
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				Local: &v1.LocalVolumeSource{
					Path:   config.HostPath,
				},
			},
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			StorageClassName: config.StorageClass,
			VolumeMode:       &config.VolumeMode,
			MountOptions:     config.MountOptions,
			NodeAffinity: 	  config.NodeAffinity,
		},
	}
	return pv
}

// GetContainerPath gets the local path (within provisioner container) of the PV
func GetContainerPath(pv *v1.PersistentVolume, config MountConfig) (string, error) {
	relativePath, err := filepath.Rel(config.HostDir, pv.Spec.Local.Path)
	if err != nil {
		return "", err
	}

	return filepath.Join(config.MountDir, relativePath), nil
}

// GetVolumeMode check volume mode of given path.
func GetVolumeMode(volUtil VolumeUtil, fullPath string) (v1.PersistentVolumeMode, error) {
	isdir, err := volUtil.IsDir(fullPath)
	if isdir {
		return v1.PersistentVolumeFilesystem, nil
	}
	return "", err
}

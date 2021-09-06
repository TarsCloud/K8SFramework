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
	"fmt"
	"os"
	"strings"
	"syscall"
	"tarsagent/controller/common"
	"time"

	"github.com/golang/glog"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

// CleanupState indicates the state of the cleanup process.
type CleanupState int

const (
	// CSUnknown State of the cleanup is unknown.
	CSUnknown CleanupState = iota + 1
	// CSNotFound No cleanup process was found.
	CSNotFound
	// CSRunning Cleanup process is still running.
	CSRunning
	// CSFailed Cleanup process has ended in failure.
	CSFailed
	// CSSucceeded Cleanup process has ended successfully.
	CSSucceeded
)

// Deleter handles PV cleanup and object deletion
// For file-based volumes, it deletes the contents of the directory
type Deleter struct {
	*common.RuntimeConfig
	CleanupStatus *CleanupStatusTracker
}

// NewDeleter creates a Deleter object to handle the cleanup and deletion of local PVs
// allocated by this provisioner
func NewDeleter(config *common.RuntimeConfig, cleanupTracker *CleanupStatusTracker) *Deleter {
	return &Deleter{
		RuntimeConfig: config,
		CleanupStatus: cleanupTracker,
	}
}

// DeletePVs will scan through all the existing PVs that are released, and cleanup and
// delete them
func (d *Deleter) DeletePVs() {
	nowTime := time.Now()
	for _, pv := range d.Cache.ListPVs() {
		if pv.Status.Phase != v1.VolumeReleased && pv.Status.Phase != v1.VolumeAvailable {
			continue
		}
		name := pv.Name
		switch pv.Spec.PersistentVolumeReclaimPolicy {
		case v1.PersistentVolumeReclaimRetain:
			glog.Infof("reclaimVolume[%s]: policy is Retain, nothing to do", name)
		case v1.PersistentVolumeReclaimRecycle:
			glog.Infof("reclaimVolume[%s]: policy is Recycle which is not supported", name)
			d.RuntimeConfig.Recorder.Eventf(pv, v1.EventTypeWarning, "VolumeUnsupportedReclaimPolicy", "Volume has unsupported PersistentVolumeReclaimPolicy: Recycle")
		case v1.PersistentVolumeReclaimDelete:
			glog.Infof("reclaimVolume[%s]: policy is Delete", name)
			var err error
			if pv.Status.Phase == v1.VolumeReleased {
				err = d.deletePV(pv)
			} else {
				if pv.CreationTimestamp.Add(24 * time.Hour).Before(nowTime) {
					err = d.deletePV(pv)
				}
			}
			if err != nil {
				cleaningLocalPVErr := fmt.Errorf("Error cleaning PV %q: %v", name, err.Error())
				d.RuntimeConfig.Recorder.Eventf(pv, v1.EventTypeWarning, common.EventVolumeFailedDelete, cleaningLocalPVErr.Error())
				glog.Error(err)
			}
		default:
			// Unknown PersistentVolumeReclaimPolicy
			d.RuntimeConfig.Recorder.Eventf(pv, v1.EventTypeWarning, "VolumeUnknownReclaimPolicy", "Volume has unrecognized PersistentVolumeReclaimPolicy")
		}
	}
}

func (d *Deleter) getVolPathMode(pv *v1.PersistentVolume) (string, v1.PersistentVolumeMode, error) {
	mountPath, err := common.GetContainerPath(pv, d.TStorageClass)
	if err != nil {
		return "", "", err
	}

	volMode, err := common.GetVolumeMode(d.VolUtil, mountPath)
	if err != nil {
		return "", "", err
	}

	return mountPath, volMode, nil
}

func (d *Deleter) deletePV(pv *v1.PersistentVolume) error {
	if pv.Spec.Local == nil {
		return fmt.Errorf("Unsupported volume type")
	}

	// Exit if cleaning is still in progress.
	if d.CleanupStatus.InProgress(pv.Name) {
		return nil
	}

	// Check if cleaning was just completed.
	state, _, err := d.CleanupStatus.RemoveStatus(pv.Name)
	if err != nil {
		return err
	}

	switch state {
	case CSSucceeded:
		// Found a completed cleaning entry
		glog.Infof("Deleting pv %s after successful cleanup", pv.Name)
		if err = d.APIUtil.DeletePV(pv.Name); err != nil {
			if !errors.IsNotFound(err) {
				d.RuntimeConfig.Recorder.Eventf(pv, v1.EventTypeWarning, common.EventVolumeFailedDelete,
					err.Error())
				return fmt.Errorf("Error deleting PV %q: %v", pv.Name, err.Error())
			}
		}
		return nil
	case CSFailed:
		glog.Infof("Cleanup for pv %s failed. Restarting cleanup", pv.Name)
	case CSNotFound:
		glog.Infof("Start cleanup for pv %s", pv.Name)
	default:
		return fmt.Errorf("Unexpected state %d for pv %s", state, pv.Name)
	}

	return d.runProcess(pv, d.TStorageClass)
}

func (d *Deleter) runProcess(pv *v1.PersistentVolume, config common.MountConfig) error {
	// Run as exec script.
	err := d.CleanupStatus.ProcTable.MarkRunning(pv.Name)
	if err != nil {
		return err
	}

	mountPath, volMode, err := d.getVolPathMode(pv)
	if err != nil {
		pathErr, ok := err.(*os.PathError)
		if ok && pathErr.Err == syscall.ENOENT {
			glog.Errorf("failed to get volume mode of path %q: %v, delete pv directly.", mountPath, err)
			// Set process as succeeded.
			if err := d.CleanupStatus.ProcTable.MarkSucceeded(pv.Name); err != nil {
				glog.Error(err)
			}
			return nil
		} else {
			return fmt.Errorf("failed to get volume mode of path %q: %v", mountPath, err)
		}
	}

	go d.asyncCleanPV(pv, volMode, mountPath, config)
	return nil
}

func (d *Deleter) asyncCleanPV(pv *v1.PersistentVolume, volMode v1.PersistentVolumeMode, mountPath string,
	config common.MountConfig) {

	err := d.cleanPV(pv, volMode, mountPath, config)
	if err != nil {
		glog.Error(err)
		// Set process as failed.
		if err := d.CleanupStatus.ProcTable.MarkFailed(pv.Name); err != nil {
			glog.Error(err)
		}
		return
	}
	// Set process as succeeded.
	if err := d.CleanupStatus.ProcTable.MarkSucceeded(pv.Name); err != nil {
		glog.Error(err)
	}
}

func (d *Deleter) cleanPV(pv *v1.PersistentVolume, volMode v1.PersistentVolumeMode, mountPath string,
	config common.MountConfig) error {
	// Make absolutely sure here that we are not deleting anything outside of mounted dir
	if !strings.HasPrefix(mountPath, config.MountDir) {
		return fmt.Errorf("Unexpected error pv %q mountPath %s but mount dir is %s", pv.Name, mountPath,
			config.MountDir)
	}

	var err error
	switch volMode {
	case v1.PersistentVolumeFilesystem:
		err = d.cleanFilePV(pv, mountPath)
	default:
		err = fmt.Errorf("Unexpected volume mode %q for deleting path %q", volMode, pv.Spec.Local.Path)
	}
	return err
}

func (d *Deleter) cleanFilePV(pv *v1.PersistentVolume, mountPath string) error {
	glog.Infof("Deleting PV file volume %q contents at hostpath %q, mountpath %q",
		pv.Name, pv.Spec.Local.Path, mountPath)
	if pv.Status.Phase == v1.VolumeAvailable {
		// A little of bit non-sense because of the server dir remained
		return d.VolUtil.DeleteEmptyDir(mountPath)
	} else if pv.Status.Phase == v1.VolumeReleased {
		if err := d.VolUtil.DeleteContents(mountPath); err != nil {
			return err
		}
		// A little of bit non-sense because of the server dir remained
		return d.VolUtil.DeleteEmptyDir(mountPath)
		return nil
	} else {
		return fmt.Errorf("Deleting PV file volume %q has unexpected phase %s\n", pv.Name, pv.Status.Phase)
	}
}

// CleanupStatusTracker tracks cleanup processes that are either process based or jobs based.
type CleanupStatusTracker struct {
	ProcTable	ProcTable
}

// InProgress returns true if the cleaning for the specified PV is in progress.
func (c *CleanupStatusTracker) InProgress(pvName string) bool {
	return c.ProcTable.IsRunning(pvName)
}

// RemoveStatus removes and returns the status and start time of a completed cleaning process.
// The method returns an error if the process has not yet completed.
func (c *CleanupStatusTracker) RemoveStatus(pvName string) (CleanupState, *time.Time, error) {
	return c.ProcTable.RemoveEntry(pvName)
}

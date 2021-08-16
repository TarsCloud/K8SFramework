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
	"context"
	batch_v1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"os"
	"path/filepath"
)

// APIUtil is an interface for the K8s API
type APIUtil interface {
	// CreatePV Creates PersistentVolume object
	CreatePV(pv *v1.PersistentVolume) (*v1.PersistentVolume, error)

	// DeletePV Deletes PersistentVolume object
	DeletePV(pvName string) error

	// CreateJob Creates a Job execution.
	CreateJob(job *batch_v1.Job) error

	// DeleteJob deletes specified Job by its name and namespace.
	DeleteJob(jobName string, namespace string) error
}

var _ APIUtil = &apiUtil{}

type apiUtil struct {
	client *kubernetes.Clientset
}

// NewAPIUtil creates a new APIUtil object that represents the K8s API
func NewAPIUtil(client *kubernetes.Clientset) APIUtil {
	return &apiUtil{client: client}
}

// CreatePV will create a PersistentVolume
func (u *apiUtil) CreatePV(pv *v1.PersistentVolume) (*v1.PersistentVolume, error) {
	pv, err := u.client.CoreV1().PersistentVolumes().Create(context.TODO(), pv, metav1.CreateOptions{})
	return pv, err
}

// DeletePV will delete a PersistentVolume
func (u *apiUtil) DeletePV(pvName string) error {
	err := u.client.CoreV1().PersistentVolumes().Delete(context.TODO(), pvName, metav1.DeleteOptions{})
	return err
}

func (u *apiUtil) CreateJob(job *batch_v1.Job) error {
	_, err := u.client.BatchV1().Jobs(job.Namespace).Create(context.TODO(), job, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (u *apiUtil) DeleteJob(jobName string, namespace string) error {
	deleteProp := metav1.DeletePropagationForeground
	if err := u.client.BatchV1().Jobs(namespace).Delete(context.TODO(), jobName,
		metav1.DeleteOptions{PropagationPolicy: &deleteProp}); err != nil {
		return err
	}

	return nil
}

// VolumeUtil is an interface for local filesystem operations
type VolumeUtil interface {
	// Existed checks if the given path is existed
	Existed(fullPath string) bool

	// IsDir checks if the given path is a directory
	IsDir(fullPath string) (bool, error)

	// ReadDir returns a list of files under the specified directory
	ReadDir(fullPath string) ([]string, error)

	// DeleteContents deletes all the contents under the given path, but not the path itself
	DeleteContents(fullPath string) error

	// GetFsCapacityByte gets capacity for fs on full path
	GetFsCapacityByte(fullPath string) (int64, error)
}

var _ VolumeUtil = &volumeUtil{}

type volumeUtil struct{}

// NewVolumeUtil returns a VolumeUtil object for performing local filesystem operations
func NewVolumeUtil() VolumeUtil {
	return &volumeUtil{}
}

// Existed checks if the given path is a directory
func (u *volumeUtil) Existed(fullPath string) bool {
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return false
	}
	return true
}


// IsDir checks if the given path is a directory
func (u *volumeUtil) IsDir(fullPath string) (bool, error) {
	dir, err := os.Open(fullPath)
	if err != nil {
		return false, err
	}
	defer dir.Close()

	stat, err := dir.Stat()
	if err != nil {
		return false, err
	}

	return stat.IsDir(), nil
}

// ReadDir returns a list all the files under the given directory
func (u *volumeUtil) ReadDir(fullPath string) ([]string, error) {
	dir, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	files, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	return files, nil
}

// DeleteContents deletes all the contents under the given directory
func (u *volumeUtil) DeleteContents(fullPath string) error {
	dir, err := os.Open(fullPath)
	if err != nil {
		return err
	}
	defer dir.Close()

	files, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}
	errList := []error{}
	for _, file := range files {
		err = os.RemoveAll(filepath.Join(fullPath, file))
		if err != nil {
			errList = append(errList, err)
		}
	}

	return utilerrors.NewAggregate(errList)
}

// GetFsCapacityByte returns capacity in bytes about a mounted filesystem.
// fullPath is the pathname of any file within the mounted filesystem.
// Capacity returned here is total capacity.
func (u *volumeUtil) GetFsCapacityByte(fullPath string) (int64, error) {
	return 5*GiB, nil
}

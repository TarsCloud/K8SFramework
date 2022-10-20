package storage

import (
	"fmt"
	"os"
	"syscall"
)

const (
	TLVPVProvisioner = "tars-local-volume-provisioner"

	PVCProtectionFinalizer = "kubernetes.io/pvc-protection"

	PVProtectionFinalizer = "kubernetes.io/pv-protection"

	AnnDynamicallyProvisioned = "pv.kubernetes.io/provisioned-by"

	DefaultPerm = "0755"

	NodeNameEnv = "NodeName"

	TLVInPod = "/usr/local/app/tars/host-mount"
)

type ProvisioningState string

const (
	ProvisioningAgain ProvisioningState = "TryAgain"

	ProvisioningFinished ProvisioningState = "Finished"
)

type IgnoredError struct {
	Reason string
}

func (e *IgnoredError) Error() string {
	return fmt.Sprintf("ignored because %s", e.Reason)
}

type VolumeUtil struct{}

// Existed checks if the given path is existed
func (u *VolumeUtil) Existed(fullPath string) bool {
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return false
	}
	return true
}

// IsDir checks if the given path is a directory
func (u *VolumeUtil) IsDir(fullPath string) (bool, error) {
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

func (u *VolumeUtil) GetFilePerm(fullPath string) (os.FileMode, error) {
	stat, err := os.Stat(fullPath)
	if err != nil {
		return 0, err
	}
	return stat.Mode().Perm(), nil
}

func (u *VolumeUtil) GetFileUidGid(fullPath string) (int, int, error) {
	stat, err := os.Stat(fullPath)
	if err != nil {
		return 0, 0, err
	}
	var uid, gid int
	if info, ok := stat.Sys().(*syscall.Stat_t); ok {
		uid = int(info.Uid)
		gid = int(info.Gid)
	} else {
		uid = os.Getuid()
		gid = os.Getgid()
	}
	return uid, gid, nil
}

func (u *VolumeUtil) MakeDir(fullPath string, perm os.FileMode) error {
	oldMask := syscall.Umask(0)
	if err := os.MkdirAll(fullPath, perm); err != nil {
		return err
	}
	syscall.Umask(oldMask)

	return nil
}

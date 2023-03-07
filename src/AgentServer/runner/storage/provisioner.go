package storage

import (
	"fmt"
	"hash/fnv"
	k8sCoreV1 "k8s.io/api/core/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	tarsMeta "k8s.tars.io/meta"
	"os"
	"path"
	"strconv"
	"strings"
	"tarsagent/gflag"
)

type TLocalProvisioner struct {
	podBase            string
	hostBase           string
	identity           string
	node               string
	name               string
	supportLocalVolume bool
	volumeUtil         *VolumeUtil
	reclaimPolicy      k8sCoreV1.PersistentVolumeReclaimPolicy
}

type TLocalVolumeMatchInfo struct {
	app         string
	server      string
	directory   string
	podABSPath  string
	hostABSPath string
	volumeName  string
}

type TLocalVolumeModeInfo struct {
	uid  int
	gid  int
	perm os.FileMode
}

// newTLocalProvisioner creates a new tars local provisioner
func newTLocalProvisioner() *TLocalProvisioner {
	node := gflag.NodeName
	h := fnv.New32a()
	_, _ = h.Write([]byte(gflag.NodeName))
	_, _ = h.Write([]byte(tarsMeta.TStorageClassName))

	return &TLocalProvisioner{
		podBase:            TLVInPod,
		hostBase:           gflag.TLVInHost,
		identity:           fmt.Sprintf("%x", h.Sum32()),
		node:               node,
		name:               TLVPVProvisioner,
		supportLocalVolume: false,
		volumeUtil:         &VolumeUtil{},
		reclaimPolicy:      k8sCoreV1.PersistentVolumeReclaimRetain,
	}
}

func (p *TLocalProvisioner) GetVolumePathInfo(claim *k8sCoreV1.PersistentVolumeClaim) (*TLocalVolumeMatchInfo, error) {
	if claim.Spec.Selector == nil || claim.Spec.Selector.MatchLabels == nil {
		return nil, fmt.Errorf("unexecptec claim spec.selector")
	}

	matchLabels := claim.Spec.Selector.MatchLabels
	app, _ := matchLabels[tarsMeta.TServerAppLabel]
	server, _ := matchLabels[tarsMeta.TServerNameLabel]
	directory, _ := matchLabels[tarsMeta.TLocalVolumeLabel]

	volumeName := fmt.Sprintf("%s-%s-%s-%s-%s", claim.Namespace, directory, strings.ToLower(app), strings.ToLower(server), p.identity)
	if claim.Spec.VolumeName != "" && claim.Spec.VolumeName != volumeName {
		return nil, fmt.Errorf("unexecptec claim spec.volumeName value")
	}

	if app == "" || server == "" || directory == "" {
		return nil, fmt.Errorf("unexecptec claim spec.selector")
	}

	rePath := fmt.Sprintf("%s/%s.%s/%s", claim.Namespace, app, server, directory)

	info := &TLocalVolumeMatchInfo{
		app:         app,
		server:      server,
		directory:   directory,
		podABSPath:  path.Join(p.podBase, rePath),
		hostABSPath: path.Join(p.hostBase, rePath),
		volumeName:  volumeName,
	}
	return info, nil
}

func (p *TLocalProvisioner) GetVolumeModeInfo(claim *k8sCoreV1.PersistentVolumeClaim) *TLocalVolumeModeInfo {
	var gid, uid = 0, 0
	var perm int64
	if claim.Annotations != nil {
		permAnn := claim.Annotations[tarsMeta.TLocalVolumeModeAnnotation]
		if permAnn == "" {
			permAnn = DefaultPerm
		}
		perm, _ = strconv.ParseInt(permAnn, 8, 32)
		if uidAnn := claim.Annotations[tarsMeta.TLocalVolumeUIDAnnotation]; uidAnn != "" {
			uid, _ = strconv.Atoi(uidAnn)
		}

		if gidAnn := claim.Annotations[tarsMeta.TLocalVolumeGIDAnnotation]; gidAnn != "" {
			gid, _ = strconv.Atoi(gidAnn)
		}
	}

	return &TLocalVolumeModeInfo{
		uid:  uid,
		gid:  gid,
		perm: os.FileMode(perm),
	}
}

// Provision creates a storage asset and returns a PV object representing it.
func (p *TLocalProvisioner) Provision(claim *k8sCoreV1.PersistentVolumeClaim) (*k8sCoreV1.PersistentVolume, ProvisioningState, error) {
	pathInfo, err := p.GetVolumePathInfo(claim)
	if err != nil {
		return nil, ProvisioningFinished, err
	}

	if pathInfo.directory != "delay-bind" && !p.supportLocalVolume {
		return nil, ProvisioningAgain, fmt.Errorf("node not support tars local volume now")
	}

	modeInfo := p.GetVolumeModeInfo(claim)

	err = p.syncVolume(pathInfo.podABSPath, modeInfo.perm, modeInfo.uid, modeInfo.gid)
	if err != nil {
		return nil, ProvisioningAgain, err
	}

	pv := &k8sCoreV1.PersistentVolume{
		TypeMeta: k8sMetaV1.TypeMeta{},
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:   pathInfo.volumeName,
			Labels: claim.Spec.Selector.MatchLabels,
			Annotations: map[string]string{
				AnnDynamicallyProvisioned: p.name,
			},
			Finalizers: []string{PVProtectionFinalizer},
		},
		Spec: k8sCoreV1.PersistentVolumeSpec{
			Capacity: k8sCoreV1.ResourceList{
				k8sCoreV1.ResourceStorage: claim.Spec.Resources.Requests[k8sCoreV1.ResourceStorage],
			},
			PersistentVolumeSource: k8sCoreV1.PersistentVolumeSource{
				Local: &k8sCoreV1.LocalVolumeSource{
					Path: pathInfo.hostABSPath,
				},
			},
			AccessModes:                   claim.Spec.AccessModes,
			PersistentVolumeReclaimPolicy: p.reclaimPolicy,
			StorageClassName:              tarsMeta.TStorageClassName,
			MountOptions:                  nil,
			VolumeMode:                    nil,
			NodeAffinity: &k8sCoreV1.VolumeNodeAffinity{
				Required: &k8sCoreV1.NodeSelector{
					NodeSelectorTerms: []k8sCoreV1.NodeSelectorTerm{
						{
							MatchExpressions: []k8sCoreV1.NodeSelectorRequirement{
								{
									Key:      tarsMeta.K8SHostNameLabel,
									Operator: k8sCoreV1.NodeSelectorOpIn,
									Values:   []string{p.node},
								},
							},
						},
					},
				},
			},
		},
	}

	return pv, ProvisioningFinished, nil
}

func (p *TLocalProvisioner) VolumeName(claim *k8sCoreV1.PersistentVolumeClaim) (string, error) {
	info, err := p.GetVolumePathInfo(claim)
	if err != nil {
		return "", err
	}
	return info.volumeName, nil
}

func (p *TLocalProvisioner) Delete(volume *k8sCoreV1.PersistentVolume) error {
	if p.reclaimPolicy != k8sCoreV1.PersistentVolumeReclaimDelete {
		hostAbsPath := volume.Spec.Local.Path
		podAbsPath := strings.Replace(hostAbsPath, p.hostBase, p.podBase, 1)
		if err := os.RemoveAll(podAbsPath); err != nil {
			return err
		}
	}
	return nil
}

func (p *TLocalProvisioner) ProvisionedBy(name string) bool {
	return strings.HasSuffix(name, p.identity)
}

func (p *TLocalProvisioner) syncVolume(path string, perm os.FileMode, uid int, gid int) error {
	existed := p.volumeUtil.Existed(path)

	if existed {
		ok, err := p.volumeUtil.IsDir(path)
		if err != nil {
			return fmt.Errorf("get path(%s) stat failed: %s", path, err.Error())
		}

		if !ok {
			klog.Infof("begin to remove path(%s)", path)
			err = os.RemoveAll(path)
			if err != nil {
				return fmt.Errorf("remove path(%s) failed: %s", path, err.Error())
			}
			klog.Infof("remove path(%s) success")
			existed = false
		}
	}

	if existed {
		currentPerm, err := p.volumeUtil.GetFilePerm(path)
		if err != nil {
			return fmt.Errorf("get path(%s) perm failed: %s", path, err.Error())
		}
		if currentPerm.String() != perm.String() {
			err = os.Chmod(path, perm)
			if err != nil {
				return fmt.Errorf("chmod path(%s) to perm(%s) failed: %s", path, perm.String(), err.Error())
			}
			klog.Infof("chmod path(%s) to perm(%s) success", path, perm.String())
		}
	} else {
		if err := p.volumeUtil.MakeDir(path, perm); err != nil {
			return fmt.Errorf("mkdir path(%s) failed: %s", path, err.Error())
		}
	}

	currentUid, currentGid, err := p.volumeUtil.GetFileUidGid(path)
	if currentUid != uid || currentGid != gid {
		err = os.Chown(path, uid, gid)
		if err != nil {
			return fmt.Errorf("chown path(%s) to %d:%d failed: %s", path, gid, uid, err.Error())
		}
	}

	return nil
}

func (p *TLocalProvisioner) SyncClaim(claim *k8sCoreV1.PersistentVolumeClaim) (ProvisioningState, error) {
	pathInfo, err := p.GetVolumePathInfo(claim)
	if err != nil {
		return ProvisioningFinished, err
	}

	modeInfo := p.GetVolumeModeInfo(claim)

	if err = p.syncVolume(pathInfo.podABSPath, modeInfo.perm, modeInfo.uid, modeInfo.gid); err != nil {

	}
	return ProvisioningFinished, nil
}

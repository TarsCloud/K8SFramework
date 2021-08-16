package image

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	k8sCoreV1 "k8s.io/api/core/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	crdV1Alpha1 "k8s.tars.io/api/crd/v1alpha1"
	"strings"
	"tarsagent/controller/common"
	"time"
)

// The Downloader uses an Informer to download the Docker Image.
type Downloader struct {
	*common.RuntimeConfig
}

// NewDownloader returns a Downloader object to download image
func NewDownloader(config *common.RuntimeConfig) *Downloader {
	d := &Downloader{RuntimeConfig: config}
	sharedInformer := config.CrdInformerFactory.Crd().V1alpha1().TImages()
	sharedInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			timage, ok := obj.(*crdV1Alpha1.TImage)
			if !ok {
				glog.Errorf("Added object is not a crdV1Alpha1.TImage type")
				return
			}
			if !d.preDownloadCheck(timage) {
				return
			}
			glog.Infof("Ready to download image %s.%s", timage.ImageType, timage.Name)
			go d.downloadDockerImage(timage)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldTImage, ok := newObj.(*crdV1Alpha1.TImage)
			if !ok {
				glog.Errorf("Updated object is not a crdV1Alpha1.TImage type")
				return
			}
			newTImage, ok := newObj.(*crdV1Alpha1.TImage)
			if !ok {
				glog.Errorf("Updated object is not a crdV1Alpha1.TImage type")
				return
			}
			if oldTImage.GetResourceVersion() == newTImage.GetResourceVersion() {
				return
			}
			if !d.preDownloadCheck(newTImage) {
				return
			}
			glog.Infof("Ready to download image %s.%s", newTImage.ImageType, newTImage.Name)
			go d.downloadDockerImage(newTImage)
		},
		DeleteFunc: func(obj interface{}) {
			return
		},
	})
	return d
}

func (d *Downloader) preDownloadCheck(timage *crdV1Alpha1.TImage) bool {
	// timage empty
	if len(timage.Releases) <= 0 {
		return false
	}

	// image type
	if timage.ImageType != common.TImageNodeType &&
		timage.ImageType != common.TImageBaseType &&
		timage.ImageType != common.TImageServerType {
		return false
	}

	// node taint
	node := d.RuntimeConfig.Node
	for _, taint := range node.Spec.Taints {
		if taint.Key == "node.kubernetes.io/disk-pressure" || taint.Key == "node.kubernetes.io/out-of-disk" ||
			taint.Key == "node.kubernetes.io/unschedulable" {
			return false
		}
	}

	// node ability
	if node.Labels == nil {
		return false
	}

	_, ok := node.Labels[fmt.Sprintf("%s.%s",
		common.NodeNamespaceAffinityPrefix, timage.Namespace)]
	if !ok {
		return false
	}

	if timage.ImageType == common.TImageServerType {
		if timage.Labels == nil {
			return false
		}
		ServerApp, ok := timage.Labels[common.TServerAppLabel]
		if !ok {
			return false
		}
		_, ok = node.Labels[fmt.Sprintf("%s.%s.%s",
			common.NodeServerAppAffinityPrefix, timage.Namespace, ServerApp)]
		if !ok {
			return false
		}
	}

	return true
}

func (d *Downloader) downloadDockerImage(timage *crdV1Alpha1.TImage) {
	// Ready to download images with builtin or createTime properties in healthy node
	validImages := make([]*crdV1Alpha1.TImageRelease, 0, len(timage.Releases))
	for _, image := range timage.Releases {
		if strings.Contains(image.ID, "builtin") {
			validImages = append(validImages, image)
		} else if image.CreateTime.Equal(&k8sMetaV1.Time{}) {
			continue
		} else {
			validImages = append(validImages, image)
		}
	}
	if len(validImages) <= 0 {
		return
	}

	dockerClient, err := NewDockerClient()
	if err != nil {
		glog.Errorf("create docker interface error: %s\n", err.Error())
		return
	}

	for _, image := range validImages {
		var secret *k8sCoreV1.Secret

		if image.Secret != "" {
			secret, err = d.K8sClient.CoreV1().Secrets(timage.Namespace).
				Get(context.TODO(), image.Secret, k8sMetaV1.GetOptions{})
			if err != nil {
				glog.Errorf("get secret %s error: %s\n", image.Secret, err.Error())
				continue
			}
		}

		if err = dockerClient.pullImage(image.Image, 5*time.Minute, secret); err != nil {
			glog.Errorf("fail to pull image %s: %s\n", image.Image, err.Error())
			continue
		}
		glog.Infof("succ to pull image %s \n", image.Image)
	}
}


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

package controller

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/utils/mount"
	crdClientSet "k8s.tars.io/client-go/clientset/versioned"
	crdInformers "k8s.tars.io/client-go/informers/externalversions"
	"math/rand"
	"os"
	"tarsagent/controller/common"
	"tarsagent/controller/image"
	"tarsagent/controller/localpv"
	"time"

	"github.com/golang/glog"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

// StartController starts the sync loop for the TImage downloader and local PV discovery/deleter
func StartController(namespace string, k8sConfig *rest.Config) {
	glog.Info("Initializing agent controller\n")

	k8sClient, crdClient, err := getClientSet(k8sConfig)
	if err != nil {
		return
	}

	k8sNode, err := getNode(k8sClient)
	if err != nil {
		return
	}

	startController(namespace, k8sClient, crdClient, k8sNode)

	glog.Info("Agent controller started\n")
}

func getClientSet(k8sConfig *rest.Config) (*kubernetes.Clientset, *crdClientSet.Clientset, error) {
	k8sClient, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		glog.Fatalf("Error creating clientset: %v\n", err)
		return nil, nil, err
	}

	crdClient, err := crdClientSet.NewForConfig(k8sConfig)
	if err != nil {
		glog.Fatalf("Error creating clientset: %v\n", err)
		return nil, nil, err
	}

	return k8sClient, crdClient, nil
}

func getNode(k8sClient *kubernetes.Clientset) (*v1.Node, error) {
	nodeName := os.Getenv("NodeName")
	if nodeName == "" {
		glog.Fatalf("environment variable NodeName not set\n")
		return nil, fmt.Errorf("empty node name.\n")
	}

	k8sNode, err := k8sClient.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		glog.Fatalf("Could not get node information: %v", err)
		return nil, err
	}

	return k8sNode, nil
}

func startController(namespace string, k8sClient *kubernetes.Clientset, crdClient *crdClientSet.Clientset, k8sNode *v1.Node) {
	discoveryMap := make(map[string]common.MountConfig)
	discoveryMap[common.TStorageClassName] = common.MountConfig{
		HostDir: common.TLocalVolumeHostDir,
		MountDir: common.TLocalVolumeHostDir,
		VolumeMode: common.TLocalVolumeMode,
	}

	nodeLabelForPV := []string{common.NodeLabelKey}

	usrConfig := &common.UserConfig{
		Node:            k8sNode,
		DiscoveryMap:	 discoveryMap,
		NodeLabelsForPV: nodeLabelForPV,
		MinResyncPeriod: metav1.Duration{Duration: 5 * time.Minute},
		Namespace:       namespace,
	}

	agentName := fmt.Sprintf("agent-downloader-provisioner-%v", usrConfig.Node.Name)

	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: v1core.New(k8sClient.CoreV1().RESTClient()).Events("")})
	recorder := broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: agentName})

	// We choose a random resync period between MinResyncPeriod and 2 * MinResyncPeriod,
	// So that local provisioners deployed on multiple nodes at same time don't list the apiserver simultaneously.
	resyncPeriod := time.Duration(usrConfig.MinResyncPeriod.Seconds()*(1+rand.Float64())) * time.Second

	runtimeConfig := &common.RuntimeConfig{
		UserConfig:         usrConfig,
		Cache:           	common.NewVolumeCache(),
		VolUtil:         	common.NewVolumeUtil(),
		APIUtil:         	common.NewAPIUtil(k8sClient),
		K8sClient:          k8sClient,
		CrdClient:          crdClient,
		Name:               agentName,
		Recorder:           recorder,
		Mounter:            mount.New("" /* default mount path */),
		K8sInformerFactory: informers.NewSharedInformerFactory(k8sClient, resyncPeriod),
		CrdInformerFactory: crdInformers.NewSharedInformerFactory(crdClient, resyncPeriod),
	}

	// Image downloader
	image.NewDownloader(runtimeConfig)

	// Local PV cache/discovery/deleter
	discoverer, deleter, err := getLocalPVComponents(runtimeConfig)
	if err != nil {
		glog.Fatalf("Error initializing localpv components: %v", err)
	}

	// Start k8s informers after all event listeners are registered.
	runtimeConfig.K8sInformerFactory.Start(wait.NeverStop)
	for v, synced := range runtimeConfig.K8sInformerFactory.WaitForCacheSync(wait.NeverStop) {
		if !synced {
			glog.Fatalf("Error syncing k8s informer for %v", v)
		}
	}

	// Start crd informers after all event listeners are registered.
	runtimeConfig.CrdInformerFactory.Start(wait.NeverStop)
	for v, synced := range runtimeConfig.CrdInformerFactory.WaitForCacheSync(wait.NeverStop) {
		if !synced {
			glog.Fatalf("Error syncing crd informer for %v", v)
		}
	}
	for {
		runtimeConfig.Node, _ = getNode(k8sClient)
		deleter.DeletePVs()
		discoverer.DiscoverLocalVolumes()
		time.Sleep(10 * time.Second)
	}
}

func getLocalPVComponents(runtimeConfig *common.RuntimeConfig) (*localpv.Discoverer, *localpv.Deleter, error) {
	// Local PV cache
	localpv.NewPopulator(runtimeConfig)

	// Local PV discovery
	procTable := localpv.NewProcTable()
	cleanupTracker := &localpv.CleanupStatusTracker{ProcTable: procTable}

	discoverer, err := localpv.NewDiscoverer(runtimeConfig, cleanupTracker)
	if err != nil {
		return nil, nil, err
	}

	// Local PV deleter
	deleter := localpv.NewDeleter(runtimeConfig, cleanupTracker)

	return discoverer, deleter, nil
}

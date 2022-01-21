package controller

import (
	"fmt"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sWatchV1 "k8s.io/apimachinery/pkg/watch"
	k8sInformers "k8s.io/client-go/informers"
	k8sInformersAppsV1 "k8s.io/client-go/informers/apps/v1"
	k8sInformersCoreV1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/metadata/metadatainformer"
	"k8s.io/client-go/tools/cache"
	crdV1beta2 "k8s.tars.io/api/crd/v1beta2"
	crdMeta "k8s.tars.io/api/meta"
	crdInformers "k8s.tars.io/client-go/informers/externalversions"
	crdInformersV1beta2 "k8s.tars.io/client-go/informers/externalversions/crd/v1beta2"
)

type EventsReceiver interface {
	EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{})
}

type Informers struct {
	k8sInformerFactory           k8sInformers.SharedInformerFactory
	k8sInformerFactoryWithFilter k8sInformers.SharedInformerFactory
	k8sMetadataInformerFactor    metadatainformer.SharedInformerFactory
	crdInformerFactory           crdInformers.SharedInformerFactory

	synced  bool
	synceds []cache.InformerSynced

	NodeInformer                  k8sInformersCoreV1.NodeInformer
	ServiceInformer               k8sInformersCoreV1.ServiceInformer
	PodInformer                   k8sInformersCoreV1.PodInformer
	PersistentVolumeClaimInformer k8sInformersCoreV1.PersistentVolumeClaimInformer

	DaemonSetInformer   k8sInformersAppsV1.DaemonSetInformer
	StatefulSetInformer k8sInformersAppsV1.StatefulSetInformer

	TServerInformer   crdInformersV1beta2.TServerInformer
	TEndpointInformer crdInformersV1beta2.TEndpointInformer

	TImageInformer           crdInformersV1beta2.TImageInformer
	TTreeInformer            crdInformersV1beta2.TTreeInformer
	TExitedRecordInformer    crdInformersV1beta2.TExitedRecordInformer
	TDeployInformer          crdInformersV1beta2.TDeployInformer
	TAccountInformer         crdInformersV1beta2.TAccountInformer
	TFrameworkConfigInformer crdInformersV1beta2.TFrameworkConfigInformer

	TConfigInformer   k8sInformers.GenericInformer
	TTemplateInformer k8sInformers.GenericInformer

	receivers []EventsReceiver
}

func newInformers(clients *Clients) *Informers {

	k8sInformerFactory := k8sInformers.NewSharedInformerFactory(clients.K8sClient, 0)

	k8sInformerFactoryWithFilter := k8sInformers.NewSharedInformerFactoryWithOptions(clients.K8sClient, 0, k8sInformers.WithTweakListOptions(
		func(options *k8sMetaV1.ListOptions) {
			options.LabelSelector = fmt.Sprintf("%s,%s", crdMeta.TServerAppLabel, crdMeta.TServerNameLabel)
		}))

	crdInformerFactory := crdInformers.NewSharedInformerFactoryWithOptions(clients.CrdClient, 0)

	metadataInformerFactory := metadatainformer.NewSharedInformerFactory(clients.K8sMetadataClient, 0)

	nodeInformer := k8sInformerFactory.Core().V1().Nodes()

	serviceInformer := k8sInformerFactoryWithFilter.Core().V1().Services()
	podInformer := k8sInformerFactoryWithFilter.Core().V1().Pods()
	persistentVolumeClaimInformer := k8sInformerFactoryWithFilter.Core().V1().PersistentVolumeClaims()
	daemonSetInformer := k8sInformerFactoryWithFilter.Apps().V1().DaemonSets()
	statefulSetInformer := k8sInformerFactoryWithFilter.Apps().V1().StatefulSets()

	tserverInformer := crdInformerFactory.Crd().V1beta2().TServers()
	tendpointInformer := crdInformerFactory.Crd().V1beta2().TEndpoints()

	timageInformer := crdInformerFactory.Crd().V1beta2().TImages()
	ttreeInformer := crdInformerFactory.Crd().V1beta2().TTrees()
	texitedRecordInformer := crdInformerFactory.Crd().V1beta2().TExitedRecords()
	tdeployInformer := crdInformerFactory.Crd().V1beta2().TDeploys()
	taccountInformer := crdInformerFactory.Crd().V1beta2().TAccounts()
	tframeworkconfigInformer := crdInformerFactory.Crd().V1beta2().TFrameworkConfigs()

	tconfigInformer := metadataInformerFactory.ForResource(crdV1beta2.SchemeGroupVersion.WithResource("tconfigs"))
	ttemplateInformer := metadataInformerFactory.ForResource(crdV1beta2.SchemeGroupVersion.WithResource("ttemplates"))

	informers = &Informers{
		k8sInformerFactory:           k8sInformerFactory,
		k8sInformerFactoryWithFilter: k8sInformerFactoryWithFilter,
		k8sMetadataInformerFactor:    metadataInformerFactory,
		crdInformerFactory:           crdInformerFactory,

		NodeInformer:                  nodeInformer,
		ServiceInformer:               serviceInformer,
		PodInformer:                   podInformer,
		PersistentVolumeClaimInformer: persistentVolumeClaimInformer,

		DaemonSetInformer:   daemonSetInformer,
		StatefulSetInformer: statefulSetInformer,

		TServerInformer:       tserverInformer,
		TEndpointInformer:     tendpointInformer,
		TTemplateInformer:     ttemplateInformer,
		TImageInformer:        timageInformer,
		TTreeInformer:         ttreeInformer,
		TExitedRecordInformer: texitedRecordInformer,
		TDeployInformer:       tdeployInformer,
		TAccountInformer:      taccountInformer,

		TConfigInformer:          tconfigInformer,
		TFrameworkConfigInformer: tframeworkconfigInformer,

		synced: false,
		synceds: []cache.InformerSynced{
			nodeInformer.Informer().HasSynced,
			serviceInformer.Informer().HasSynced,
			podInformer.Informer().HasSynced,
			persistentVolumeClaimInformer.Informer().HasSynced,

			statefulSetInformer.Informer().HasSynced,
			daemonSetInformer.Informer().HasSynced,

			tserverInformer.Informer().HasSynced,
			tendpointInformer.Informer().HasSynced,
			ttemplateInformer.Informer().HasSynced,
			timageInformer.Informer().HasSynced,
			ttreeInformer.Informer().HasSynced,
			texitedRecordInformer.Informer().HasSynced,
			tdeployInformer.Informer().HasSynced,
			taccountInformer.Informer().HasSynced,
			tframeworkconfigInformer.Informer().HasSynced,

			tconfigInformer.Informer().HasSynced,
		},
	}

	setEventHandler("node", informers.NodeInformer.Informer(), informers)
	setEventHandler("service", informers.ServiceInformer.Informer(), informers)
	setEventHandler("pod", informers.PodInformer.Informer(), informers)
	setEventHandler("persistentvolumeclaim", persistentVolumeClaimInformer.Informer(), informers)

	setEventHandler("statefulset", informers.StatefulSetInformer.Informer(), informers)
	setEventHandler("daemonset", informers.DaemonSetInformer.Informer(), informers)

	setEventHandler("tserver", informers.TServerInformer.Informer(), informers)
	setEventHandler("tendpoint", informers.TEndpointInformer.Informer(), informers)
	setEventHandler("ttemplate", informers.TTemplateInformer.Informer(), informers)
	setEventHandler("timage", informers.TImageInformer.Informer(), informers)
	setEventHandler("ttree", informers.TTreeInformer.Informer(), informers)
	setEventHandler("texitedrecord", informers.TExitedRecordInformer.Informer(), informers)
	setEventHandler("tdeploy", informers.TDeployInformer.Informer(), informers)
	setEventHandler("taccount", informers.TAccountInformer.Informer(), informers)
	setEventHandler("tframeworkconfig", informers.TFrameworkConfigInformer.Informer(), informers)
	setEventHandler("tconfig", informers.TConfigInformer.Informer(), informers)

	return informers
}

func (i *Informers) send(name string, event k8sWatchV1.EventType, obj interface{}) {
	for _, r := range i.receivers {
		r.EnqueueObj(name, event, obj)
	}
}

func (i *Informers) Register(r EventsReceiver) {
	i.receivers = append(i.receivers, r)
}

func setEventHandler(resourceKind string, resourceInformer cache.SharedInformer, informers *Informers) {
	eventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			informers.send(resourceKind, k8sWatchV1.Added, obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldMeta := oldObj.(k8sMetaV1.Object)
			newMeta := newObj.(k8sMetaV1.Object)
			if newMeta.GetResourceVersion() != oldMeta.GetResourceVersion() {
				informers.send(resourceKind, k8sWatchV1.Modified, newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			informers.send(resourceKind, k8sWatchV1.Deleted, obj)
		},
	}
	resourceInformer.AddEventHandler(eventHandler)
}

func (i *Informers) Start(stop chan struct{}) {
	i.k8sInformerFactory.Start(stop)
	i.k8sInformerFactoryWithFilter.Start(stop)
	i.k8sMetadataInformerFactor.Start(stop)
	i.crdInformerFactory.Start(stop)
}

func (i *Informers) Synced() bool {
	return i.synced
}

func (i *Informers) WaitForCacheSync(stopCh chan struct{}) bool {
	i.synced = cache.WaitForCacheSync(stopCh, i.synceds...)
	return i.synced
}

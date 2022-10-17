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
	tarsInformers "k8s.tars.io/client-go/informers/externalversions"
	tarsInformersV1beta3 "k8s.tars.io/client-go/informers/externalversions/crd/v1beta3"
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
	tarsMeta "k8s.tars.io/meta"
)

type EventsReceiver interface {
	EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{})
}

type Informers struct {
	k8sInformerFactory           k8sInformers.SharedInformerFactory
	k8sInformerFactoryWithFilter k8sInformers.SharedInformerFactory
	k8sMetadataInformerFactor    metadatainformer.SharedInformerFactory
	tarsInformerFactory          tarsInformers.SharedInformerFactory

	synced  bool
	synceds []cache.InformerSynced

	NodeInformer                  k8sInformersCoreV1.NodeInformer
	ServiceInformer               k8sInformersCoreV1.ServiceInformer
	PodInformer                   k8sInformersCoreV1.PodInformer
	PersistentVolumeClaimInformer k8sInformersCoreV1.PersistentVolumeClaimInformer

	DaemonSetInformer   k8sInformersAppsV1.DaemonSetInformer
	StatefulSetInformer k8sInformersAppsV1.StatefulSetInformer

	TServerInformer   tarsInformersV1beta3.TServerInformer
	TEndpointInformer tarsInformersV1beta3.TEndpointInformer

	TImageInformer           tarsInformersV1beta3.TImageInformer
	TTreeInformer            tarsInformersV1beta3.TTreeInformer
	TExitedRecordInformer    tarsInformersV1beta3.TExitedRecordInformer
	TAccountInformer         tarsInformersV1beta3.TAccountInformer
	TFrameworkConfigInformer tarsInformersV1beta3.TFrameworkConfigInformer

	TConfigInformer   k8sInformers.GenericInformer
	TTemplateInformer k8sInformers.GenericInformer

	receivers []EventsReceiver
}

func newInformers(clients *Clients) *Informers {

	k8sInformerFactory := k8sInformers.NewSharedInformerFactory(clients.K8sClient, 0)

	k8sInformerFactoryWithFilter := k8sInformers.NewSharedInformerFactoryWithOptions(clients.K8sClient, 0, k8sInformers.WithTweakListOptions(
		func(options *k8sMetaV1.ListOptions) {
			options.LabelSelector = fmt.Sprintf("%s,%s", tarsMeta.TServerAppLabel, tarsMeta.TServerNameLabel)
		}))

	tarsInformerFactory := tarsInformers.NewSharedInformerFactoryWithOptions(clients.CrdClient, 0)

	metadataInformerFactory := metadatainformer.NewSharedInformerFactory(clients.K8sMetadataClient, 0)

	nodeInformer := k8sInformerFactory.Core().V1().Nodes()

	serviceInformer := k8sInformerFactoryWithFilter.Core().V1().Services()
	podInformer := k8sInformerFactoryWithFilter.Core().V1().Pods()
	persistentVolumeClaimInformer := k8sInformerFactoryWithFilter.Core().V1().PersistentVolumeClaims()
	daemonSetInformer := k8sInformerFactoryWithFilter.Apps().V1().DaemonSets()
	statefulSetInformer := k8sInformerFactoryWithFilter.Apps().V1().StatefulSets()

	tserverInformer := tarsInformerFactory.Crd().V1beta3().TServers()
	tendpointInformer := tarsInformerFactory.Crd().V1beta3().TEndpoints()

	timageInformer := tarsInformerFactory.Crd().V1beta3().TImages()
	ttreeInformer := tarsInformerFactory.Crd().V1beta3().TTrees()
	texitedRecordInformer := tarsInformerFactory.Crd().V1beta3().TExitedRecords()

	taccountInformer := tarsInformerFactory.Crd().V1beta3().TAccounts()
	tframeworkconfigInformer := tarsInformerFactory.Crd().V1beta3().TFrameworkConfigs()

	tconfigInformer := metadataInformerFactory.ForResource(tarsCrdV1beta3.SchemeGroupVersion.WithResource("tconfigs"))
	ttemplateInformer := metadataInformerFactory.ForResource(tarsCrdV1beta3.SchemeGroupVersion.WithResource("ttemplates"))

	informers = &Informers{
		k8sInformerFactory:           k8sInformerFactory,
		k8sInformerFactoryWithFilter: k8sInformerFactoryWithFilter,
		k8sMetadataInformerFactor:    metadataInformerFactory,
		tarsInformerFactory:          tarsInformerFactory,

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
	i.tarsInformerFactory.Start(stop)
}

func (i *Informers) Synced() bool {
	return i.synced
}

func (i *Informers) WaitForCacheSync(stopCh chan struct{}) bool {
	i.synced = cache.WaitForCacheSync(stopCh, i.synceds...)
	return i.synced
}

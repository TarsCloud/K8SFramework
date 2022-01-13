package main

import (
	"k8s.io/client-go/tools/cache"
	crdV1beta2 "k8s.tars.io/api/crd/v1beta2"
	"k8s.tars.io/api/meta"
	crdInformers "k8s.tars.io/client-go/informers/externalversions"
	crdInformersV1beta2 "k8s.tars.io/client-go/informers/externalversions/crd/v1beta2"
)

type Informers struct {
	crdInformerFactory       crdInformers.SharedInformerFactory
	synced                   []cache.InformerSynced
	TFrameworkConfigInformer crdInformersV1beta2.TFrameworkConfigInformer
}

var _defaultDockerRegistry string
var _defaultDockerSecret string

type Watcher struct {
	k8SContext *K8SContext
	Informers
}

func NewWatcher(k8sContext *K8SContext) *Watcher {
	crdInformerFactory := crdInformers.NewSharedInformerFactoryWithOptions(k8sContext.crdClient, 0, crdInformers.WithNamespace(k8sContext.namespace))
	tframeworkconfigInformer := crdInformerFactory.Crd().V1beta2().TFrameworkConfigs()
	watcher := &Watcher{
		k8SContext: k8sContext,
		Informers: Informers{
			crdInformerFactory:       crdInformerFactory,
			TFrameworkConfigInformer: tframeworkconfigInformer,
			synced:                   []cache.InformerSynced{tframeworkconfigInformer.Informer().HasSynced},
		},
	}

	watcher.TFrameworkConfigInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				switch obj.(type) {
				case *crdV1beta2.TFrameworkConfig:
					tfc := obj.(*crdV1beta2.TFrameworkConfig)
					if tfc.Name == meta.FixedTFrameworkConfigResourceName {
						setMaxReleases(tfc.RecordLimit.TImageRelease)
						setRegistry(tfc.ImageRegistry.Registry, tfc.ImageRegistry.Secret)
						setTagFormat(tfc.ImageBuild.TagFormat)
					}
				default:
					return
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				switch newObj.(type) {
				case *crdV1beta2.TFrameworkConfig:
					tfc := newObj.(*crdV1beta2.TFrameworkConfig)
					if tfc.Name == meta.FixedTFrameworkConfigResourceName {
						setMaxReleases(tfc.RecordLimit.TImageRelease)
						setRegistry(tfc.ImageRegistry.Registry, tfc.ImageRegistry.Secret)
						setTagFormat(tfc.ImageBuild.TagFormat)
					}
				default:
					return
				}
			},
			DeleteFunc: func(obj interface{}) {
			},
		},
	)
	return watcher
}

func (w *Watcher) Start(stopChan chan struct{}) {
	w.crdInformerFactory.Start(stopChan)
}

func (w *Watcher) GetDockerRegistry() (registry, secret string) {
	return _defaultDockerRegistry, _defaultDockerSecret
}

func (w *Watcher) WaitSync(stopCh chan struct{}) bool {
	return cache.WaitForCacheSync(stopCh, w.synced...)
}

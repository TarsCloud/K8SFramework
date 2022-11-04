package main

import (
	"k8s.io/client-go/tools/cache"
	tarsAppsV1beta3 "k8s.tars.io/apps/v1beta3"
	tarsInformers "k8s.tars.io/client-go/informers/externalversions"
	tarsInformersV1beta3 "k8s.tars.io/client-go/informers/externalversions/apps/v1beta3"
	tarsMeta "k8s.tars.io/meta"
)

type Informers struct {
	crdInformerFactory       tarsInformers.SharedInformerFactory
	synced                   []cache.InformerSynced
	TFrameworkConfigInformer tarsInformersV1beta3.TFrameworkConfigInformer
}

type Watcher struct {
	Informers
}

func NewWatcher() *Watcher {
	crdInformerFactory := tarsInformers.NewSharedInformerFactoryWithOptions(glK8sContext.crdClient, 0, tarsInformers.WithNamespace(glK8sContext.namespace))
	tframeworkconfigInformer := crdInformerFactory.Apps().V1beta3().TFrameworkConfigs()
	watcher := &Watcher{
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
				case *tarsAppsV1beta3.TFrameworkConfig:
					tfc := obj.(*tarsAppsV1beta3.TFrameworkConfig)
					if tfc.Name == tarsMeta.FixedTFrameworkConfigResourceName {
						setMaxReleases(tfc.RecordLimit.TImageRelease)
						setExecutor(tfc.ImageBuild.Executor.Image, tfc.ImageBuild.Executor.Secret)
						setRepository(tfc.ImageUpload.Registry, tfc.ImageUpload.Secret)
						setTagFormat(tfc.ImageBuild.TagFormat)
					}
				default:
					return
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				switch newObj.(type) {
				case *tarsAppsV1beta3.TFrameworkConfig:
					tfc := newObj.(*tarsAppsV1beta3.TFrameworkConfig)
					if tfc.Name == tarsMeta.FixedTFrameworkConfigResourceName {
						setMaxReleases(tfc.RecordLimit.TImageRelease)
						setExecutor(tfc.ImageBuild.Executor.Image, tfc.ImageBuild.Executor.Secret)
						setRepository(tfc.ImageUpload.Registry, tfc.ImageUpload.Secret)
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

func (w *Watcher) WaitSync(stopCh chan struct{}) bool {
	return cache.WaitForCacheSync(stopCh, w.synced...)
}

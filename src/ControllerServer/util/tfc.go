package util

import (
	"k8s.io/client-go/tools/cache"
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"sync"
)

var tfcMap sync.Map
var tfcInformerSynced cache.InformerSynced

func setupTFCWatch(factories *InformerFactories) {
	tfcInformer := factories.TarsInformerFactory.Crd().V1beta3().TFrameworkConfigs()
	tfcInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			tfc := obj.(*tarsCrdV1beta3.TFrameworkConfig)
			if tfc.Name == tarsMeta.FixedTFrameworkConfigResourceName {
				setTFrameworkConfig(tfc)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newTfc := newObj.(*tarsCrdV1beta3.TFrameworkConfig)
			oldTfc := oldObj.(*tarsCrdV1beta3.TFrameworkConfig)
			if newTfc.Name == tarsMeta.FixedTFrameworkConfigResourceName && newTfc.ResourceVersion != oldTfc.ResourceVersion {
				setTFrameworkConfig(newTfc)
			}
		},
	})
	tfcInformerSynced = tfcInformer.Informer().HasSynced
}

func GetTFrameworkConfig(namespace string) *tarsCrdV1beta3.TFrameworkConfig {
	tfc, _ := tfcMap.Load(namespace)
	if tfc != nil {
		return tfc.(*tarsCrdV1beta3.TFrameworkConfig)
	}
	return nil
}

func setTFrameworkConfig(tfc *tarsCrdV1beta3.TFrameworkConfig) {
	namespace := tfc.Namespace
	tfcCopy := tfc.DeepCopy()
	tfcMap.Store(namespace, tfcCopy)
}

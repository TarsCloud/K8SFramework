package runtime

import (
	"context"
	"fmt"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	tarsV1beta3 "k8s.tars.io/apis/tars/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"sync"
)

var tfcMap sync.Map
var tfcInformerSynced cache.InformerSynced

type TFrameworkConfig struct {
}

func (r *TFrameworkConfig) GetDefaultNodeImage(namespace string) (image string, secret string) {
	var tfc *tarsV1beta3.TFrameworkConfig
	if tfcInformerSynced() {
		if tfc = r.GetTFrameworkConfig(namespace); tfc != nil {
			return tfc.NodeImage.Image, tfc.NodeImage.Secret
		}

		utilRuntime.HandleError(fmt.Errorf("no default node image set"))
		return tarsMeta.ServiceImagePlaceholder, ""
	}

	tfc, _ = tarsClient.TarsV1beta3().TFrameworkConfigs(namespace).Get(context.TODO(), tarsMeta.FixedTFrameworkConfigResourceName, k8sMetaV1.GetOptions{})
	if tfc != nil {
		return tfc.NodeImage.Image, tfc.NodeImage.Secret
	}

	utilRuntime.HandleError(fmt.Errorf("no default node image set"))
	return tarsMeta.ServiceImagePlaceholder, ""
}

func (r *TFrameworkConfig) GetTFrameworkConfig(namespace string) *tarsV1beta3.TFrameworkConfig {
	tfc, _ := tfcMap.Load(namespace)
	if tfc != nil {
		return tfc.(*tarsV1beta3.TFrameworkConfig)
	}
	return nil
}

func (r *TFrameworkConfig) setTFrameworkConfig(tfc *tarsV1beta3.TFrameworkConfig) {
	namespace := tfc.Namespace
	tfcCopy := tfc.DeepCopy()
	tfcMap.Store(namespace, tfcCopy)
}

func (r *TFrameworkConfig) setupTFCWatch(factories *InformerFactories, namespace bool) {
	tfcInformer := factories.TarsInformerFactory.Tars().V1beta3().TFrameworkConfigs()
	tfcInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			tfc := obj.(*tarsV1beta3.TFrameworkConfig)
			if tfc.Name == tarsMeta.FixedTFrameworkConfigResourceName {
				r.setTFrameworkConfig(tfc)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newTfc := newObj.(*tarsV1beta3.TFrameworkConfig)
			oldTfc := oldObj.(*tarsV1beta3.TFrameworkConfig)
			if newTfc.Name == tarsMeta.FixedTFrameworkConfigResourceName && newTfc.ResourceVersion != oldTfc.ResourceVersion {
				r.setTFrameworkConfig(newTfc)
			}
		},
	})
	tfcInformerSynced = tfcInformer.Informer().HasSynced
}

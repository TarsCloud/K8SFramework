package runtime

import (
	"fmt"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sInformers "k8s.io/client-go/informers"
	"k8s.io/client-go/metadata/metadatainformer"
	tarsInformers "k8s.tars.io/client-go/informers/externalversions"
	tarsMeta "k8s.tars.io/meta"
)

type InformerFactories struct {
	K8SInformerFactory               k8sInformers.SharedInformerFactory
	K8SInformerFactoryWithTarsFilter k8sInformers.SharedInformerFactory
	MetadataInformerFactor           metadatainformer.SharedInformerFactory
	TarsInformerFactory              tarsInformers.SharedInformerFactory
}

func (i *InformerFactories) Start(stop chan struct{}) {
	i.K8SInformerFactory.Start(stop)
	i.K8SInformerFactoryWithTarsFilter.Start(stop)
	i.MetadataInformerFactor.Start(stop)
	i.TarsInformerFactory.Start(stop)
}

func newInformerFactories(clients *Client, cluster bool) *InformerFactories {

	if cluster {
		return &InformerFactories{
			K8SInformerFactory: k8sInformers.NewSharedInformerFactory(clients.K8sClient, 0),

			K8SInformerFactoryWithTarsFilter: k8sInformers.NewSharedInformerFactoryWithOptions(clients.K8sClient, 0, k8sInformers.WithTweakListOptions(
				func(options *k8sMetaV1.ListOptions) {
					options.LabelSelector = fmt.Sprintf("%s,%s", tarsMeta.TServerAppLabel, tarsMeta.TServerNameLabel)
				})),

			MetadataInformerFactor: metadatainformer.NewSharedInformerFactory(clients.K8sMetadataClient, 0),

			TarsInformerFactory: tarsInformers.NewSharedInformerFactoryWithOptions(clients.CrdClient, 0),
		}
	}

	return &InformerFactories{
		K8SInformerFactory: k8sInformers.NewSharedInformerFactoryWithOptions(clients.K8sClient, 0, k8sInformers.WithNamespace(Namespace)),

		K8SInformerFactoryWithTarsFilter: k8sInformers.NewSharedInformerFactoryWithOptions(clients.K8sClient, 0, k8sInformers.WithTweakListOptions(
			func(options *k8sMetaV1.ListOptions) {
				options.LabelSelector = fmt.Sprintf("%s,%s", tarsMeta.TServerAppLabel, tarsMeta.TServerNameLabel)
			},
		), k8sInformers.WithNamespace(Namespace)),

		MetadataInformerFactor: metadatainformer.NewFilteredSharedInformerFactory(clients.K8sMetadataClient, 0, Namespace, nil),

		TarsInformerFactory: tarsInformers.NewSharedInformerFactoryWithOptions(clients.CrdClient, 0, tarsInformers.WithNamespace(Namespace)),
	}
}

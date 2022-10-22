package informer

import (
	"k8s.io/client-go/tools/cache"
	tarsCrdListerV1beta3 "k8s.tars.io/client-go/listers/crd/v1beta3"
)

type Listers struct {
	TSLister tarsCrdListerV1beta3.TServerLister
	TSSynced cache.InformerSynced

	TTLister cache.GenericLister
	TTSynced cache.InformerSynced

	TCLister cache.GenericLister
	TCSynced cache.InformerSynced

	TRLister cache.GenericLister
	TRSynced cache.InformerSynced
}

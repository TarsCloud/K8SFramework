package lister

import (
	"k8s.io/client-go/tools/cache"
	tarsListerV1beta3 "k8s.tars.io/client-go/listers/tars/v1beta3"
)

type Listers struct {
	TSLister tarsListerV1beta3.TServerLister
	TSSynced cache.InformerSynced

	TILister tarsListerV1beta3.TImageLister
	TISynced cache.InformerSynced

	TTLister cache.GenericLister
	TTSynced cache.InformerSynced

	TCLister cache.GenericLister
	TCSynced cache.InformerSynced

	TRLister cache.GenericLister
	TRSynced cache.InformerSynced
}

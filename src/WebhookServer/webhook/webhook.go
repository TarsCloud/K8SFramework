package webhook

import (
	"fmt"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	tarsAppsV1beta3 "k8s.tars.io/apps/v1beta3"
	tarsRuntime "k8s.tars.io/runtime"
	"net/http"
	"tarswebhook/webhook/conversion"
	"tarswebhook/webhook/informer"
	"tarswebhook/webhook/mutating"
	"tarswebhook/webhook/validating"
	"time"
)

const CertFile = "/etc/tarswebhook-cert/tls.crt"
const CertKey = "/etc/tarswebhook-cert/tls.key"

type Webhook struct {
	mutating   *mutating.Mutating
	validating *validating.Validating
	conversion *conversion.Conversion
	listers    *informer.Listers
}

func New() *Webhook {
	tsInformer := tarsRuntime.Factories.TarsInformerFactory.Apps().V1beta3().TServers()
	ttInformer := tarsRuntime.Factories.MetadataInformerFactor.ForResource(tarsAppsV1beta3.SchemeGroupVersion.WithResource("ttemplates"))
	tcInformer := tarsRuntime.Factories.MetadataInformerFactor.ForResource(tarsAppsV1beta3.SchemeGroupVersion.WithResource("tconfigs"))
	trInformer := tarsRuntime.Factories.MetadataInformerFactor.ForResource(tarsAppsV1beta3.SchemeGroupVersion.WithResource("ttrees"))

	listers := &informer.Listers{
		TSLister: tsInformer.Lister(),
		TSSynced: tsInformer.Informer().HasSynced,

		TTLister: ttInformer.Lister(),
		TTSynced: ttInformer.Informer().HasSynced,

		TCLister: tcInformer.Lister(),
		TCSynced: tcInformer.Informer().HasSynced,

		TRLister: trInformer.Lister(),
		TRSynced: trInformer.Informer().HasSynced,
	}

	webhook := &Webhook{
		conversion: conversion.New(),
		mutating:   mutating.New(listers),
		validating: validating.New(listers),
		listers:    listers,
	}

	return webhook
}

func (h *Webhook) Run(stopCh chan struct{}) {

	cache.WaitForCacheSync(stopCh, h.listers.TSSynced, h.listers.TCSynced, h.listers.TRSynced)

	wait.Until(func() {
		validatingFunc := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			w.Header().Add("Connection", "keep-alive")
			h.validating.Handle(w, r)
		}

		mutatingFunc := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			w.Header().Add("Connection", "keep-alive")
			h.mutating.Handle(w, r)
		}

		conversionFunc := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			w.Header().Add("Connection", "keep-alive")
			h.conversion.Handle(w, r)
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/validating", validatingFunc)
		mux.HandleFunc("/mutating", mutatingFunc)
		mux.HandleFunc("/conversion", conversionFunc)

		srv := &http.Server{
			Addr:              ":443",
			Handler:           mux,
			ReadTimeout:       5 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      12 * time.Second,
		}
		// ListenAndServe always returns a non-nil error. After Shutdown or Close,
		// the returned error is ErrServerClosed.
		err := srv.ListenAndServeTLS(CertFile, CertKey)
		if err != nil {
			utilRuntime.HandleError(fmt.Errorf("will exist because : %s \n", err.Error()))
			close(stopCh)
		}
	}, time.Second, stopCh)
}

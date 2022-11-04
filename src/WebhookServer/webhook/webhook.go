package webhook

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	tarsAppsV1beta3 "k8s.tars.io/apps/v1beta3"
	tarsRuntime "k8s.tars.io/runtime"
	"math/big"
	"net/http"
	"os"
	"tarswebhook/webhook/conversion"
	"tarswebhook/webhook/informer"
	"tarswebhook/webhook/mutating"
	"tarswebhook/webhook/validating"
	"time"
)

const CaCrtFile = "/etc/tarswebhook-cert/ca.crt"
const CaKeyFile = "/etc/tarswebhook-cert/ca.key"
const ServiceName = "tars-webhook-service"

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

func selfSigneCert() (*tls.Certificate, error) {

	caCrtContent, err := ioutil.ReadFile(CaCrtFile)
	if err != nil {
		return nil, fmt.Errorf("read %s file error: %s", CaCrtFile, err.Error())
	}

	caCrtBlock, _ := pem.Decode(caCrtContent)
	caCrt, err := x509.ParseCertificate(caCrtBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parser %s content error: %s", CaCrtFile, err.Error())
	}

	caKeyContent, err := ioutil.ReadFile(CaKeyFile)
	if err != nil {
		return nil, fmt.Errorf("read %s file error: %s", CaKeyFile, err.Error())
	}

	caKeyBlock, _ := pem.Decode(caKeyContent)
	if caKeyBlock.Type != "RSA PRIVATE KEY" {
		if err != nil {
			return nil, fmt.Errorf("parser %s content error: %s", CaKeyFile, err.Error())
		}
	}

	caKey, err := x509.ParsePKCS1PrivateKey(caKeyBlock.Bytes)
	if err != nil {
		if err != nil {
			return nil, fmt.Errorf("parser %s content error: %s", CaKeyFile, err.Error())
		}
	}

	commonName := fmt.Sprintf("%s.%s.svc", ServiceName, tarsRuntime.Namespace)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(2022),
		Subject: pkix.Name{
			Organization: []string{"WebhookService"},
			CommonName:   commonName,
		},
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(1, 1, 1),
		IsCA:        false,
		DNSNames:    []string{commonName},
	}

	certKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	certEncode, err := x509.CreateCertificate(rand.Reader, template, caCrt, certKey.Public(), caKey)
	if err != nil {
		return nil, fmt.Errorf("create certificate error: %s", err.Error())
	}

	var out tls.Certificate
	out.Certificate = append(out.Certificate, certEncode)
	out.PrivateKey = certKey

	return &out, nil
}

func (h *Webhook) Run(stopCh chan struct{}) {
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

		cert, err := selfSigneCert()
		if err != nil {
			utilRuntime.HandleError(fmt.Errorf("will exist because: %s \n", err.Error()))
			os.Exit(-1)
		}

		srv := &http.Server{
			Addr:    ":443",
			Handler: mux,
			TLSConfig: &tls.Config{
				NextProtos:   []string{"http/1.1"},
				Certificates: []tls.Certificate{*cert},
			},
			ReadTimeout:       5 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      12 * time.Second,
		}
		// ListenAndServe always returns a non-nil error. After Shutdown or Close,
		// the returned error is ErrServerClosed.
		err = srv.ListenAndServeTLS("", "")
		if err != nil {
			utilRuntime.HandleError(fmt.Errorf("will exist because: %s \n", err.Error()))
		}
	}, time.Second, stopCh)
}

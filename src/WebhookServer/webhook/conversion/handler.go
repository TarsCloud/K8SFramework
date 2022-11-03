package conversion

import (
	"encoding/json"
	"fmt"
	k8sExtensionsV1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	tarsMeta "k8s.tars.io/meta"
	"net/http"
	conversionAppsV1beta2 "tarswebhook/webhook/conversion/v1beta2"
	conversionAppsV1beta3 "tarswebhook/webhook/conversion/v1beta3"
)

type Conversion struct {
}

func New() *Conversion {
	c := &Conversion{}
	return c
}

func (*Conversion) Handle(w http.ResponseWriter, r *http.Request) {
	requestConversionView := &k8sExtensionsV1.ConversionReview{}
	err := json.NewDecoder(r.Body).Decode(&requestConversionView)
	if err != nil {
		return
	}

	v, _ := extractAPIVersion(requestConversionView.Request.Objects[0])

	key := generateKey(v.Kind, v.APIVersion, requestConversionView.Request.DesiredAPIVersion)
	conversion, ok := conversionFunctions[key]
	if !ok {
		return
	}

	responseConversionView := &k8sExtensionsV1.ConversionReview{
		TypeMeta: k8sMetaV1.TypeMeta{
			Kind:       requestConversionView.Kind,
			APIVersion: requestConversionView.APIVersion,
		},
		Response: &k8sExtensionsV1.ConversionResponse{
			UID: requestConversionView.Request.UID,
			Result: k8sMetaV1.Status{
				Status: "Success",
			},
			ConvertedObjects: conversion(requestConversionView.Request.Objects),
		},
	}

	responseBytes, _ := json.Marshal(responseConversionView)
	_, _ = w.Write(responseBytes)
}

func extractAPIVersion(in runtime.RawExtension) (*k8sMetaV1.TypeMeta, error) {
	var typeMeta = &k8sMetaV1.TypeMeta{}
	if err := json.Unmarshal(in.Raw, typeMeta); err != nil {
		return nil, err
	}
	return typeMeta, nil
}

var conversionFunctions map[string]func([]runtime.RawExtension) []runtime.RawExtension

func generateKey(kind string, fromGV, toGV string) string {
	return fmt.Sprintf("%s/%s-%s", kind, fromGV, toGV)
}

func registry(kind string, fromGV, toGV string, conversion func([]runtime.RawExtension) []runtime.RawExtension) {
	if conversionFunctions == nil {
		conversionFunctions = map[string]func([]runtime.RawExtension) []runtime.RawExtension{}
	}
	key := generateKey(kind, fromGV, toGV)
	conversionFunctions[key] = conversion
}

func init() {

	registry(tarsMeta.TServerKind, tarsMeta.TarsGroupVersionV1B1, tarsMeta.TarsGroupVersionV1B2, conversionAppsV1beta2.CvTServer1b1To1b2)
	registry(tarsMeta.TServerKind, tarsMeta.TarsGroupVersionV1B2, tarsMeta.TarsGroupVersionV1B1, conversionAppsV1beta2.CvTServer1b2To1b1)

	registry(tarsMeta.TServerKind, tarsMeta.TarsGroupVersionV1B1, tarsMeta.TarsGroupVersionV1B3, conversionAppsV1beta3.CvTServer1b1To1b3)
	registry(tarsMeta.TServerKind, tarsMeta.TarsGroupVersionV1B2, tarsMeta.TarsGroupVersionV1B3, conversionAppsV1beta3.CvTServer1b2To1b3)

	registry(tarsMeta.TServerKind, tarsMeta.TarsGroupVersionV1B3, tarsMeta.TarsGroupVersionV1B1, conversionAppsV1beta3.CvTServer1b3To1b1)
	registry(tarsMeta.TServerKind, tarsMeta.TarsGroupVersionV1B3, tarsMeta.TarsGroupVersionV1B2, conversionAppsV1beta3.CvTServer1b3To1b2)

	registry(tarsMeta.TFrameworkConfigKind, tarsMeta.TarsGroupVersionV1B2, tarsMeta.TarsGroupVersionV1B3, conversionAppsV1beta3.CvTFC1b2To1b3)
	registry(tarsMeta.TFrameworkConfigKind, tarsMeta.TarsGroupVersionV1B3, tarsMeta.TarsGroupVersionV1B2, conversionAppsV1beta3.CvTFC1b3To1b2)
}

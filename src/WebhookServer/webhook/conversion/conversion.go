package conversion

import (
	"encoding/json"
	"fmt"
	k8sExtensionsV1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	"net/http"
)

type Converter func([]runtime.RawExtension) []runtime.RawExtension

var conversionFunctions = map[string]Converter{}

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

func generateKey(kind, from, to string) string {
	return fmt.Sprintf("FROM %s TO %s, Kind=%s", from, to, kind)
}

func Registry(kind string, fromGV, toGV schema.GroupVersion, converter Converter) {
	from := fromGV.String()
	to := toGV.String()
	key := generateKey(kind, from, to)
	klog.Infof("registry conversion key [%s]", key)
	conversionFunctions[key] = converter
}

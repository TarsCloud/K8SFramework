package v1beta3

import (
	"fmt"
	"hash/crc32"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	tarsV1beta3 "k8s.tars.io/apis/tars/v1beta3"
	tarsTool "k8s.tars.io/tool"
	"tarswebhook/webhook/lister"
	"tarswebhook/webhook/mutating"
	"time"
)

func mutatingCreateTConfig(listers *lister.Listers, requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	tconfig := &tarsV1beta3.TConfig{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, tconfig)

	var jsonPatch tarsTool.JsonPatch

	if tconfig.Labels == nil {
		labels := map[string]string{}
		jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
			OP:    tarsTool.JsonPatchAdd,
			Path:  "/metadata/labels",
			Value: labels,
		})
	}

	jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
		OP:    tarsTool.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ServerApp",
		Value: tconfig.App,
	})

	jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
		OP:    tarsTool.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ServerName",
		Value: tconfig.Server,
	})

	jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
		OP:    tarsTool.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ConfigName",
		Value: tconfig.ConfigName,
	})

	jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
		OP:    tarsTool.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1PodSeq",
		Value: tconfig.PodSeq,
	})

	jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
		OP:    tarsTool.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1Activated",
		Value: fmt.Sprintf("%t", tconfig.Activated),
	})

	versionString := fmt.Sprintf("%s-%x", time.Now().Format("20060102030405"), crc32.ChecksumIEEE([]byte(tconfig.Name)))
	jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
		OP:    tarsTool.JsonPatchAdd,
		Path:  "/version",
		Value: versionString,
	})

	jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
		OP:    tarsTool.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1Version",
		Value: versionString,
	})

	jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
		OP:    tarsTool.JsonPatchAdd,
		Path:  "/updateTime",
		Value: k8sMetaV1.Now().ToUnstructured(),
	})

	if jsonPatch != nil {
		return json.Marshal(jsonPatch)
	}
	return nil, nil
}

func mutatingUpdateTConfig(listers *lister.Listers, requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	tconfig := &tarsV1beta3.TConfig{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, tconfig)

	var jsonPatch tarsTool.JsonPatch

	if tconfig.Labels == nil {
		labels := map[string]string{}
		jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
			OP:    tarsTool.JsonPatchAdd,
			Path:  "/metadata/labels",
			Value: labels,
		})
	}

	jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
		OP:    tarsTool.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ServerApp",
		Value: tconfig.App,
	})

	jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
		OP:    tarsTool.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ServerName",
		Value: tconfig.Server,
	})

	jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
		OP:    tarsTool.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ConfigName",
		Value: tconfig.ConfigName,
	})

	jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
		OP:    tarsTool.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1PodSeq",
		Value: tconfig.PodSeq,
	})

	jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
		OP:    tarsTool.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1Activated",
		Value: fmt.Sprintf("%t", tconfig.Activated),
	})

	jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
		OP:    tarsTool.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1Version",
		Value: tconfig.Version,
	})

	if jsonPatch != nil {
		return json.Marshal(jsonPatch)
	}
	return nil, nil
}

func init() {
	gvr := tarsV1beta3.SchemeGroupVersion.WithResource("tconfigs")
	mutating.Registry(k8sAdmissionV1.Create, &gvr, mutatingCreateTConfig)
	mutating.Registry(k8sAdmissionV1.Update, &gvr, mutatingUpdateTConfig)
}

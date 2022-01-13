package conversion

import (
	"encoding/json"
	"fmt"
	k8sExtensionsV1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	crdV1beta1 "k8s.tars.io/api/crd/v1beta1"
	crdV1beta2 "k8s.tars.io/api/crd/v1beta2"
	crdMeta "k8s.tars.io/api/meta"
	"net/http"
	"tarscontroller/controller"
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
			ConvertedObjects: conversionFunctions[v.Kind][v.APIVersion][requestConversionView.Request.DesiredAPIVersion](requestConversionView.Request.Objects),
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

// map[Kind]map[FromGV]map[ToGV]func([]runtime.RawExtension) []runtime.RawExtension
var conversionFunctions map[string]map[string]map[string]func([]runtime.RawExtension) []runtime.RawExtension

func _cvTServer1a2To1b2(src *crdV1beta1.TServer) (dst *crdV1beta2.TServer) {
	dst = &crdV1beta2.TServer{
		TypeMeta: k8sMetaV1.TypeMeta{
			APIVersion: "k8s.tars.io/v1beta2",
			Kind:       crdMeta.TServerKind,
		},
		ObjectMeta: src.ObjectMeta,
	}

	bs, _ := json.Marshal(src.Spec)
	_ = json.Unmarshal(bs, &dst.Spec)

	dst.Status = crdV1beta2.TServerStatus(src.Status)

	var conversionAnnotation string
	if src.ObjectMeta.Annotations != nil {
		conversionAnnotation, _ = src.ObjectMeta.Annotations[crdMeta.TConversionAnnotationPrefix+"."+"1a21b2"]
	}

	if conversionAnnotation != "" {
		var diff = crdMeta.TServerConversion1a21b2{}
		err := json.Unmarshal([]byte(conversionAnnotation), &diff)
		if err == nil {
			dst.Spec.K8S.UpdateStrategy = diff.Append.UpdateStrategy
			dst.Spec.K8S.ImagePullPolicy = diff.Append.ImagePullPolicy
			dst.Spec.K8S.LauncherType = diff.Append.LauncherType
			if diff.Append.TServerReleaseNode != nil {
				dst.Spec.Release.TServerReleaseNode = diff.Append.TServerReleaseNode
			}
			return dst
		}
		utilRuntime.HandleError(fmt.Errorf("read conversion annotation error: %s", err.Error()))
	}

	dst.Spec.K8S.UpdateStrategy = crdMeta.DefaultStatefulsetUpdateStrategy
	dst.Spec.K8S.ImagePullPolicy = crdMeta.DefaultImagePullPolicy
	dst.Spec.K8S.LauncherType = crdMeta.DefaultLauncherType
	if dst.Spec.Tars != nil && dst.Spec.Release != nil {
		image, secret := controller.GetDefaultNodeImage(src.Namespace)
		dst.Spec.Release.TServerReleaseNode = &crdV1beta2.TServerReleaseNode{
			Image:  image,
			Secret: secret,
		}
	}
	return dst
}

func cvTServer1a2To1b2(s []runtime.RawExtension) []runtime.RawExtension {
	d := make([]runtime.RawExtension, len(s), len(s))
	for i := range s {
		var src = &crdV1beta1.TServer{}
		_ = json.Unmarshal(s[i].Raw, src)
		dst := _cvTServer1a2To1b2(src)
		d[i].Raw, _ = json.Marshal(dst)
	}
	return d
}

func _cvTServer1b2To1a2(src *crdV1beta2.TServer) (dst *crdV1beta1.TServer) {
	dst = &crdV1beta1.TServer{
		TypeMeta: k8sMetaV1.TypeMeta{
			APIVersion: "k8s.tars.io/v1beta1",
			Kind:       crdMeta.TServerKind,
		},
		ObjectMeta: src.ObjectMeta,
	}

	bs, _ := json.Marshal(src.Spec)
	_ = json.Unmarshal(bs, &dst.Spec)

	dst.Status = crdV1beta1.TServerStatus(src.Status)

	diff := crdMeta.TServerConversion1a21b2{
		Append: crdMeta.TServerAppend1a21b2{
			UpdateStrategy:  src.Spec.K8S.UpdateStrategy,
			ImagePullPolicy: src.Spec.K8S.ImagePullPolicy,
			LauncherType:    src.Spec.K8S.LauncherType,
		},
	}
	if dst.Spec.Tars != nil && dst.Spec.Release != nil {
		diff.Append.TServerReleaseNode = src.Spec.Release.TServerReleaseNode
	}

	dbs, _ := json.Marshal(diff)
	if dst.Annotations == nil {
		dst.Annotations = map[string]string{
			crdMeta.TConversionAnnotationPrefix + "." + "1a21b2": string(dbs),
		}
	} else {
		dst.Annotations[crdMeta.TConversionAnnotationPrefix+"."+"1a21b2"] = string(dbs)
	}
	return dst
}

func cvTServer1b2To1a2(s []runtime.RawExtension) []runtime.RawExtension {
	d := make([]runtime.RawExtension, len(s), len(s))
	for i := range s {
		var src = &crdV1beta2.TServer{}
		_ = json.Unmarshal(s[i].Raw, src)
		dst := _cvTServer1b2To1a2(src)
		d[i].Raw, _ = json.Marshal(dst)
	}
	return d
}

func cvTDeploy1a2To1b2(s []runtime.RawExtension) []runtime.RawExtension {
	d := make([]runtime.RawExtension, len(s), len(s))

	for i := range s {
		var src = &crdV1beta1.TDeploy{}
		_ = json.Unmarshal(s[i].Raw, src)

		fakeTserver := &crdV1beta1.TServer{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Annotations: map[string]string{},
			},
			Spec: src.Apply,
		}
		tserver := _cvTServer1a2To1b2(fakeTserver)
		var dst = &crdV1beta2.TDeploy{
			TypeMeta: k8sMetaV1.TypeMeta{
				APIVersion: "k8s.tars.io/v1beta2",
				Kind:       "TDeploy",
			},
			ObjectMeta: src.ObjectMeta,
			Apply:      tserver.Spec,
			Approve:    (*crdV1beta2.TDeployApprove)(src.Approve),
			Deployed:   src.Deployed,
		}
		if dst.ObjectMeta.Annotations == nil {
			dst.ObjectMeta.Annotations = tserver.Annotations
		} else {
			for k, v := range tserver.ObjectMeta.Annotations {
				dst.ObjectMeta.Annotations[k] = v
			}
		}
		d[i].Raw, _ = json.Marshal(dst)
	}
	return d
}

func cvTDeploy1b2To1a2(s []runtime.RawExtension) []runtime.RawExtension {
	d := make([]runtime.RawExtension, len(s), len(s))

	for i := range s {
		var src = &crdV1beta2.TDeploy{}
		_ = json.Unmarshal(s[i].Raw, src)

		fakeTserver := &crdV1beta2.TServer{
			ObjectMeta: k8sMetaV1.ObjectMeta{
				Annotations: map[string]string{},
			},
			Spec: src.Apply,
		}
		tserver := _cvTServer1b2To1a2(fakeTserver)
		var dst = &crdV1beta1.TDeploy{
			TypeMeta: k8sMetaV1.TypeMeta{
				APIVersion: "k8s.tars.io/v1beta1",
				Kind:       "TDeploy",
			},
			ObjectMeta: src.ObjectMeta,
			Apply:      tserver.Spec,
			Approve:    (*crdV1beta1.TDeployApprove)(src.Approve),
			Deployed:   src.Deployed,
		}
		if dst.ObjectMeta.Annotations == nil {
			dst.ObjectMeta.Annotations = tserver.Annotations
		} else {
			for k, v := range tserver.ObjectMeta.Annotations {
				dst.ObjectMeta.Annotations[k] = v
			}
		}
		d[i].Raw, _ = json.Marshal(dst)
	}
	return d
}

func init() {
	conversionFunctions = map[string]map[string]map[string]func([]runtime.RawExtension) []runtime.RawExtension{
		"TServer": {
			"k8s.tars.io/v1beta1": {"k8s.tars.io/v1beta2": cvTServer1a2To1b2},
			"k8s.tars.io/v1beta2":  {"k8s.tars.io/v1beta1": cvTServer1b2To1a2},
		},
		"TDeploy": {
			"k8s.tars.io/v1beta1": {"k8s.tars.io/v1beta2": cvTDeploy1a2To1b2},
			"k8s.tars.io/v1beta2":  {"k8s.tars.io/v1beta1": cvTDeploy1b2To1a2},
		},
	}
}

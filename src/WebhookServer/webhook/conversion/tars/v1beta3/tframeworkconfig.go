package v1beta3

import (
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/klog/v2"
	tarsV1beta2 "k8s.tars.io/apis/tars/v1beta2"
	tarsV1beta3 "k8s.tars.io/apis/tars/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"tarswebhook/webhook/conversion"
)

func conversionUpChainV1b2ToV1b3(src map[string][]*tarsV1beta2.TFrameworkTarsEndpoint) (dst map[string][]*tarsV1beta3.TFrameworkTarsEndpoint) {
	if src == nil {
		return nil
	}
	dst = map[string][]*tarsV1beta3.TFrameworkTarsEndpoint{}
	for k, v := range src {
		var nv []*tarsV1beta3.TFrameworkTarsEndpoint
		for _, p := range v {
			nv = append(nv, (*tarsV1beta3.TFrameworkTarsEndpoint)(p))
		}
		dst[k] = nv
	}
	return dst
}

func CvTFC1b2To1b3(s []runtime.RawExtension) []runtime.RawExtension {
	d := make([]runtime.RawExtension, len(s), len(s))
	for i := range s {
		var src = &tarsV1beta2.TFrameworkConfig{}
		_ = json.Unmarshal(s[i].Raw, src)

		dst := &tarsV1beta3.TFrameworkConfig{
			TypeMeta: k8sMetaV1.TypeMeta{
				APIVersion: tarsMeta.TarsGroupVersionV1B1,
				Kind:       tarsMeta.TFrameworkConfigKind,
			},
			ObjectMeta: src.ObjectMeta,
			ImageBuild: tarsV1beta3.TFrameworkImageBuild{
				MaxBuildTime: src.ImageBuild.MaxBuildTime,
				TagFormat:    src.ImageBuild.TagFormat,
				Executor:     tarsV1beta3.TFrameworkImage{},
			},
			ImageUpload: tarsV1beta3.TFrameworkImageUpload{
				Registry: src.ImageRegistry.Registry,
				Secret:   src.ImageRegistry.Secret,
			},
			RecordLimit: tarsV1beta3.TFrameworkRecordLimit(src.RecordLimit),
			NodeImage:   tarsV1beta3.TFrameworkImage(src.NodeImage),
			UPChain:     conversionUpChainV1b2ToV1b3(src.UPChain),
			Expand:      src.Expand,
		}

		for ii := 0; ii < 1; ii++ {
			if src.ObjectMeta.Annotations == nil {
				break
			}
			conversionAnnotation, _ := src.ObjectMeta.Annotations[V1b2V1b3Annotation]
			if conversionAnnotation == "" {
				delete(dst.ObjectMeta.Annotations, V1b2V1b3Annotation)
				break
			}
			var diff = TFCConversion1b21b3{}
			err := json.Unmarshal([]byte(conversionAnnotation), &diff)
			if err != nil {
				klog.Errorf("read conversion annotation error: %s", err.Error())
				delete(dst.ObjectMeta.Annotations, V1b2V1b3Annotation)
				break
			}
			dst.ImageBuild.Executor = diff.Append.Executor
		}
		d[i].Raw, _ = json.Marshal(dst)
	}
	return d
}

func conversionUpChainV1b3ToV1b2(src map[string][]*tarsV1beta3.TFrameworkTarsEndpoint) (dst map[string][]*tarsV1beta2.TFrameworkTarsEndpoint) {
	if src == nil {
		return nil
	}
	dst = map[string][]*tarsV1beta2.TFrameworkTarsEndpoint{}
	for k, v := range src {
		var nv []*tarsV1beta2.TFrameworkTarsEndpoint
		for _, p := range v {
			nv = append(nv, (*tarsV1beta2.TFrameworkTarsEndpoint)(p))
		}
		dst[k] = nv
	}
	return dst
}

func CvTFC1b3To1b2(s []runtime.RawExtension) []runtime.RawExtension {
	d := make([]runtime.RawExtension, len(s), len(s))
	for i := range s {
		var src = &tarsV1beta3.TFrameworkConfig{}
		_ = json.Unmarshal(s[i].Raw, src)

		dst := &tarsV1beta2.TFrameworkConfig{
			TypeMeta: k8sMetaV1.TypeMeta{
				APIVersion: tarsMeta.TarsGroupVersionV1B2,
				Kind:       tarsMeta.TFrameworkConfigKind,
			},
			ObjectMeta: src.ObjectMeta,
			ImageBuild: tarsV1beta2.TFrameworkImageBuild{
				MaxBuildTime: src.ImageBuild.MaxBuildTime,
				TagFormat:    src.ImageBuild.TagFormat,
			},
			ImageRegistry: tarsV1beta2.TFrameworkImageRegistry{
				Registry: src.ImageUpload.Registry,
				Secret:   src.ImageUpload.Secret,
			},
			RecordLimit: tarsV1beta2.TFrameworkRecordLimit(src.RecordLimit),
			NodeImage:   tarsV1beta2.TFrameworkNodeImage(src.NodeImage),
			UPChain:     conversionUpChainV1b3ToV1b2(src.UPChain),
			Expand:      src.Expand,
		}

		diff := TFCConversion1b21b3{
			Append: TFCAppend1b21b3{
				Executor: src.ImageBuild.Executor,
			},
		}
		bs, _ := json.Marshal(diff)
		if dst.Annotations == nil {
			dst.Annotations = map[string]string{}
		}
		dst.Annotations[V1b2V1b3Annotation] = string(bs)
		d[i].Raw, _ = json.Marshal(dst)
	}
	return d
}

func init() {
	conversion.Registry(tarsMeta.TFrameworkConfigKind, tarsV1beta3.SchemeGroupVersion, tarsV1beta2.SchemeGroupVersion, CvTFC1b3To1b2)
	conversion.Registry(tarsMeta.TFrameworkConfigKind, tarsV1beta2.SchemeGroupVersion, tarsV1beta3.SchemeGroupVersion, CvTFC1b2To1b3)
}

package e2e

import (
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
)

func ObjLayoutToString(obj k8sMetaV1.Object, namespace string) string {
	obj.SetNamespace(namespace)
	bs, _ := json.Marshal(obj)
	return string(bs)
}

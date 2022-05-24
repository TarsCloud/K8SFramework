package v1beta3

import (
	"crypto/sha1"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"hash/crc32"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/integer"
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
	tarsMetaTools "k8s.tars.io/meta/tools"
	tarsMetaV1beta3 "k8s.tars.io/meta/v1beta3"
	"math"
	"regexp"
	"strconv"
	"strings"
	"tarscontroller/controller"
	"time"
)

var functions map[string]func(*k8sAdmissionV1.AdmissionReview) ([]byte, error)

func init() {
	functions = map[string]func(*k8sAdmissionV1.AdmissionReview) ([]byte, error){

		"CREATE/TServer": mutatingCreateTServer,
		"UPDATE/TServer": mutatingUpdateTServer,

		"CREATE/TConfig": mutatingCreateTConfig,
		"UPDATE/TConfig": mutatingUpdateTConfig,

		"CREATE/TTree": mutatingCreateTTree,
		"UPDATE/TTree": mutatingUpdateTTree,

		"CREATE/TAccount": mutatingCreateTAccount,
		"UPDATE/TAccount": mutatingUpdateTAccount,

		"CREATE/TImage": mutatingCreateTImage,
		"UPDATE/TImage": mutatingUpdateTImage,

		"CREATE/TTemplate": mutatingCreateTTemplate,
		"UPDATE/TTemplate": mutatingUpdateTTemplate,
	}

}

func Handle(clients *controller.Clients, informer *controller.Informers, view *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	key := fmt.Sprintf("%s/%s", string(view.Request.Operation), view.Request.Kind.Kind)
	if fun, ok := functions[key]; ok {
		return fun(view)
	}
	return nil, fmt.Errorf("unsupported mutating %s %s.%s", view.Request.Operation, view.Request.Kind.Version, view.Request.Kind.Kind)
}

func mutatingCreateTServer(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	tserver := &tarsCrdV1beta3.TServer{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, tserver)

	var jsonPatch tarsMetaTools.JsonPatch

	if tserver.Labels == nil {
		labels := map[string]string{}
		jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
			OP:    tarsMetaTools.JsonPatchAdd,
			Path:  "/metadata/labels",
			Value: labels,
		})
	}

	jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
		OP:    tarsMetaTools.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ServerApp",
		Value: tserver.Spec.App,
	})

	jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
		OP:    tarsMetaTools.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ServerName",
		Value: tserver.Spec.Server,
	})

	jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
		OP:    tarsMetaTools.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1SubType",
		Value: string(tserver.Spec.SubType),
	})

	if tserver.Spec.Tars != nil {
		jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
			OP:    tarsMetaTools.JsonPatchAdd,
			Path:  "/metadata/labels/tars.io~1Template",
			Value: tserver.Spec.Tars.Template,
		})

		gatesMap := map[string]interface{}{}
		gatesArray := []string{tarsMetaV1beta3.TPodReadinessGate}
		for _, v := range tserver.Spec.K8S.ReadinessGates {
			if v != tarsMetaV1beta3.TPodReadinessGate {
				if _, ok := gatesMap[v]; !ok {
					gatesArray = append(gatesArray, v)
					gatesMap[v] = nil
				}
			}
		}

		jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
			OP:    tarsMetaTools.JsonPatchAdd,
			Path:  "/spec/k8s/readinessGates",
			Value: gatesArray,
		})
	}

	if tserver.Spec.Normal != nil {
		if _, ok := tserver.Labels[tarsMetaV1beta3.TemplateLabel]; ok {
			jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
				OP:   tarsMetaTools.JsonPatchRemove,
				Path: "/metadata/labels/tars.io~1Template",
			})
		}

		if len(tserver.Spec.K8S.ReadinessGates) > 0 {
			gatesMap := map[string]interface{}{}
			var gatesArray []string

			for _, v := range tserver.Spec.K8S.ReadinessGates {
				if _, ok := gatesMap[v]; !ok {
					gatesArray = append(gatesArray, v)
					gatesMap[v] = nil
				}
			}

			jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
				OP:    tarsMetaTools.JsonPatchAdd,
				Path:  "/spec/k8s/readinessGates",
				Value: gatesArray,
			})
		}
	}

	if len(tserver.Spec.K8S.HostPorts) > 0 || tserver.Spec.K8S.HostIPC {
		jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
			OP:    tarsMetaTools.JsonPatchAdd,
			Path:  "/spec/k8s/notStacked",
			Value: true,
		})
	}

	maxReplicasValue := math.MaxInt32
	minReplicasValue := math.MinInt32

	if tserver.Annotations != nil {
		const pattern = "^[1-9]?[0-9]$"
		if maxReplicas, ok := tserver.Annotations[tarsMetaV1beta3.TMaxReplicasAnnotation]; ok {
			matched, _ := regexp.MatchString(pattern, maxReplicas)
			if !matched {
				return nil, fmt.Errorf(tarsMetaV1beta3.ResourceInvalidError, "tserver", "unexpected annotation format")
			}
			maxReplicasValue, _ = strconv.Atoi(maxReplicas)
		}

		if minReplicas, ok := tserver.Annotations[tarsMetaV1beta3.TMinReplicasAnnotation]; ok {
			matched, _ := regexp.MatchString(pattern, minReplicas)
			if !matched {
				return nil, fmt.Errorf(tarsMetaV1beta3.ResourceInvalidError, "tserver", "unexpected annotation format")
			}
			minReplicasValue, _ = strconv.Atoi(minReplicas)
		}

		if minReplicasValue > maxReplicasValue {
			return nil, fmt.Errorf(tarsMetaV1beta3.ResourceInvalidError, "tserver", "unexpected annotation value")
		}
	}

	if tserver.Spec.Release == nil {
		jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
			OP:    tarsMetaTools.JsonPatchReplace,
			Path:  "/spec/k8s/replicas",
			Value: 0,
		})
	} else {
		replicas := int(tserver.Spec.K8S.Replicas)
		replicas = integer.IntMax(replicas, minReplicasValue)
		replicas = integer.IntMin(replicas, maxReplicasValue)

		jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
			OP:    tarsMetaTools.JsonPatchReplace,
			Path:  "/spec/k8s/replicas",
			Value: replicas,
		})

		if tserver.Spec.Release.Time.IsZero() {
			now := k8sMetaV1.Now()
			jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
				OP:    tarsMetaTools.JsonPatchAdd,
				Path:  "/spec/release/time",
				Value: now.ToUnstructured(),
			})
		}

		jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
			OP:    tarsMetaTools.JsonPatchAdd,
			Path:  "/metadata/labels/tars.io~1ServerID",
			Value: tserver.Spec.Release.ID,
		})

		if tserver.Spec.Tars != nil {
			if tserver.Spec.Release.TServerReleaseNode == nil || tserver.Spec.Release.TServerReleaseNode.Image == "" {
				image, secret := controller.GetDefaultNodeImage(tserver.Namespace)
				if image == tarsMetaV1beta3.ServiceImagePlaceholder {
					return nil, fmt.Errorf(tarsMetaV1beta3.ResourceInvalidError, tserver, "no default node image has been set")
				}

				jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
					OP:    tarsMetaTools.JsonPatchAdd,
					Path:  "/spec/release/nodeImage",
					Value: image,
				})

				jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
					OP:    tarsMetaTools.JsonPatchAdd,
					Path:  "/spec/release/nodeSecret",
					Value: secret,
				})
			}
		}

		if tserver.Spec.Normal != nil {
			if tserver.Spec.Release.TServerReleaseNode != nil {
				if tserver.Spec.Release.TServerReleaseNode.Image != "" {
					jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
						OP:   tarsMetaTools.JsonPatchRemove,
						Path: "/spec/release/nodeImage",
					})
				}

				if tserver.Spec.Release.TServerReleaseNode.Secret != "" {
					jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
						OP:   tarsMetaTools.JsonPatchRemove,
						Path: "/spec/release/nodeSecret",
					})
				}
			}
		}
	}

	if jsonPatch != nil {
		return json.Marshal(jsonPatch)
	}
	return nil, nil
}

func mutatingUpdateTServer(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	return mutatingCreateTServer(requestAdmissionView)
}

func mutatingCreateTConfig(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	tconfig := &tarsCrdV1beta3.TConfig{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, tconfig)

	var jsonPatch tarsMetaTools.JsonPatch

	if tconfig.Labels == nil {
		labels := map[string]string{}
		jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
			OP:    tarsMetaTools.JsonPatchAdd,
			Path:  "/metadata/labels",
			Value: labels,
		})
	}

	jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
		OP:    tarsMetaTools.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ServerApp",
		Value: tconfig.App,
	})

	jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
		OP:    tarsMetaTools.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ServerName",
		Value: tconfig.Server,
	})

	jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
		OP:    tarsMetaTools.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ConfigName",
		Value: tconfig.ConfigName,
	})

	jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
		OP:    tarsMetaTools.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1PodSeq",
		Value: tconfig.PodSeq,
	})

	jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
		OP:    tarsMetaTools.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1Activated",
		Value: fmt.Sprintf("%t", tconfig.Activated),
	})

	versionString := fmt.Sprintf("%s-%x", time.Now().Format("20060102030405"), crc32.ChecksumIEEE([]byte(tconfig.Name)))
	jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
		OP:    tarsMetaTools.JsonPatchAdd,
		Path:  "/version",
		Value: versionString,
	})

	jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
		OP:    tarsMetaTools.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1Version",
		Value: versionString,
	})

	if jsonPatch != nil {
		return json.Marshal(jsonPatch)
	}
	return nil, nil
}

func mutatingUpdateTConfig(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	tconfig := &tarsCrdV1beta3.TConfig{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, tconfig)

	var jsonPatch tarsMetaTools.JsonPatch

	if tconfig.Labels == nil {
		labels := map[string]string{}
		jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
			OP:    tarsMetaTools.JsonPatchAdd,
			Path:  "/metadata/labels",
			Value: labels,
		})
	}

	jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
		OP:    tarsMetaTools.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ServerApp",
		Value: tconfig.App,
	})

	jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
		OP:    tarsMetaTools.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ServerName",
		Value: tconfig.Server,
	})

	jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
		OP:    tarsMetaTools.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ConfigName",
		Value: tconfig.ConfigName,
	})

	jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
		OP:    tarsMetaTools.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1PodSeq",
		Value: tconfig.PodSeq,
	})

	jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
		OP:    tarsMetaTools.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1Activated",
		Value: fmt.Sprintf("%t", tconfig.Activated),
	})

	jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
		OP:    tarsMetaTools.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1Version",
		Value: tconfig.Version,
	})

	if jsonPatch != nil {
		return json.Marshal(jsonPatch)
	}
	return nil, nil
}

func mutatingCreateTTree(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	newTTree := &tarsCrdV1beta3.TTree{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, newTTree)

	businessMap := make(map[string]interface{}, len(newTTree.Businesses))
	for _, business := range newTTree.Businesses {
		businessMap[business.Name] = nil
	}

	var jsonPatch tarsMetaTools.JsonPatch

	for i, app := range newTTree.Apps {
		if app.BusinessRef != "" {
			if _, ok := businessMap[app.BusinessRef]; !ok {
				jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
					OP:    tarsMetaTools.JsonPatchReplace,
					Path:  fmt.Sprintf("/apps/%d/businessRef", i),
					Value: "",
				})
			}
		}
	}

	if jsonPatch != nil {
		return json.Marshal(jsonPatch)
	}
	return nil, nil
}

func mutatingUpdateTTree(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	return mutatingCreateTTree(requestAdmissionView)
}

const PasswordPattern = `^[\x21-\x7e]{6,32}$`
const BcryptHashCost = 6

func generateBcryptPassword(in string) ([]byte, error) {
	sha1String := fmt.Sprintf("%x", sha1.Sum([]byte(in)))
	return bcrypt.GenerateFromPassword([]byte(sha1String), BcryptHashCost)
}

const UnsafeTAccountAnnotationKey = "kubectl.kubernetes.io/last-applied-configuration"
const UnsafeTAccountAnnotationPath = "/metadata/annotations/kubectl.kubernetes.io~1last-applied-configuration"

func mutatingCreateTAccount(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	newTAccount := &tarsCrdV1beta3.TAccount{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, newTAccount)

	var jsonPatch tarsMetaTools.JsonPatch

	if newTAccount.Spec.Authentication.Password != nil {
		passwordString := *newTAccount.Spec.Authentication.Password
		ok, _ := regexp.MatchString(PasswordPattern, passwordString)
		if !ok {
			err := fmt.Errorf("password should match pattern %s", PasswordPattern)
			return nil, err
		}
		bcryptPassword, _ := generateBcryptPassword(passwordString)
		jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
			OP:   tarsMetaTools.JsonPatchRemove,
			Path: "/spec/authentication/password",
		})

		jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
			OP:    tarsMetaTools.JsonPatchAdd,
			Path:  "/spec/authentication/bcryptPassword",
			Value: string(bcryptPassword),
		})
	}

	tokens := make([]tarsCrdV1beta3.TAccountAuthenticationToken, 0, 0)
	jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
		OP:    tarsMetaTools.JsonPatchAdd,
		Path:  "/spec/authentication/tokens",
		Value: tokens,
	})

	if newTAccount.Annotations != nil {
		if _, ok := newTAccount.Annotations[UnsafeTAccountAnnotationKey]; ok {
			jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
				OP:   tarsMetaTools.JsonPatchRemove,
				Path: UnsafeTAccountAnnotationPath,
			})
		}
	}

	if jsonPatch != nil {
		return json.Marshal(jsonPatch)
	}
	return nil, nil
}

func mutatingUpdateTAccount(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	newTAccount := &tarsCrdV1beta3.TAccount{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, newTAccount)

	oldTAccount := &tarsCrdV1beta3.TAccount{}
	_ = json.Unmarshal(requestAdmissionView.Request.OldObject.Raw, oldTAccount)

	var jsonPatch tarsMetaTools.JsonPatch

	if newTAccount.Annotations != nil {
		if _, ok := newTAccount.Annotations[UnsafeTAccountAnnotationKey]; ok {
			jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
				OP:   tarsMetaTools.JsonPatchRemove,
				Path: UnsafeTAccountAnnotationPath,
			})
		}
	}

	passwordChanged := false
	for i := 0; i < 1; i++ {
		if newTAccount.Spec.Authentication.Password != nil {
			passwordString := *newTAccount.Spec.Authentication.Password
			ok, _ := regexp.MatchString(PasswordPattern, passwordString)
			if !ok {
				err := fmt.Errorf("password should match pattern %s", PasswordPattern)
				return nil, err
			}

			bcryptPassword, _ := generateBcryptPassword(passwordString)

			jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
				OP:   tarsMetaTools.JsonPatchRemove,
				Path: "/spec/authentication/password",
			})

			jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
				OP:    tarsMetaTools.JsonPatchAdd,
				Path:  "/spec/authentication/bcryptPassword",
				Value: string(bcryptPassword),
			})

			passwordChanged = true
			break
		}

		if !equality.Semantic.DeepEqual(oldTAccount.Spec.Authentication.BCryptPassword, newTAccount.Spec.Authentication.BCryptPassword) {
			passwordChanged = true
			break
		}
	}

	if passwordChanged {
		tokens := make([]tarsCrdV1beta3.TAccountAuthenticationToken, 0, 0)
		jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
			OP:    tarsMetaTools.JsonPatchAdd,
			Path:  "/spec/authentication/tokens",
			Value: tokens,
		})
	}

	if jsonPatch != nil {
		return json.Marshal(jsonPatch)
	}
	return nil, nil
}

func mutatingCreateTImage(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	timage := &tarsCrdV1beta3.TImage{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, timage)

	var jsonPatch tarsMetaTools.JsonPatch

	if timage.Labels == nil {
		labels := map[string]string{}
		jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
			OP:    tarsMetaTools.JsonPatchAdd,
			Path:  "/metadata/labels",
			Value: labels,
		})
	}

	jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
		OP:    tarsMetaTools.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ImageType",
		Value: timage.ImageType,
	})

	shouldAddSupportedLabel := make(map[string]string, len(timage.SupportedType))

	if timage.ImageType == "base" && timage.SupportedType != nil {
		for _, v := range timage.SupportedType {
			shouldAddSupportedLabel[fmt.Sprintf("tars.io/Supported.%s", v)] = v
		}
	}

	for k := range timage.Labels {
		if _, ok := shouldAddSupportedLabel[k]; ok {
			delete(shouldAddSupportedLabel, k)
		} else {
			if strings.HasPrefix(k, "tars.io/Supported.") {
				v := strings.ReplaceAll(k, "tars.io/", "tars.io~1")
				jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
					OP:   tarsMetaTools.JsonPatchRemove,
					Path: fmt.Sprintf("/metadata/labels/%s", v),
				})
			}
		}
	}

	for _, v := range shouldAddSupportedLabel {
		jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
			OP:    tarsMetaTools.JsonPatchAdd,
			Path:  fmt.Sprintf("/metadata/labels/tars.io~1Supported.%s", v),
			Value: v,
		})
	}

	// if there is a duplicate id, we will keep the previous one
	existing := map[string]interface{}{}
	removes := map[int]interface{}{}
	for i, v := range timage.Releases {
		if _, ok := existing[v.ID]; ok {
			newSeqAfterRemove := i - len(removes)
			jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
				OP:   tarsMetaTools.JsonPatchRemove,
				Path: fmt.Sprintf("/releases/%d", newSeqAfterRemove),
			})
			removes[i] = nil
		}
		existing[v.ID] = nil
	}

	now := k8sMetaV1.Now().ToUnstructured()
	for i, v := range timage.Releases {
		if v.CreateTime.IsZero() {
			if _, ok := removes[i]; !ok {
				newSeqAfterRemove := i
				if i > len(removes) {
					newSeqAfterRemove = i - len(removes)
				}
				jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
					OP:    tarsMetaTools.JsonPatchAdd,
					Path:  fmt.Sprintf("/releases/%d/createTime", newSeqAfterRemove),
					Value: now,
				})
			}
		}
	}

	if jsonPatch != nil {
		return json.Marshal(jsonPatch)
	}
	return nil, nil
}

func mutatingUpdateTImage(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	return mutatingCreateTImage(requestAdmissionView)
}

func mutatingCreateTTemplate(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	ttemplate := &tarsCrdV1beta3.TTemplate{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, ttemplate)

	var jsonPatch tarsMetaTools.JsonPatch

	for i := 0; i < 1; i++ {

		fatherless := ttemplate.Name == ttemplate.Spec.Parent

		if fatherless {
			if ttemplate.Labels != nil {
				if _, ok := ttemplate.Labels[tarsMetaV1beta3.ParentLabel]; ok {
					jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
						OP:   tarsMetaTools.JsonPatchRemove,
						Path: "/metadata/labels/tars.io~1Parent",
					})
				}
			}
			break
		}
		if ttemplate.Labels == nil {
			labels := map[string]string{}
			jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
				OP:    tarsMetaTools.JsonPatchAdd,
				Path:  "/metadata/labels",
				Value: labels,
			})
		}

		jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
			OP:    tarsMetaTools.JsonPatchAdd,
			Path:  "/metadata/labels/tars.io~1Parent",
			Value: ttemplate.Spec.Parent,
		})
	}

	if jsonPatch != nil {
		return json.Marshal(jsonPatch)
	}

	return nil, nil
}

func mutatingUpdateTTemplate(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	return mutatingCreateTTemplate(requestAdmissionView)
}

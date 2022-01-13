package v1beta2

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"hash/crc32"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/integer"
	crdV1beta2 "k8s.tars.io/api/crd/v1beta2"
	crdMeta "k8s.tars.io/api/meta"
	"math"
	"regexp"
	"strconv"
	"strings"
	"tarscontroller/controller"
	"time"
)

type Handler struct {
	clients  *controller.Clients
	informer *controller.Informers
}

func New(clients *controller.Clients, informers *controller.Informers) *Handler {
	return &Handler{clients: clients, informer: informers}
}

var functions map[string]func(*k8sAdmissionV1.AdmissionReview) ([]byte, error)

func init() {
	functions = map[string]func(*k8sAdmissionV1.AdmissionReview) ([]byte, error){
		"CREATE/TDeploy": mutatingCreateTDeploy,
		"UPDATE/TDeploy": mutatingUpdateTDeploy,

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
	}
}

func (v *Handler) Handle(view *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	key := fmt.Sprintf("%s/%s", string(view.Request.Operation), view.Request.Kind.Kind)
	if fun, ok := functions[key]; ok {
		return fun(view)
	}
	return nil, fmt.Errorf("unsupported mutating %s %s.%s", view.Request.Operation, view.Request.Kind.Version, view.Request.Kind.Kind)
}

func mutatingCreateTDeploy(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	tdeploy := &crdV1beta2.TDeploy{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, tdeploy)

	var jsonPatch crdMeta.JsonPatch

	if tdeploy.Labels == nil {
		labels := map[string]string{}
		jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
			OP:    crdMeta.JsonPatchAdd,
			Path:  "/metadata/labels",
			Value: labels,
		})
	}

	jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
		OP:    crdMeta.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1Approve",
		Value: "Pending",
	})

	if tdeploy.Approve != nil {
		jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
			OP:   crdMeta.JsonPatchRemove,
			Path: "/approve",
		})
	}

	if tdeploy.Deployed != nil {
		jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
			OP:   crdMeta.JsonPatchRemove,
			Path: "/deployed",
		})
	}

	if jsonPatch != nil {
		return json.Marshal(jsonPatch)
	}
	return nil, nil
}

func mutatingUpdateTDeploy(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	tdeploy := &crdV1beta2.TDeploy{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, tdeploy)

	var jsonPatch crdMeta.JsonPatch

	if tdeploy.Labels == nil {
		labels := map[string]string{}
		jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
			OP:    crdMeta.JsonPatchAdd,
			Path:  "/metadata/labels",
			Value: labels,
		})
	}

	if tdeploy.Approve == nil {
		jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
			OP:    crdMeta.JsonPatchAdd,
			Path:  "/metadata/labels/tars.io~1Approve",
			Value: "Pending",
		})
	} else if tdeploy.Approve.Result {
		jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
			OP:    crdMeta.JsonPatchAdd,
			Path:  "/metadata/labels/tars.io~1Approve",
			Value: "Approved",
		})
	} else {
		jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
			OP:    crdMeta.JsonPatchAdd,
			Path:  "/metadata/labels/tars.io~1Approve",
			Value: "Reject",
		})
	}

	if jsonPatch != nil {
		return json.Marshal(jsonPatch)
	}
	return nil, nil
}

func mutatingCreateTServer(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	tserver := &crdV1beta2.TServer{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, tserver)

	var jsonPatch crdMeta.JsonPatch

	if tserver.Labels == nil {
		labels := map[string]string{}
		jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
			OP:    crdMeta.JsonPatchAdd,
			Path:  "/metadata/labels",
			Value: labels,
		})
	}

	jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
		OP:    crdMeta.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ServerApp",
		Value: tserver.Spec.App,
	})

	jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
		OP:    crdMeta.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ServerName",
		Value: tserver.Spec.Server,
	})

	jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
		OP:    crdMeta.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1SubType",
		Value: string(tserver.Spec.SubType),
	})

	if tserver.Spec.Tars != nil {
		jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
			OP:    crdMeta.JsonPatchAdd,
			Path:  "/metadata/labels/tars.io~1Template",
			Value: tserver.Spec.Tars.Template,
		})

		if tserver.Spec.K8S.ReadinessGate == nil {
			jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
				OP:    crdMeta.JsonPatchAdd,
				Path:  "/spec/k8s/readinessGate",
				Value: crdMeta.TPodReadinessGate,
			})
		}
	}

	if len(tserver.Spec.K8S.HostPorts) > 0 || tserver.Spec.K8S.HostIPC {
		jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
			OP:    crdMeta.JsonPatchAdd,
			Path:  "/spec/k8s/notStacked",
			Value: true,
		})
	}

	maxReplicasValue := math.MaxInt32
	minReplicasValue := math.MinInt32

	if tserver.Annotations != nil {
		const pattern = "^[1-9]?[0-9]$"
		if maxReplicas, ok := tserver.Annotations[crdMeta.TMaxReplicasAnnotation]; ok {
			matched, _ := regexp.MatchString(pattern, maxReplicas)
			if !matched {
				return nil, fmt.Errorf(crdMeta.ResourceInvalidError, "tserver", "unexpected annotation format")
			}
			maxReplicasValue, _ = strconv.Atoi(maxReplicas)
		}

		if minReplicas, ok := tserver.Annotations[crdMeta.TMinReplicasAnnotation]; ok {
			matched, _ := regexp.MatchString(pattern, minReplicas)
			if !matched {
				return nil, fmt.Errorf(crdMeta.ResourceInvalidError, "tserver", "unexpected annotation format")
			}
			minReplicasValue, _ = strconv.Atoi(minReplicas)
		}

		if minReplicasValue > maxReplicasValue {
			return nil, fmt.Errorf(crdMeta.ResourceInvalidError, "tserver", "unexpected annotation value")
		}
	}

	if tserver.Spec.Release == nil {
		jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
			OP:    crdMeta.JsonPatchReplace,
			Path:  "/spec/k8s/replicas",
			Value: 0,
		})
	} else {
		replicas := int(tserver.Spec.K8S.Replicas)
		replicas = integer.IntMax(replicas, minReplicasValue)
		replicas = integer.IntMin(replicas, maxReplicasValue)

		jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
			OP:    crdMeta.JsonPatchReplace,
			Path:  "/spec/k8s/replicas",
			Value: replicas,
		})

		if tserver.Spec.Release.Time.IsZero() {
			now := k8sMetaV1.Now()
			jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
				OP:    crdMeta.JsonPatchAdd,
				Path:  "/spec/release/time",
				Value: now.ToUnstructured(),
			})
		}

		if tserver.Spec.Tars != nil {
			if tserver.Spec.Release.TServerReleaseNode == nil || tserver.Spec.Release.TServerReleaseNode.Image == "" {
				image, secret := controller.GetDefaultNodeImage(tserver.Namespace)
				if image == crdMeta.ServiceImagePlaceholder {
					return nil, fmt.Errorf(crdMeta.ResourceInvalidError, tserver, "no default node image has been set")
				}

				jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
					OP:    crdMeta.JsonPatchAdd,
					Path:  "/spec/release/nodeImage",
					Value: image,
				})

				jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
					OP:    crdMeta.JsonPatchAdd,
					Path:  "/spec/release/nodeSecret",
					Value: secret,
				})
			}
		}

		if tserver.Spec.Normal != nil {
			if tserver.Spec.Release.TServerReleaseNode != nil {
				if tserver.Spec.Release.TServerReleaseNode.Image != "" {
					jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
						OP:   crdMeta.JsonPatchRemove,
						Path: "/spec/release/nodeImage",
					})
				}

				if tserver.Spec.Release.TServerReleaseNode.Secret != "" {
					jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
						OP:   crdMeta.JsonPatchRemove,
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
	tconfig := &crdV1beta2.TConfig{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, tconfig)

	var jsonPatch crdMeta.JsonPatch

	if tconfig.Labels == nil {
		labels := map[string]string{}
		jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
			OP:    crdMeta.JsonPatchAdd,
			Path:  "/metadata/labels",
			Value: labels,
		})
	}

	jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
		OP:    crdMeta.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ServerApp",
		Value: tconfig.App,
	})

	jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
		OP:    crdMeta.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ServerName",
		Value: tconfig.Server,
	})

	jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
		OP:    crdMeta.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ConfigName",
		Value: tconfig.ConfigName,
	})

	jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
		OP:    crdMeta.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1PodSeq",
		Value: tconfig.PodSeq,
	})

	jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
		OP:    crdMeta.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1Activated",
		Value: tconfig.Activated,
	})

	versionString := fmt.Sprintf("%s-%x", time.Now().Format("20060102030405"), crc32.ChecksumIEEE([]byte(tconfig.Name)))
	jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
		OP:    crdMeta.JsonPatchAdd,
		Path:  "/version",
		Value: versionString,
	})

	jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
		OP:    crdMeta.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1Version",
		Value: versionString,
	})

	if jsonPatch != nil {
		return json.Marshal(jsonPatch)
	}
	return nil, nil
}

func mutatingUpdateTConfig(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	tconfig := &crdV1beta2.TConfig{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, tconfig)

	var jsonPatch crdMeta.JsonPatch

	if tconfig.Labels == nil {
		labels := map[string]string{}
		jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
			OP:    crdMeta.JsonPatchAdd,
			Path:  "/metadata/labels",
			Value: labels,
		})
	}

	jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
		OP:    crdMeta.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ServerApp",
		Value: tconfig.App,
	})

	jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
		OP:    crdMeta.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ServerName",
		Value: tconfig.Server,
	})

	jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
		OP:    crdMeta.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1ConfigName",
		Value: tconfig.ConfigName,
	})

	jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
		OP:    crdMeta.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1PodSeq",
		Value: tconfig.PodSeq,
	})

	jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
		OP:    crdMeta.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1Activated",
		Value: tconfig.Activated,
	})

	jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
		OP:    crdMeta.JsonPatchAdd,
		Path:  "/metadata/labels/tars.io~1Version",
		Value: tconfig.Version,
	})

	if jsonPatch != nil {
		return json.Marshal(jsonPatch)
	}
	return nil, nil
}

func mutatingCreateTTree(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	newTTree := &crdV1beta2.TTree{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, newTTree)

	businessMap := make(map[string]interface{}, len(newTTree.Businesses))
	for _, business := range newTTree.Businesses {
		businessMap[business.Name] = nil
	}

	var jsonPatch crdMeta.JsonPatch

	for i, app := range newTTree.Apps {
		if app.BusinessRef != "" {
			if _, ok := businessMap[app.BusinessRef]; !ok {
				newTTreeApps := &crdV1beta2.TTreeApp{
					Name:         app.Name,
					BusinessRef:  "",
					CreatePerson: app.CreatePerson,
					CreateTime:   app.CreateTime,
					Mark:         app.Mark,
				}
				bs, _ := json.Marshal(newTTreeApps)
				jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
					OP:    crdMeta.JsonPatchReplace,
					Path:  fmt.Sprintf("/apps/%d", i),
					Value: string(bs),
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
	newTAccount := &crdV1beta2.TAccount{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, newTAccount)

	var jsonPatch crdMeta.JsonPatch

	if newTAccount.Spec.Authentication.Password != nil {
		passwordString := *newTAccount.Spec.Authentication.Password
		ok, _ := regexp.MatchString(PasswordPattern, passwordString)
		if !ok {
			err := fmt.Errorf("password should match pattern %s", PasswordPattern)
			return nil, err
		}
		bcryptPassword, _ := generateBcryptPassword(passwordString)
		jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
			OP:   crdMeta.JsonPatchRemove,
			Path: "/spec/authentication/password",
		})

		jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
			OP:    crdMeta.JsonPatchAdd,
			Path:  "/spec/authentication/bcryptPassword",
			Value: string(bcryptPassword),
		})
	}

	tokens := make([]crdV1beta2.TAccountAuthenticationToken, 0, 0)
	jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
		OP:    crdMeta.JsonPatchAdd,
		Path:  "/spec/authentication/tokens",
		Value: tokens,
	})

	if newTAccount.Annotations != nil {
		if _, ok := newTAccount.Annotations[UnsafeTAccountAnnotationKey]; ok {
			jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
				OP:   crdMeta.JsonPatchRemove,
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
	newTAccount := &crdV1beta2.TAccount{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, newTAccount)

	oldTAccount := &crdV1beta2.TAccount{}
	_ = json.Unmarshal(requestAdmissionView.Request.OldObject.Raw, oldTAccount)

	var jsonPatch crdMeta.JsonPatch

	if newTAccount.Annotations != nil {
		if _, ok := newTAccount.Annotations[UnsafeTAccountAnnotationKey]; ok {
			jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
				OP:   crdMeta.JsonPatchRemove,
				Path: UnsafeTAccountAnnotationPath,
			})
		}
	}

	passwordChanged := false
	for {
		if newTAccount.Spec.Authentication.Password != nil {
			passwordString := *newTAccount.Spec.Authentication.Password
			ok, _ := regexp.MatchString(PasswordPattern, passwordString)
			if !ok {
				err := fmt.Errorf("password should match pattern %s", PasswordPattern)
				return nil, err
			}

			bcryptPassword, _ := generateBcryptPassword(passwordString)

			jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
				OP:   crdMeta.JsonPatchRemove,
				Path: "/spec/authentication/password",
			})

			jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
				OP:    crdMeta.JsonPatchAdd,
				Path:  "/spec/authentication/bcryptPassword",
				Value: string(bcryptPassword),
			})

			passwordChanged = true
			break
		}

		if newTAccount.Spec.Authentication.BCryptPassword == nil {
			passwordChanged = true
			break
		}

		if oldTAccount.Spec.Authentication.BCryptPassword == nil {
			passwordChanged = true
		}

		oldPassword := []byte(*oldTAccount.Spec.Authentication.BCryptPassword)
		newPassword := []byte(*newTAccount.Spec.Authentication.BCryptPassword)

		err := bcrypt.CompareHashAndPassword(oldPassword, newPassword)
		if err != nil {
			passwordChanged = true
			break
		}
	}

	if passwordChanged {
		tokens := make([]crdV1beta2.TAccountAuthenticationToken, 0, 0)
		jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
			OP:    crdMeta.JsonPatchAdd,
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
	timage := &crdV1beta2.TImage{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, timage)

	var jsonPatch crdMeta.JsonPatch

	if timage.Labels == nil {
		labels := map[string]string{}
		jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
			OP:    crdMeta.JsonPatchAdd,
			Path:  "/metadata/labels",
			Value: labels,
		})
	}

	jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
		OP:    crdMeta.JsonPatchAdd,
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
				jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
					OP:   crdMeta.JsonPatchRemove,
					Path: fmt.Sprintf("/metadata/labels/%s", v),
				})
			}
		}
	}

	for _, v := range shouldAddSupportedLabel {
		jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
			OP:    crdMeta.JsonPatchAdd,
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
			jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
				OP:   crdMeta.JsonPatchRemove,
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
				jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
					OP:    crdMeta.JsonPatchAdd,
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

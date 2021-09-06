package v1alpha2

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"hash/crc32"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crdV1alpha2 "k8s.tars.io/api/crd/v1alpha2"
	"regexp"
	"strings"
	"tarscontroller/meta"
	"time"
)

type Handler struct {
	clients  *meta.Clients
	informer *meta.Informers
}

func New(clients *meta.Clients, informers *meta.Informers) *Handler {
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
	tdeploy := &crdV1alpha2.TDeploy{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, tdeploy)

	var patchContents = make([][]byte, 0, 10)
	patchContents = append(patchContents, []byte{'['})

	if tdeploy.Labels == nil {
		patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels\",\"value\":{}}")))
	}
	patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels/tars.io~1Approve\",\"value\":\"Pending\"}")))

	if tdeploy.Approve != nil {
		patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"remove\",\"path\":\"/approve\"}")))
	}

	if tdeploy.Deployed != nil {
		patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"remove\",\"path\":\"/deployed\"}")))
	}

	if len(patchContents) == 1 {
		return nil, nil
	}

	totalPatchContent := bytes.Join(patchContents, []byte{','})
	totalPatchContent[1] = ' '
	totalPatchContent = append(totalPatchContent, ']')
	return totalPatchContent, nil
}

func mutatingUpdateTDeploy(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	tdeploy := &crdV1alpha2.TDeploy{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, tdeploy)

	var patchContents = make([][]byte, 0, 10)

	patchContents = append(patchContents, []byte{'['})

	if tdeploy.Labels == nil {
		patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels\",\"value\":{}}")))
	}

	if tdeploy.Approve == nil {
		patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels/tars.io~1Approve\",\"value\":\"Pending\"}")))
	} else if tdeploy.Approve.Result {
		patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels/tars.io~1Approve\",\"value\":\"Approved\"}")))
	} else {
		patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels/tars.io~1Approve\",\"value\":\"Reject\"}")))
	}

	if len(patchContents) == 1 {
		return nil, nil
	}

	totalPatchContent := bytes.Join(patchContents, []byte{','})
	totalPatchContent[1] = ' '
	totalPatchContent = append(totalPatchContent, ']')
	return totalPatchContent, nil
}

func mutatingCreateTServer(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	tserver := &crdV1alpha2.TServer{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, tserver)

	var patchContents = make([][]byte, 0, 10)
	patchContents = append(patchContents, []byte{'['})

	if tserver.Labels == nil {
		patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels\",\"value\":{}}")))
	}

	patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\": \"add\", \"path\": \"/metadata/labels/tars.io~1ServerApp\", \"value\": \"%s\"}", tserver.Spec.App)))

	patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\": \"add\", \"path\": \"/metadata/labels/tars.io~1ServerName\", \"value\": \"%s\"}", tserver.Spec.Server)))

	patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\": \"add\", \"path\": \"/metadata/labels/tars.io~1SubType\", \"value\": \"%s\"}", tserver.Spec.SubType)))

	if tserver.Spec.Tars != nil {
		patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\": \"add\", \"path\": \"/metadata/labels/tars.io~1Template\", \"value\": \"%s\"}", tserver.Spec.Tars.Template)))
		if tserver.Spec.K8S.ReadinessGate == nil {
			patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\": \"add\", \"path\":\"/spec/k8s/readinessGate\",\"value\":\"%s\"}", meta.TPodReadinessGate)))
		}
	}

	if tserver.Spec.Release == nil {
		patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\": \"add\", \"path\":\"/spec/k8s/replicas\",\"value\": %d}", 0)))
	} else {

		if tserver.Spec.Release.Time.IsZero() {
			now := k8sMetaV1.Now()
			nowBS, _ := json.Marshal(now)
			nowString := string(nowBS)
			patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/spec/release/time\",\"value\":%s}", nowString)))
		}

		if tserver.Spec.K8S.Replicas == nil {
			if requestAdmissionView.Request.Object.Raw == nil {
				patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\": \"add\", \"path\":\"/spec/k8s/replicas\",\"value\": %d}", 1)))
			} else {
				oldTServer := &crdV1alpha2.TServer{}
				_ = json.Unmarshal(requestAdmissionView.Request.OldObject.Raw, oldTServer)
				patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\": \"add\", \"path\":\"/spec/k8s/replicas\",\"value\": %d}", *oldTServer.Spec.K8S.Replicas)))
			}
		}
	}

	if len(tserver.Spec.K8S.HostPorts) > 0 || tserver.Spec.K8S.HostIPC {
		patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\": \"add\", \"path\":\"/spec/k8s/notStacked\",\"value\":%t}", true)))
	}

	if len(patchContents) == 1 {
		return nil, nil
	}

	totalPatchContent := bytes.Join(patchContents, []byte{','})
	totalPatchContent[1] = ' '
	totalPatchContent = append(totalPatchContent, ']')
	return totalPatchContent, nil
}

func mutatingUpdateTServer(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	return mutatingCreateTServer(requestAdmissionView)
}

func mutatingCreateTConfig(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	tconfig := &crdV1alpha2.TConfig{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, tconfig)

	var patchContents = make([][]byte, 0, 8)
	patchContents = append(patchContents, []byte{'['})

	if tconfig.Labels == nil {
		patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels\",\"value\":{}}")))
	}

	patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels/tars.io~1ServerApp\",\"value\":\"%s\"}", tconfig.App)))
	patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels/tars.io~1ServerName\",\"value\":\"%s\"}", tconfig.Server)))
	patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels/tars.io~1ConfigName\",\"value\":\"%s\"}", tconfig.ConfigName)))
	patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels/tars.io~1PodSeq\",\"value\":\"%s\"}", tconfig.PodSeq)))
	patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels/tars.io~1Activated\",\"value\":\"%t\"}", tconfig.Activated)))

	versionString := fmt.Sprintf("%s-%x", time.Now().Format("20060102030405"), crc32.ChecksumIEEE([]byte(tconfig.Name)))
	patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/version\",\"value\":\"%s\"}", versionString)))
	patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels/tars.io~1Version\",\"value\":\"%s\"}", versionString)))

	if len(patchContents) == 1 {
		return nil, nil
	}
	totalPatchContent := bytes.Join(patchContents, []byte{','})
	totalPatchContent[1] = ' '
	totalPatchContent = append(totalPatchContent, ']')
	return totalPatchContent, nil
}

func mutatingUpdateTConfig(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	tconfig := &crdV1alpha2.TConfig{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, tconfig)

	var patchContents = make([][]byte, 0, 8)
	patchContents = append(patchContents, []byte{'['})

	if tconfig.Labels == nil {
		patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels\",\"value\":{}}")))
	}

	patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels/tars.io~1ServerApp\",\"value\":\"%s\"}", tconfig.App)))
	patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels/tars.io~1ServerName\",\"value\":\"%s\"}", tconfig.Server)))
	patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels/tars.io~1ConfigName\",\"value\":\"%s\"}", tconfig.ConfigName)))
	patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels/tars.io~1PodSeq\",\"value\":\"%s\"}", tconfig.PodSeq)))

	patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels/tars.io~1Activated\",\"value\":\"%t\"}", tconfig.Activated)))
	patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels/tars.io~1Version\",\"value\":\"%s\"}", tconfig.Version)))

	if len(patchContents) == 1 {
		return nil, nil
	}
	totalPatchContent := bytes.Join(patchContents, []byte{','})
	totalPatchContent[1] = ' '
	totalPatchContent = append(totalPatchContent, ']')
	return totalPatchContent, nil
}

func mutatingCreateTTree(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	newTTree := &crdV1alpha2.TTree{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, newTTree)

	businessMap := make(map[string]interface{}, len(newTTree.Businesses))
	for _, business := range newTTree.Businesses {
		businessMap[business.Name] = nil
	}

	var patchContents = make([][]byte, 0, 5)
	patchContents = append(patchContents, []byte{'['})

	for i, app := range newTTree.Apps {
		if app.BusinessRef != "" {
			if _, ok := businessMap[app.BusinessRef]; !ok {
				newTTreeApps := &crdV1alpha2.TTreeApp{
					Name:         app.Name,
					BusinessRef:  "",
					CreatePerson: app.CreatePerson,
					CreateTime:   app.CreateTime,
					Mark:         app.Mark,
				}
				bs, _ := json.Marshal(newTTreeApps)
				patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"replace\",\"path\":\"/apps/%d\",\"value\":\"%s\"}", i, bs)))
			}
		}
	}

	if len(patchContents) == 1 {
		return nil, nil
	}

	totalPatchContent := bytes.Join(patchContents, []byte{','})
	totalPatchContent[1] = ' '
	totalPatchContent = append(totalPatchContent, ']')
	return totalPatchContent, nil
}

func mutatingUpdateTTree(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	return mutatingCreateTTree(requestAdmissionView)
}

const PasswordPattern = `^[\x21-\x7e]{6,32}$`
const BcryptHashCost = 6

func BcryptPassword(in string) ([]byte, error) {
	sha1String := fmt.Sprintf("%x", sha1.Sum([]byte(in)))
	return bcrypt.GenerateFromPassword([]byte(sha1String), BcryptHashCost)
}

func mutatingCreateTAccount(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	newTAccount := &crdV1alpha2.TAccount{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, newTAccount)

	var patchContents = make([][]byte, 0, 6)
	patchContents = append(patchContents, []byte{'['})

	if newTAccount.Spec.Authentication.Password != nil {
		passwordString := *newTAccount.Spec.Authentication.Password
		ok, _ := regexp.MatchString(PasswordPattern, passwordString)
		if !ok {
			err := fmt.Errorf("password should match pattern %s", PasswordPattern)
			return nil, err
		}
		bcryptPassword, _ := BcryptPassword(passwordString)
		patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"remove\",\"path\":\"/spec/authentication/password\"}")))
		patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/spec/authentication/bcryptPassword\",\"value\":\"%s\"}", bcryptPassword)))
	}
	if newTAccount.Spec.Authentication.Tokens == nil || len(newTAccount.Spec.Authentication.Tokens) != 0 {
		patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/spec/authentication/tokens\",\"value\":[]}")))
	}
	if len(patchContents) == 1 {
		return nil, nil
	}
	totalPatchContent := bytes.Join(patchContents, []byte{','})
	totalPatchContent[1] = ' '
	totalPatchContent = append(totalPatchContent, ']')
	return totalPatchContent, nil
}

func mutatingUpdateTAccount(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	newTAccount := &crdV1alpha2.TAccount{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, newTAccount)

	oldTAccount := &crdV1alpha2.TAccount{}
	_ = json.Unmarshal(requestAdmissionView.Request.OldObject.Raw, oldTAccount)

	var patchContents = make([][]byte, 0, 5)
	patchContents = append(patchContents, []byte{'['})

	passwordChanged := false
	for {
		if newTAccount.Spec.Authentication.Password != nil {
			passwordString := *newTAccount.Spec.Authentication.Password
			ok, _ := regexp.MatchString(PasswordPattern, passwordString)
			if !ok {
				err := fmt.Errorf("password should match pattern %s", PasswordPattern)
				return nil, err
			}
			bcryptPassword, _ := BcryptPassword(passwordString)
			patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"remove\",\"path\":\"/spec/authentication/password\"}")))
			patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/spec/authentication/bcryptPassword\",\"value\":\"%s\"}", bcryptPassword)))
			passwordChanged = true
			break
		}

		if newTAccount.Spec.Authentication.BCryptPassword == nil {
			passwordChanged = true
			break
		}

		if !equality.Semantic.DeepEqual(oldTAccount.Spec.Authentication.BCryptPassword, newTAccount.Spec.Authentication.BCryptPassword) {
			passwordChanged = true
			break
		}

		break
	}

	if passwordChanged {
		patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/spec/authentication/tokens\",\"value\":[]}")))
	}

	if len(patchContents) == 1 {
		return nil, nil
	}

	totalPatchContent := bytes.Join(patchContents, []byte{','})
	totalPatchContent[1] = ' '
	totalPatchContent = append(totalPatchContent, ']')
	return totalPatchContent, nil
}

func mutatingCreateTImage(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	newTImage := &crdV1alpha2.TImage{}
	_ = json.Unmarshal(requestAdmissionView.Request.Object.Raw, newTImage)

	var patchContents = make([][]byte, 0, 6)
	patchContents = append(patchContents, []byte{'['})

	if newTImage.Labels == nil {
		patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels\",\"value\":{}}")))
	}
	patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels/tars.io~1ImageType\",\"value\":\"%s\"}", newTImage.ImageType)))

	shouldAddSupportedLabel := make(map[string]string, len(newTImage.SupportedType))

	if newTImage.ImageType == "base" && newTImage.SupportedType != nil {
		for _, v := range newTImage.SupportedType {
			shouldAddSupportedLabel[fmt.Sprintf("tars.io/Supported.%s", v)] = v
		}
	}

	for k := range newTImage.Labels {
		if _, ok := shouldAddSupportedLabel[k]; ok {
			delete(shouldAddSupportedLabel, k)
		} else {
			if strings.HasPrefix(k, "tars.io/Supported.") {
				v := strings.ReplaceAll(k, "tars.io/", "tars.io~1")
				patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"remove\",\"path\":\"/metadata/labels/%s\"}", v)))
			}
		}
	}

	for _, v := range shouldAddSupportedLabel {
		patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/metadata/labels/tars.io~1Supported.%s\",\"value\":\"\"}", v)))
	}

	now := k8sMetaV1.Now()
	nowBS, _ := json.Marshal(now)
	nowString := string(nowBS)
	for i, v := range newTImage.Releases {
		if v.CreateTime.IsZero() {
			patchContents = append(patchContents, []byte(fmt.Sprintf("{\"op\":\"add\",\"path\":\"/releases/%d/createTime\",\"value\":%s}", i, nowString)))
		}
	}

	totalPatchContent := bytes.Join(patchContents, []byte{','})
	totalPatchContent[1] = ' '
	totalPatchContent = append(totalPatchContent, ']')
	return totalPatchContent, nil
}

func mutatingUpdateTImage(requestAdmissionView *k8sAdmissionV1.AdmissionReview) ([]byte, error) {
	return mutatingCreateTImage(requestAdmissionView)
}

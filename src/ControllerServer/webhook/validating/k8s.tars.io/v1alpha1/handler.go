package v1alpha1

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	patchTypes "k8s.io/apimachinery/pkg/types"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	crdV1alpha1 "k8s.tars.io/api/crd/v1alpha1"
	"strings"
	"tarscontroller/meta"
)

var functions = map[string]func(*meta.Clients, *meta.Informers, *k8sAdmissionV1.AdmissionReview) error{}

func init() {
	functions = map[string]func(*meta.Clients, *meta.Informers, *k8sAdmissionV1.AdmissionReview) error{

		"CREATE/TDeploy": validCreateTDeploy,
		"UPDATE/TDeploy": validUpdateTDeploy,
		"DELETE/TDeploy": validDeleteTDeploy,

		"CREATE/TServer": validCreateTServer,
		"UPDATE/TServer": validUpdateTServer,
		"DELETE/TServer": validDeleteTServer,

		"CREATE/TEndpoint": validCreateTEndpoint,
		"UPDATE/TEndpoint": validUpdateTEndpoint,
		"DELETE/TEndpoint": validDeleteTEndpoint,

		"CREATE/TConfig": validCreateTConfig,
		"UPDATE/TConfig": validUpdateTConfig,
		"DELETE/TConfig": validDeleteTConfig,

		"CREATE/TTemplate": validCreateTTemplate,
		"UPDATE/TTemplate": validUpdateTTemplate,
		"DELETE/TTemplate": validDeleteTTemplate,

		"CREATE/TImage": validCreateTImage,
		"UPDATE/TImage": validUpdateTImage,
		"DELETE/TImage": validDeleteTImage,

		"CREATE/TTree": validCreateTTree,
		"UPDATE/TTree": validUpdateTTree,
		"DELETE/TTree": validDeleteTTree,

		"CREATE/TAccount": validCreateTAccount,
		"UPDATE/TAccount": validUpdateTAccount,
		"DELETE/TAccount": validDeleteTAccount,
	}
}

type Handler struct {
	clients  *meta.Clients
	informer *meta.Informers
}

func New(clients *meta.Clients, informers *meta.Informers) *Handler {
	return &Handler{clients: clients, informer: informers}
}

func (v *Handler) Handle(view *k8sAdmissionV1.AdmissionReview) error {
	key := fmt.Sprintf("%s/%s", view.Request.Operation, view.Request.Kind.Kind)
	if fun, ok := functions[key]; ok {
		return fun(v.clients, v.informer, view)
	}
	return fmt.Errorf("unsupported validating %s %s.%s", view.Request.Operation, view.Request.Kind.Version, view.Request.Kind.Kind)
}

func validCreateTDeploy(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	tdeploy := &crdV1alpha1.TDeploy{}
	_ = json.Unmarshal(view.Request.Object.Raw, tdeploy)

	if tdeploy.Approve != nil {
		return fmt.Errorf("should not set /approve field when create tdeploy resource")
	}

	if tdeploy.Deployed != nil {
		return fmt.Errorf("should not set /deployed field when create tdeploy resource")
	}

	return nil
}

func validUpdateTDeploy(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	if view.Request.UserInfo.Username == meta.GetControllerUsername() {
		return nil
	}

	newTDeploy := &crdV1alpha1.TDeploy{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTDeploy)

	oldTDeploy := &crdV1alpha1.TDeploy{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTDeploy)

	if !equality.Semantic.DeepEqual(newTDeploy.Deployed, oldTDeploy.Deployed) {
		return fmt.Errorf("only use authorized account can set \"/deployed\" field")
	}

	if oldTDeploy.Approve != nil {
		if !equality.Semantic.DeepEqual(newTDeploy.Apply, oldTDeploy.Apply) {
			return fmt.Errorf(meta.FiledImmutableError, "tserver", "/apply")
		}
		if !equality.Semantic.DeepEqual(newTDeploy.Approve, oldTDeploy.Approve) {
			return fmt.Errorf(meta.FiledImmutableError, "tserver", "/approve")
		}

		return nil
	}

	if newTDeploy.Approve == nil || newTDeploy.Approve.Result == false {
		return nil
	}

	namespace := newTDeploy.Namespace

	tserverName := fmt.Sprintf("%s-%s", strings.ToLower(newTDeploy.Apply.App), strings.ToLower(newTDeploy.Apply.Server))
	_, err := informer.TServerInformer.Lister().TServers(namespace).Get(tserverName)

	if err == nil {
		return fmt.Errorf(meta.ResourceExistError, "tserver", namespace, tserverName)
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf(meta.ResourceGetError, "tserver", namespace, tserverName, err.Error())
	}

	fakeTServer := &crdV1alpha1.TServer{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      tserverName,
			Namespace: namespace,
		},
		Spec: newTDeploy.Apply,
	}
	return validTServer(fakeTServer, nil, clients, informer)
}

func validDeleteTDeploy(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}

func validTServer(newTServer, oldTServer *crdV1alpha1.TServer, clients *meta.Clients, informer *meta.Informers) error {

	if oldTServer != nil {
		if newTServer.Spec.App != oldTServer.Spec.App {
			return fmt.Errorf(meta.FiledImmutableError, "tserver", ".spec.app")
		}

		if newTServer.Spec.Server != oldTServer.Spec.Server {
			return fmt.Errorf(meta.FiledImmutableError, "tserver", ".spec.server")
		}

		if newTServer.Spec.SubType != oldTServer.Spec.SubType {
			return fmt.Errorf(meta.FiledImmutableError, "tserver", ".spec.subType")
		}

		if oldTServer.Spec.Tars == nil {
			if newTServer.Spec.Tars != nil {
				return fmt.Errorf(meta.FiledImmutableError, "tserver", ".spec.tars")
			}
		}

		if oldTServer.Spec.Normal == nil {
			if newTServer.Spec.Normal != nil {
				return fmt.Errorf(meta.FiledImmutableError, "tserver", ".spec.normal")
			}
		}
	}

	namespace := newTServer.Namespace

	if newTServer.Name != strings.ToLower(newTServer.Spec.App)+"-"+strings.ToLower(newTServer.Spec.Server) {
		return fmt.Errorf(meta.ResourceInvalidError, "tserver", "unexpected resource name")
	}

	portNames := map[string]interface{}{}
	portValues := map[int32]interface{}{}

	if newTServer.Spec.Tars != nil {

		for _, servant := range newTServer.Spec.Tars.Servants {
			portName := strings.ToLower(servant.Name)
			portValue := servant.Port

			if portName == meta.NodeServantName {
				return fmt.Errorf(meta.ResourceInvalidError, "tserver", fmt.Sprintf("servants name value should not equal %s", meta.NodeServantName))
			}

			if portValue == meta.NodeServantPort {
				return fmt.Errorf(meta.ResourceInvalidError, "tserver", fmt.Sprintf("servants port value should not equal %d", meta.NodeServantPort))
			}

			if _, ok := portNames[portName]; ok {
				return fmt.Errorf(meta.ResourceInvalidError, "tserver", fmt.Sprintf("duplicate servant name value %s", servant.Name))
			}

			if _, ok := portValues[portValue]; ok {
				return fmt.Errorf(meta.ResourceInvalidError, "tserver", fmt.Sprintf("duplicate port value %d", servant.Port))
			}

			portNames[portName] = nil
			portValues[portValue] = nil
		}

		for _, port := range newTServer.Spec.Tars.Ports {
			portName := strings.ToLower(port.Name)
			portValue := port.Port

			if portName == meta.NodeServantName {
				return fmt.Errorf(meta.ResourceInvalidError, "tserver", fmt.Sprintf("port name value should not equal %s", meta.NodeServantName))
			}

			if portValue == meta.NodeServantPort {
				return fmt.Errorf(meta.ResourceInvalidError, "tserver", fmt.Sprintf("port value should not equal %d", meta.NodeServantPort))
			}

			if _, ok := portNames[portName]; ok {
				return fmt.Errorf(meta.ResourceInvalidError, "tserver", fmt.Sprintf("duplicate port name value %s", port.Name))
			}

			if _, ok := portValues[portValue]; ok {
				return fmt.Errorf(meta.ResourceInvalidError, "tserver", fmt.Sprintf("duplicate port value %d", port.Port))
			}
			portNames[portName] = nil
			portValues[portValue] = nil
		}

		templateName := newTServer.Spec.Tars.Template
		_, err := informer.TTemplateInformer.Lister().TTemplates(namespace).Get(templateName)
		if err != nil {
			if !errors.IsNotFound(err) {
				return fmt.Errorf(meta.ResourceGetError, "ttemplate", namespace, templateName, err.Error())
			}
			return fmt.Errorf(meta.ResourceNotExistError, "ttemplate", namespace, templateName)
		}
	} else if newTServer.Spec.Normal != nil {
		for _, port := range newTServer.Spec.Normal.Ports {
			portName := strings.ToLower(port.Name)
			portValue := port.Port

			if _, ok := portNames[portName]; ok {
				return fmt.Errorf(meta.ResourceInvalidError, "tserver", fmt.Sprintf("duplicate port name value %s", port.Name))
			}

			if _, ok := portValues[portValue]; ok {
				return fmt.Errorf(meta.ResourceInvalidError, "tserver", fmt.Sprintf("duplicate port value %d", port.Port))
			}
			portNames[portName] = nil
			portValues[portValue] = nil
		}
	}

	if newTServer.Spec.K8S.HostPorts != nil {

		hostPortPorts := map[int32]interface{}{}
		hostPortNameRefs := map[string]interface{}{}

		for _, hostPort := range newTServer.Spec.K8S.HostPorts {
			nameRef := strings.ToLower(hostPort.NameRef)
			if _, ok := portNames[nameRef]; !ok {
				return fmt.Errorf(meta.ResourceInvalidError, "tserver", fmt.Sprintf("port name %s not exist", hostPort.NameRef))
			}

			if _, ok := hostPortNameRefs[nameRef]; ok {
				return fmt.Errorf(meta.ResourceInvalidError, "tserver", fmt.Sprintf("duplicate nameRef value %s", hostPort.NameRef))
			}

			if _, ok := hostPortPorts[hostPort.Port]; ok {
				return fmt.Errorf(meta.ResourceInvalidError, "tserver", fmt.Sprintf("duplicate port value %d", hostPort.Port))
			}

			hostPortNameRefs[hostPort.NameRef] = nil
			hostPortPorts[hostPort.Port] = nil
		}
	}

	if newTServer.Spec.K8S.Mounts != nil {
		mountsNames := map[string]interface{}{}
		for i := range newTServer.Spec.K8S.Mounts {
			mount := &newTServer.Spec.K8S.Mounts[i]
			if _, ok := mountsNames[mount.Name]; ok {
				return fmt.Errorf(meta.ResourceInvalidError, "tserver", fmt.Sprintf("duplicate .mounts.name value %s", mount.Name))
			}
		}
	}
	return nil
}

func validCreateTServer(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	newTServer := &crdV1alpha1.TServer{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTServer)
	return validTServer(newTServer, nil, clients, informer)
}

func validUpdateTServer(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	newTServer := &crdV1alpha1.TServer{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTServer)

	oldTServer := &crdV1alpha1.TServer{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTServer)

	return validTServer(newTServer, oldTServer, clients, informer)
}

func validDeleteTServer(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}

func validCreateTEndpoint(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	if view.Request.UserInfo.Username == meta.GetControllerUsername() {
		return nil
	}
	return fmt.Errorf("only use authorized account can create tendpoints")
}

func validUpdateTEndpoint(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	if view.Request.UserInfo.Username == meta.GetControllerUsername() {
		return nil
	}
	return fmt.Errorf("only use authorized account can update tendpoints")
}

func validDeleteTEndpoint(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}

func prepareActiveTConfig(newTConfig *crdV1alpha1.TConfig, clients *meta.Clients, informer *meta.Informers) error {
	namespace := newTConfig.Namespace
	var podSeq string
	if newTConfig.PodSeq == nil {
		podSeq = "m"
	} else {
		podSeq = *newTConfig.PodSeq
	}

	appRequirement, _ := labels.NewRequirement(meta.TServerAppLabel, "==", []string{newTConfig.App})
	serverRequirement, _ := labels.NewRequirement(meta.TServerNameLabel, "==", []string{newTConfig.Server})
	configNameRequirement, _ := labels.NewRequirement(meta.TConfigNameLabel, "==", []string{newTConfig.ConfigName})
	podSeqRequirement, _ := labels.NewRequirement(meta.TConfigPodSeqLabel, "==", []string{podSeq})
	activateRequirement, _ := labels.NewRequirement(meta.TConfigActivatedLabel, "==", []string{"true"})

	labelSelector := labels.NewSelector().Add(*appRequirement).Add(*serverRequirement).Add(*configNameRequirement).Add(*podSeqRequirement).Add(*activateRequirement)

	tconfigs, err := informer.TConfigInformer.Lister().ByNamespace(namespace).List(labelSelector)
	if err != nil {
		err = fmt.Errorf(meta.ResourceSelectorError, namespace, "tconfig", err.Error())
		utilRuntime.HandleError(err)
		return err
	}

	for _, tconfig := range tconfigs {
		v := tconfig.(k8sMetaV1.Object)
		name := v.GetName()

		if name == newTConfig.Name {
			continue
		}

		patchContent := []byte(fmt.Sprintf("[{\"op\":\"add\",\"path\":\"/metadata/labels/tars.io~1Deactivate\",\"value\":\"%s\"}]", "Deactivating"))
		_, err = clients.CrdClient.CrdV1alpha1().TConfigs(namespace).Patch(context.TODO(), name, patchTypes.JSONPatchType, patchContent, k8sMetaV1.PatchOptions{})
		if err != nil {
			err = fmt.Errorf(meta.ResourcePatchError, "tconfig", namespace, name, err.Error())
			utilRuntime.HandleError(err)
			return err
		}
	}
	return nil
}

func prepareDeleteTConfig(tconfig *crdV1alpha1.TConfig, clients *meta.Clients, informer *meta.Informers) error {
	appRequirement, _ := labels.NewRequirement(meta.TServerAppLabel, "==", []string{tconfig.App})
	serverRequirement, _ := labels.NewRequirement(meta.TServerNameLabel, "==", []string{tconfig.Server})
	configNameRequirement, _ := labels.NewRequirement(meta.TConfigNameLabel, "==", []string{tconfig.ConfigName})
	labelSelector := labels.NewSelector().Add(*appRequirement).Add(*serverRequirement).Add(*configNameRequirement)

	if tconfig.PodSeq != nil {
		podSeqRequirement, _ := labels.NewRequirement(meta.TConfigPodSeqLabel, "==", []string{*tconfig.PodSeq})
		labelSelector = labels.NewSelector().Add(*podSeqRequirement)
	}

	namespace := tconfig.Namespace
	tconfigRuntimeObjs, err := informer.TConfigInformer.Lister().ByNamespace(namespace).List(labelSelector)

	if err != nil && !errors.IsNotFound(err) {
		err = fmt.Errorf(meta.ResourceSelectorError, namespace, "tconfig", err.Error())
		utilRuntime.HandleError(err)
		return err
	}

	if tconfig.PodSeq != nil {
		for _, tconfigRuntimeObj := range tconfigRuntimeObjs {
			tconfigObj := tconfigRuntimeObj.(k8sMetaV1.Object)
			name := tconfigObj.GetName()

			if name == tconfig.Name {
				continue
			}

			patchContent := []byte("[{\"op\":\"add\",\"path\":\"/metadata/labels/tars.io~1Deleting\",\"value\":\"Deleting\"}]")
			_, err = clients.CrdClient.CrdV1alpha1().TConfigs(namespace).Patch(context.TODO(), name, patchTypes.JSONPatchType, patchContent, k8sMetaV1.PatchOptions{})
			if err != nil {
				err = fmt.Errorf(meta.ResourcePatchError, "tconfig", namespace, name, err.Error())
				utilRuntime.HandleError(err)
				return err
			}
		}
		return nil
	}

	var willMarkDeletingTConfig []string

	for _, tconfigRuntimeObj := range tconfigRuntimeObjs {
		tconfigObj := tconfigRuntimeObj.(k8sMetaV1.Object)
		name := tconfigObj.GetName()

		if name == tconfig.Name {
			continue
		}

		tconfigObjLabels := tconfigObj.GetLabels()
		if tconfigObjLabels == nil {
			err := fmt.Errorf(meta.ShouldNotHappenError, fmt.Sprintf("resource %s %s%s labels value is nil", "tconfig", namespace, tconfigObj.GetName()))
			utilRuntime.HandleError(err)
			return err
		}

		podSeq, ok := tconfigObjLabels[meta.TConfigPodSeqLabel]
		if !ok || len(podSeq) == 0 {
			err := fmt.Errorf(meta.ShouldNotHappenError, fmt.Sprintf("resource %s %s%s labels[%s] value is nil", "tconfig", namespace, tconfigObj.GetName(), meta.TConfigPodSeqLabel))
			utilRuntime.HandleError(err)
			return err
		}

		if podSeq != "m" {
			err = fmt.Errorf("cannot delete tconfig %s/%s because it is reference by anther tconfig", namespace, tconfig.Name)
			utilRuntime.HandleError(err)
			return err
		}

		willMarkDeletingTConfig = append(willMarkDeletingTConfig, name)
	}

	for _, name := range willMarkDeletingTConfig {
		patchContent := []byte("[{\"op\":\"add\",\"path\":\"/metadata/labels/tars.io~1Deleting\",\"value\":\"Deleting\"}]")
		_, err = clients.CrdClient.CrdV1alpha1().TConfigs(namespace).Patch(context.TODO(), name, patchTypes.JSONPatchType, patchContent, k8sMetaV1.PatchOptions{})
		if err != nil {
			err = fmt.Errorf(meta.ResourcePatchError, "tconfig", namespace, name, err.Error())
			utilRuntime.HandleError(err)
			return err
		}
	}

	return nil
}

func validCreateTConfig(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	newTConfig := &crdV1alpha1.TConfig{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTConfig)

	if _, ok := newTConfig.Labels[meta.TConfigDeactivateLabel]; ok {
		return fmt.Errorf("can not set label [tars.io/Deactivate] when create")
	}

	if _, ok := newTConfig.Labels["tars.io/Deleting"]; ok {
		return fmt.Errorf("can not set label [tars.io/Deleting] when create")
	}

	if newTConfig.PodSeq != nil {
		appRequirement, _ := labels.NewRequirement(meta.TServerAppLabel, "==", []string{newTConfig.App})
		serverRequirement, _ := labels.NewRequirement(meta.TServerNameLabel, "==", []string{newTConfig.Server})
		configNameRequirement, _ := labels.NewRequirement(meta.TConfigNameLabel, "==", []string{newTConfig.ConfigName})
		podSeqRequirement, _ := labels.NewRequirement(meta.TConfigPodSeqLabel, "==", []string{"m"})
		activatedRequirement, _ := labels.NewRequirement(meta.TConfigActivatedLabel, "==", []string{"true"})
		deactivatingRequirement, _ := labels.NewRequirement(meta.TConfigDeactivateLabel, "!", []string{})
		deletingRequirement, _ := labels.NewRequirement("tars.io/Deleting", "!", []string{})
		labelSelector := labels.NewSelector().Add(*appRequirement, *serverRequirement, *configNameRequirement, *configNameRequirement, *podSeqRequirement, *activatedRequirement, *deactivatingRequirement, *deletingRequirement)

		namespace := newTConfig.Namespace
		tconfigRuntimeObjs, err := informer.TConfigInformer.Lister().ByNamespace(namespace).List(labelSelector)

		if err != nil {
			if errors.IsNotFound(err) {
				return fmt.Errorf("no activated master tconfig found")
			}
			err = fmt.Errorf(meta.ResourceSelectorError, namespace, "tconfig", err.Error())
			utilRuntime.HandleError(err)
			return err
		}

		if len(tconfigRuntimeObjs) == 0 {
			return fmt.Errorf("no activated master tconfig found")
		}
	}

	if newTConfig.Activated == true {
		return prepareActiveTConfig(newTConfig, clients, informer)
	}
	return nil
}

func validUpdateTConfig(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	if view.Request.UserInfo.Username == meta.GetControllerUsername() {
		return nil
	}
	newTConfig := &crdV1alpha1.TConfig{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTConfig)
	oldTConfig := &crdV1alpha1.TConfig{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTConfig)

	if _, ok := oldTConfig.Labels["tars.io/Deleting"]; ok {
		return fmt.Errorf("can not update deleting tconfig")
	}

	if newTConfig.App != oldTConfig.App {
		return fmt.Errorf(meta.FiledImmutableError, "tconfig", "/app")
	}
	if newTConfig.Server != oldTConfig.Server {
		return fmt.Errorf(meta.FiledImmutableError, "tconfig", "/server")
	}
	if !equality.Semantic.DeepEqual(newTConfig.PodSeq, oldTConfig.PodSeq) {
		return fmt.Errorf(meta.FiledImmutableError, "tconfig", "/podSeq")
	}
	if newTConfig.ConfigName != oldTConfig.ConfigName {
		return fmt.Errorf(meta.FiledImmutableError, "tconfig", "/configName")
	}
	if newTConfig.ConfigContent != oldTConfig.ConfigContent {
		return fmt.Errorf(meta.FiledImmutableError, "tconfig", "/configContent")
	}
	if newTConfig.Version != oldTConfig.Version {
		return fmt.Errorf(meta.FiledImmutableError, "tconfig", "/version")
	}
	if newTConfig.UpdateTime != oldTConfig.UpdateTime {
		return fmt.Errorf(meta.FiledImmutableError, "TConfig", "/updateTime")
	}
	if newTConfig.UpdatePerson != oldTConfig.UpdatePerson {
		return fmt.Errorf(meta.FiledImmutableError, "TConfig", "/updatePerson")
	}
	if newTConfig.UpdateReason != oldTConfig.UpdateReason {
		return fmt.Errorf(meta.FiledImmutableError, "TConfig", "/updateReason")
	}

	if !newTConfig.Activated && oldTConfig.Activated {
		return fmt.Errorf("only use authorized account can update /activated from true to false")
	}

	if !oldTConfig.Activated && newTConfig.Activated {
		return prepareActiveTConfig(newTConfig, clients, informer)
	}

	return nil
}

func validDeleteTConfig(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {

	if view.Request.UserInfo.Username == meta.GetControllerUsername() {
		return nil
	}

	if view.Request.UserInfo.Username == meta.GarbageCollectorAccount {
		return nil
	}

	tconfig := &crdV1alpha1.TConfig{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, tconfig)

	if !tconfig.Activated {
		return nil
	}

	if _, ok := tconfig.Labels[meta.TConfigDeactivateLabel]; ok {
		return nil
	}

	if _, ok := tconfig.Labels["tars.io/Deleting"]; ok {
		return nil
	}

	return prepareDeleteTConfig(tconfig, clients, informer)
}

func validTTemplate(newTTemplate *crdV1alpha1.TTemplate, oldTTemplate *crdV1alpha1.TTemplate, clients *meta.Clients, informer *meta.Informers) error {

	parentName := newTTemplate.Spec.Parent
	if parentName == "" {
		return fmt.Errorf(meta.ResourceInvalidError, "ttemplate", "value of filed \"/spec/parent\" should not empty ")
	}

	if newTTemplate.Name == newTTemplate.Spec.Parent {
		return nil
	}

	namespace := newTTemplate.Namespace

	if _, err := informer.TTemplateInformer.Lister().TTemplates(namespace).Get(parentName); err != nil {
		return fmt.Errorf(meta.ResourceGetError, "ttemplate", namespace, parentName, err.Error())
	}

	return nil
}

func validCreateTTemplate(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	newTTemplate := &crdV1alpha1.TTemplate{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTTemplate)
	return validTTemplate(newTTemplate, nil, clients, informer)
}

func validUpdateTTemplate(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	newTTemplate := &crdV1alpha1.TTemplate{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTTemplate)

	oldTTemplate := &crdV1alpha1.TTemplate{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTTemplate)

	return validTTemplate(newTTemplate, oldTTemplate, clients, informer)
}

func validDeleteTTemplate(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	ttemplate := &crdV1alpha1.TTemplate{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, ttemplate)
	requirement, _ := labels.NewRequirement(meta.TemplateLabel, "==", []string{ttemplate.Name})
	namespace := ttemplate.Namespace
	tServers, err := informer.TServerInformer.Lister().TServers(namespace).List(labels.NewSelector().Add(*requirement))
	if err != nil {
		utilRuntime.HandleError(err)
		return err
	}
	if tServers != nil && len(tServers) != 0 {
		return fmt.Errorf("cannot delete ttemplate %s/%s because it is reference by some tserver", namespace, view.Request.Name)
	}
	return nil
}

func validTImage(newImage *crdV1alpha1.TImage, oldImage *crdV1alpha1.TImage, clients *meta.Clients, informer *meta.Informers) error {
	newTImageVersionMap := make(map[string]*crdV1alpha1.TImageRelease, len(newImage.Releases))
	for _, pos := range newImage.Releases {
		if _, ok := newTImageVersionMap[pos.ID]; ok {
			return fmt.Errorf("duplicate id value : %s", pos.ID)
		}
		newTImageVersionMap[pos.ID] = pos
	}

	if oldImage == nil {
		return nil
	}

	for _, pos := range oldImage.Releases {
		releaseInNewTImage, ok := newTImageVersionMap[pos.ID]
		if ok {
			if pos.ID != releaseInNewTImage.ID {
				return fmt.Errorf(meta.FiledImmutableError, "timage", ".release.id")
			}
			if pos.Image != releaseInNewTImage.Image {
				return fmt.Errorf(meta.FiledImmutableError, "timage", ".release.image")
			}
			if !pos.CreateTime.IsZero() && pos.CreateTime != releaseInNewTImage.CreateTime {
				return fmt.Errorf(meta.FiledImmutableError, "timage", ".release.createTime")
			}
		}
	}
	return nil
}

func validCreateTImage(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	newTImage := &crdV1alpha1.TImage{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTImage)
	return validTImage(newTImage, nil, clients, informer)
}

func validUpdateTImage(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	newTImage := &crdV1alpha1.TImage{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTImage)

	oldTImage := &crdV1alpha1.TImage{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTImage)

	return validTImage(newTImage, oldTImage, clients, informer)
}

func validDeleteTImage(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}

func validTTree(newTTree *crdV1alpha1.TTree, oldTTree *crdV1alpha1.TTree, clients *meta.Clients, informer *meta.Informers) error {
	namespace := newTTree.Namespace

	businessMap := make(map[string]interface{}, len(newTTree.Businesses))
	for _, business := range newTTree.Businesses {
		if _, ok := businessMap[business.Name]; ok {
			return fmt.Errorf(meta.ResourceInvalidError, "ttree", fmt.Sprintf("duplicate business name : %s", business.Name))
		}
		businessMap[business.Name] = nil
	}

	appMap := make(map[string]interface{}, len(newTTree.Apps))
	for _, app := range newTTree.Apps {
		if _, ok := appMap[app.Name]; ok {
			return fmt.Errorf(meta.ResourceInvalidError, "ttree", fmt.Sprintf("duplicate app name : %s", app.Name))
		}
		if app.BusinessRef != "" {
			if _, ok := businessMap[app.BusinessRef]; !ok {
				return fmt.Errorf(meta.ResourceInvalidError, "ttree", fmt.Sprintf("business/%s not exist", app.BusinessRef))
			}
		}
		appMap[app.Name] = nil
	}

	if oldTTree == nil {
		return nil
	}

	for i := range oldTTree.Apps {
		appName := oldTTree.Apps[i].Name
		if _, ok := appMap[appName]; !ok {
			requirement, _ := labels.NewRequirement(meta.TServerAppLabel, "==", []string{appName})
			tservers, err := informer.TServerInformer.Lister().TServers(namespace).List(labels.NewSelector().Add(*requirement))
			if err != nil {
				utilRuntime.HandleError(err)
				return err
			}
			if tservers != nil && len(tservers) != 0 {
				return fmt.Errorf(meta.ResourceInvalidError, "ttree", fmt.Sprintf("cannot delete ttree/apps[%s] because it is reference by some tserver", appName))
			}
		}
	}
	return nil
}

func validCreateTTree(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	newTTree := &crdV1alpha1.TTree{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTTree)

	if newTTree.Name != meta.TTreeResourceName {
		return fmt.Errorf("create ttree operation is defined")
	}

	namespace := newTTree.Namespace

	_, err := informer.TTreeInformer.Lister().TTrees(namespace).Get(meta.TTreeResourceName)
	if err == nil {
		return fmt.Errorf("create ttree operation is defined")
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("create ttree operation is defined")
	}

	return validTTree(newTTree, nil, clients, informer)
}

func validUpdateTTree(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	if view.Request.UserInfo.Username != meta.GetControllerUsername() {
		return nil
	}
	newTTree := &crdV1alpha1.TTree{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTTree)

	oldTTree := &crdV1alpha1.TTree{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTTree)

	return validTTree(newTTree, oldTTree, clients, informer)
}

func validDeleteTTree(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	newTTree := &crdV1alpha1.TTree{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTTree)

	if newTTree.Name == meta.TTreeResourceName {
		if view.Request.UserInfo.Username ==
			meta.GetHelmFinalizerUsername(view.Request.Namespace) {
			return nil
		}
		return fmt.Errorf("delete ttree operation is defined")
	}
	return nil
}

func validTAccount(newTAccount *crdV1alpha1.TAccount, oldTAccount *crdV1alpha1.TAccount, client *meta.Clients, informer *meta.Informers) error {
	expectedResourceName := fmt.Sprintf("%x", md5.Sum([]byte(newTAccount.Spec.Username)))
	if newTAccount.Name != expectedResourceName {
		return fmt.Errorf(meta.ResourceInvalidError, "taccount", "unexpected resource name")
	}
	return nil
}

func validCreateTAccount(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	newTAccount := &crdV1alpha1.TAccount{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTAccount)
	return validTAccount(newTAccount, nil, clients, informer)
}

func validUpdateTAccount(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	if view.Request.UserInfo.Username != meta.GetControllerUsername() {
		return nil
	}
	newTAccount := &crdV1alpha1.TAccount{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTAccount)

	oldTAccount := &crdV1alpha1.TAccount{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTAccount)

	return validTAccount(newTAccount, oldTAccount, clients, informer)
}

func validDeleteTAccount(clients *meta.Clients, informer *meta.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}

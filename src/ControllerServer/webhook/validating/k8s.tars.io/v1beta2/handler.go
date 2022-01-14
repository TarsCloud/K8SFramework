package v1beta2

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
	"k8s.io/apimachinery/pkg/selection"
	patchTypes "k8s.io/apimachinery/pkg/types"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	crdV1beta2 "k8s.tars.io/api/crd/v1beta2"
	crdMeta "k8s.tars.io/api/meta"
	"strings"
	"tarscontroller/controller"
)

var functions = map[string]func(*controller.Clients, *controller.Informers, *k8sAdmissionV1.AdmissionReview) error{}

func init() {
	functions = map[string]func(*controller.Clients, *controller.Informers, *k8sAdmissionV1.AdmissionReview) error{

		"CREATE/TDeploy": validCreateTDeploy,
		"UPDATE/TDeploy": validUpdateTDeploy,
		"DELETE/TDeploy": validDeleteTDeploy,

		"CREATE/TServer": validCreateTServer,
		"UPDATE/TServer": validUpdateTServer,
		"DELETE/TServer": validDeleteTServer,

		"CREATE/TConfig": validCreateTConfig,
		"UPDATE/TConfig": validUpdateTConfig,
		"DELETE/TConfig": validDeleteTConfig,

		"CREATE/TTemplate": validCreateTTemplate,
		"UPDATE/TTemplate": validUpdateTTemplate,
		"DELETE/TTemplate": validDeleteTTemplate,

		"CREATE/TTree": validCreateTTree,
		"UPDATE/TTree": validUpdateTTree,
		"DELETE/TTree": validDeleteTTree,

		"CREATE/TAccount": validCreateTAccount,
		"UPDATE/TAccount": validUpdateTAccount,
		"DELETE/TAccount": validDeleteTAccount,

		"CREATE/TFrameworkConfig": validCreateTFrameworkConfig,
		"UPDATE/TFrameworkConfig": validUpdateTFrameworkConfig,
		"DELETE/TFrameworkConfig": validDeleteTFrameworkConfig,
	}
}

type Handler struct {
	clients   *controller.Clients
	informers *controller.Informers
}

func New(clients *controller.Clients, informers *controller.Informers) *Handler {
	return &Handler{clients: clients, informers: informers}
}

func (v *Handler) Handle(view *k8sAdmissionV1.AdmissionReview) error {
	key := fmt.Sprintf("%s/%s", view.Request.Operation, view.Request.Kind.Kind)
	if fun, ok := functions[key]; ok {
		return fun(v.clients, v.informers, view)
	}
	return fmt.Errorf("unsupported validating %s %s.%s", view.Request.Operation, view.Request.Kind.Version, view.Request.Kind.Kind)
}

func validCreateTDeploy(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	tdeploy := &crdV1beta2.TDeploy{}
	_ = json.Unmarshal(view.Request.Object.Raw, tdeploy)

	if tdeploy.Approve != nil {
		return fmt.Errorf("should not set /approve field when create tdeploy resource")
	}

	if tdeploy.Deployed != nil {
		return fmt.Errorf("should not set /deployed field when create tdeploy resource")
	}

	return nil
}

func validUpdateTDeploy(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == crdMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}

	newTDeploy := &crdV1beta2.TDeploy{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTDeploy)

	oldTDeploy := &crdV1beta2.TDeploy{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTDeploy)

	if !equality.Semantic.DeepEqual(newTDeploy.Deployed, oldTDeploy.Deployed) {
		return fmt.Errorf("only use authorized account can set \"/deployed\" field")
	}

	if oldTDeploy.Approve != nil {
		if !equality.Semantic.DeepEqual(newTDeploy.Apply, oldTDeploy.Apply) {
			return fmt.Errorf(crdMeta.FiledImmutableError, "tserver", "/apply")
		}
		if !equality.Semantic.DeepEqual(newTDeploy.Approve, oldTDeploy.Approve) {
			return fmt.Errorf(crdMeta.FiledImmutableError, "tserver", "/approve")
		}

		return nil
	}

	if newTDeploy.Approve == nil || newTDeploy.Approve.Result == false {
		return nil
	}

	namespace := newTDeploy.Namespace

	tserverName := fmt.Sprintf("%s-%s", strings.ToLower(newTDeploy.Apply.App), strings.ToLower(newTDeploy.Apply.Server))
	_, err := informers.TServerInformer.Lister().TServers(namespace).Get(tserverName)

	if err == nil {
		return fmt.Errorf(crdMeta.ResourceExistError, "tserver", namespace, tserverName)
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf(crdMeta.ResourceGetError, "tserver", namespace, tserverName, err.Error())
	}

	fakeTServer := &crdV1beta2.TServer{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      tserverName,
			Namespace: namespace,
		},
		Spec: newTDeploy.Apply,
	}
	return validTServer(fakeTServer, nil, clients, informers)
}

func validDeleteTDeploy(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}

func validTServer(newTServer, oldTServer *crdV1beta2.TServer, clients *controller.Clients, informers *controller.Informers) error {

	if oldTServer != nil {
		if newTServer.Spec.App != oldTServer.Spec.App {
			return fmt.Errorf(crdMeta.FiledImmutableError, "tserver", ".spec.app")
		}

		if newTServer.Spec.Server != oldTServer.Spec.Server {
			return fmt.Errorf(crdMeta.FiledImmutableError, "tserver", ".spec.server")
		}

		if newTServer.Spec.SubType != oldTServer.Spec.SubType {
			return fmt.Errorf(crdMeta.FiledImmutableError, "tserver", ".spec.subType")
		}

		if oldTServer.Spec.Tars == nil {
			if newTServer.Spec.Tars != nil {
				return fmt.Errorf(crdMeta.FiledImmutableError, "tserver", ".spec.tars")
			}
		}

		if oldTServer.Spec.Normal == nil {
			if newTServer.Spec.Normal != nil {
				return fmt.Errorf(crdMeta.FiledImmutableError, "tserver", ".spec.normal")
			}
		}
	}

	namespace := newTServer.Namespace

	if newTServer.Name != strings.ToLower(newTServer.Spec.App)+"-"+strings.ToLower(newTServer.Spec.Server) {
		return fmt.Errorf(crdMeta.ResourceInvalidError, "tserver", "unexpected resource name")
	}

	portNames := map[string]interface{}{}
	portValues := map[int32]interface{}{}

	if newTServer.Spec.Tars != nil {

		for _, servant := range newTServer.Spec.Tars.Servants {
			portName := strings.ToLower(servant.Name)
			portValue := servant.Port

			if portName == crdMeta.NodeServantName {
				return fmt.Errorf(crdMeta.ResourceInvalidError, "tserver", fmt.Sprintf("servants name value should not equal %s", crdMeta.NodeServantName))
			}

			if portValue == crdMeta.NodeServantPort {
				return fmt.Errorf(crdMeta.ResourceInvalidError, "tserver", fmt.Sprintf("servants port value should not equal %d", crdMeta.NodeServantPort))
			}

			if _, ok := portNames[portName]; ok {
				return fmt.Errorf(crdMeta.ResourceInvalidError, "tserver", fmt.Sprintf("duplicate servant name value %s", servant.Name))
			}

			if _, ok := portValues[portValue]; ok {
				return fmt.Errorf(crdMeta.ResourceInvalidError, "tserver", fmt.Sprintf("duplicate port value %d", servant.Port))
			}

			portNames[portName] = nil
			portValues[portValue] = nil
		}

		for _, port := range newTServer.Spec.Tars.Ports {
			portName := strings.ToLower(port.Name)
			portValue := port.Port

			if portName == crdMeta.NodeServantName {
				return fmt.Errorf(crdMeta.ResourceInvalidError, "tserver", fmt.Sprintf("port name value should not equal %s", crdMeta.NodeServantName))
			}

			if portValue == crdMeta.NodeServantPort {
				return fmt.Errorf(crdMeta.ResourceInvalidError, "tserver", fmt.Sprintf("port value should not equal %d", crdMeta.NodeServantPort))
			}

			if _, ok := portNames[portName]; ok {
				return fmt.Errorf(crdMeta.ResourceInvalidError, "tserver", fmt.Sprintf("duplicate port name value %s", port.Name))
			}

			if _, ok := portValues[portValue]; ok {
				return fmt.Errorf(crdMeta.ResourceInvalidError, "tserver", fmt.Sprintf("duplicate port value %d", port.Port))
			}
			portNames[portName] = nil
			portValues[portValue] = nil
		}

		templateName := newTServer.Spec.Tars.Template
		_, err := informers.TTemplateInformer.Lister().TTemplates(namespace).Get(templateName)
		if err != nil {
			if !errors.IsNotFound(err) {
				return fmt.Errorf(crdMeta.ResourceGetError, "ttemplate", namespace, templateName, err.Error())
			}
			return fmt.Errorf(crdMeta.ResourceNotExistError, "ttemplate", namespace, templateName)
		}
	} else if newTServer.Spec.Normal != nil {
		for _, port := range newTServer.Spec.Normal.Ports {
			portName := strings.ToLower(port.Name)
			portValue := port.Port

			if _, ok := portNames[portName]; ok {
				return fmt.Errorf(crdMeta.ResourceInvalidError, "tserver", fmt.Sprintf("duplicate port name value %s", port.Name))
			}

			if _, ok := portValues[portValue]; ok {
				return fmt.Errorf(crdMeta.ResourceInvalidError, "tserver", fmt.Sprintf("duplicate port value %d", port.Port))
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
				return fmt.Errorf(crdMeta.ResourceInvalidError, "tserver", fmt.Sprintf("port name %s not exist", hostPort.NameRef))
			}

			if _, ok := hostPortNameRefs[nameRef]; ok {
				return fmt.Errorf(crdMeta.ResourceInvalidError, "tserver", fmt.Sprintf("duplicate nameRef value %s", hostPort.NameRef))
			}

			if _, ok := hostPortPorts[hostPort.Port]; ok {
				return fmt.Errorf(crdMeta.ResourceInvalidError, "tserver", fmt.Sprintf("duplicate port value %d", hostPort.Port))
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
				return fmt.Errorf(crdMeta.ResourceInvalidError, "tserver", fmt.Sprintf("duplicate .mounts.name value %s", mount.Name))
			}

			if mount.Source.TLocalVolume != nil || mount.Source.PersistentVolumeClaimTemplate != nil {
				if newTServer.Spec.K8S.DaemonSet {
					return fmt.Errorf(crdMeta.ResourceInvalidError, "tserver", fmt.Sprintf("can not use TLocalVolue and PersistentVolumeClaimTemplate when .daemonSet value is true"))
				}
			}

			mountsNames[mount.Name] = nil
		}
	}
	return nil
}

func validCreateTServer(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	newTServer := &crdV1beta2.TServer{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTServer)
	return validTServer(newTServer, nil, clients, informers)
}

func validUpdateTServer(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	newTServer := &crdV1beta2.TServer{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTServer)

	oldTServer := &crdV1beta2.TServer{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTServer)

	return validTServer(newTServer, oldTServer, clients, informers)
}

func validDeleteTServer(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}

func prepareActiveTConfig(newTConfig *crdV1beta2.TConfig, clients *controller.Clients, informers *controller.Informers) error {
	namespace := newTConfig.Namespace

	appRequirement, _ := labels.NewRequirement(crdMeta.TServerAppLabel, selection.DoubleEquals, []string{newTConfig.App})
	serverRequirement, _ := labels.NewRequirement(crdMeta.TServerNameLabel, selection.DoubleEquals, []string{newTConfig.Server})
	configNameRequirement, _ := labels.NewRequirement(crdMeta.TConfigNameLabel, selection.DoubleEquals, []string{newTConfig.ConfigName})
	podSeqRequirement, _ := labels.NewRequirement(crdMeta.TConfigPodSeqLabel, selection.DoubleEquals, []string{newTConfig.PodSeq})
	activateRequirement, _ := labels.NewRequirement(crdMeta.TConfigActivatedLabel, selection.DoubleEquals, []string{"true"})

	labelSelector := labels.NewSelector().Add(*appRequirement).Add(*serverRequirement).Add(*configNameRequirement).Add(*podSeqRequirement).Add(*activateRequirement)

	tconfigs, err := informers.TConfigInformer.Lister().ByNamespace(namespace).List(labelSelector)
	if err != nil {
		err = fmt.Errorf(crdMeta.ResourceSelectorError, namespace, "tconfig", err.Error())
		utilRuntime.HandleError(err)
		return err
	}

	jsonPatch := crdMeta.JsonPatch{
		crdMeta.JsonPatchItem{
			OP:    crdMeta.JsonPatchAdd,
			Path:  "/metadata/labels/tars.io~1Deactivate",
			Value: "Deactivating",
		},
	}
	patchContent, _ := json.Marshal(jsonPatch)
	for _, tconfig := range tconfigs {
		v := tconfig.(k8sMetaV1.Object)
		name := v.GetName()

		if name == newTConfig.Name {
			continue
		}

		_, err = clients.CrdClient.CrdV1beta2().TConfigs(namespace).Patch(context.TODO(), name, patchTypes.JSONPatchType, patchContent, k8sMetaV1.PatchOptions{})
		if err != nil {
			err = fmt.Errorf(crdMeta.ResourcePatchError, "tconfig", namespace, name, err.Error())
			utilRuntime.HandleError(err)
			return err
		}
	}
	return nil
}

func prepareDeleteTConfig(tconfig *crdV1beta2.TConfig, clients *controller.Clients, informers *controller.Informers) error {
	appRequirement, _ := labels.NewRequirement(crdMeta.TServerAppLabel, selection.DoubleEquals, []string{tconfig.App})
	serverRequirement, _ := labels.NewRequirement(crdMeta.TServerNameLabel, selection.DoubleEquals, []string{tconfig.Server})
	configNameRequirement, _ := labels.NewRequirement(crdMeta.TConfigNameLabel, selection.DoubleEquals, []string{tconfig.ConfigName})

	var labelSelector labels.Selector
	if tconfig.PodSeq == "m" {
		labelSelector = labels.NewSelector().Add(*appRequirement).Add(*serverRequirement).Add(*configNameRequirement)
	} else {
		podSeqRequirement, _ := labels.NewRequirement(crdMeta.TConfigPodSeqLabel, selection.DoubleEquals, []string{tconfig.PodSeq})
		labelSelector = labels.NewSelector().Add(*podSeqRequirement)
	}

	namespace := tconfig.Namespace
	tconfigRuntimeObjects, err := informers.TConfigInformer.Lister().ByNamespace(namespace).List(labelSelector)

	if err != nil && !errors.IsNotFound(err) {
		err = fmt.Errorf(crdMeta.ResourceSelectorError, namespace, "tconfig", err.Error())
		utilRuntime.HandleError(err)
		return err
	}

	jsonPatch := crdMeta.JsonPatch{
		crdMeta.JsonPatchItem{
			OP:    crdMeta.JsonPatchAdd,
			Path:  "/metadata/labels/tars.io~1Deleting",
			Value: "Deleting",
		},
	}
	patchContent, _ := json.Marshal(jsonPatch)

	if tconfig.PodSeq != "m" {
		for _, tconfigRuntimeObj := range tconfigRuntimeObjects {
			tconfigObj := tconfigRuntimeObj.(k8sMetaV1.Object)
			name := tconfigObj.GetName()

			if name == tconfig.Name {
				continue
			}

			_, err = clients.CrdClient.CrdV1beta2().TConfigs(namespace).Patch(context.TODO(), name, patchTypes.JSONPatchType, patchContent, k8sMetaV1.PatchOptions{})
			if err != nil {
				err = fmt.Errorf(crdMeta.ResourcePatchError, "tconfig", namespace, name, err.Error())
				utilRuntime.HandleError(err)
				return err
			}
		}
		return nil
	}

	var willMarkDeletingTConfig []string

	for _, tconfigRuntimeObj := range tconfigRuntimeObjects {
		tconfigObj := tconfigRuntimeObj.(k8sMetaV1.Object)
		name := tconfigObj.GetName()

		if name == tconfig.Name {
			continue
		}

		tconfigObjLabels := tconfigObj.GetLabels()
		if tconfigObjLabels == nil {
			err = fmt.Errorf(crdMeta.ShouldNotHappenError, fmt.Sprintf("resource %s %s%s labels value is nil", "tconfig", namespace, tconfigObj.GetName()))
			utilRuntime.HandleError(err)
			return err
		}

		podSeq, ok := tconfigObjLabels[crdMeta.TConfigPodSeqLabel]
		if !ok || len(podSeq) == 0 {
			err = fmt.Errorf(crdMeta.ShouldNotHappenError, fmt.Sprintf("resource %s %s%s labels[%s] value is nil", "tconfig", namespace, tconfigObj.GetName(), crdMeta.TConfigPodSeqLabel))
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
		_, err = clients.CrdClient.CrdV1beta2().TConfigs(namespace).Patch(context.TODO(), name, patchTypes.JSONPatchType, patchContent, k8sMetaV1.PatchOptions{})
		if err != nil {
			err = fmt.Errorf(crdMeta.ResourcePatchError, "tconfig", namespace, name, err.Error())
			utilRuntime.HandleError(err)
			return err
		}
	}

	return nil
}

func validCreateTConfig(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	newTConfig := &crdV1beta2.TConfig{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTConfig)

	if _, ok := newTConfig.Labels[crdMeta.TConfigDeactivateLabel]; ok {
		return fmt.Errorf("can not set label [%s] when create", crdMeta.TConfigDeactivateLabel)
	}

	if _, ok := newTConfig.Labels[crdMeta.TConfigDeletingLabel]; ok {
		return fmt.Errorf("can not set label [%s] when create", crdMeta.TConfigDeletingLabel)
	}

	if newTConfig.PodSeq != "m" {
		appRequirement, _ := labels.NewRequirement(crdMeta.TServerAppLabel, selection.DoubleEquals, []string{newTConfig.App})
		serverRequirement, _ := labels.NewRequirement(crdMeta.TServerNameLabel, selection.DoubleEquals, []string{newTConfig.Server})
		configNameRequirement, _ := labels.NewRequirement(crdMeta.TConfigNameLabel, selection.DoubleEquals, []string{newTConfig.ConfigName})
		podSeqRequirement, _ := labels.NewRequirement(crdMeta.TConfigPodSeqLabel, selection.DoubleEquals, []string{"m"})
		activatedRequirement, _ := labels.NewRequirement(crdMeta.TConfigActivatedLabel, selection.DoubleEquals, []string{"true"})
		deactivatingRequirement, _ := labels.NewRequirement(crdMeta.TConfigDeactivateLabel, selection.DoesNotExist, []string{})
		deletingRequirement, _ := labels.NewRequirement(crdMeta.TConfigDeletingLabel, selection.DoesNotExist, []string{})
		labelSelector := labels.NewSelector().Add(*appRequirement, *serverRequirement, *configNameRequirement, *configNameRequirement, *podSeqRequirement, *activatedRequirement, *deactivatingRequirement, *deletingRequirement)

		namespace := newTConfig.Namespace
		tconfigRuntimeObjects, err := informers.TConfigInformer.Lister().ByNamespace(namespace).List(labelSelector)

		if err != nil {
			if errors.IsNotFound(err) {
				return fmt.Errorf("no activated master tconfig found")
			}
			err = fmt.Errorf(crdMeta.ResourceSelectorError, namespace, "tconfig", err.Error())
			utilRuntime.HandleError(err)
			return err
		}

		if len(tconfigRuntimeObjects) == 0 {
			return fmt.Errorf("no activated master tconfig found")
		}
	}

	if newTConfig.Activated {
		return prepareActiveTConfig(newTConfig, clients, informers)
	}
	return nil
}

func validUpdateTConfig(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == crdMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}
	newTConfig := &crdV1beta2.TConfig{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTConfig)
	oldTConfig := &crdV1beta2.TConfig{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTConfig)

	if _, ok := oldTConfig.Labels["tars.io/Deleting"]; ok {
		return fmt.Errorf("can not update deleting tconfig")
	}

	if newTConfig.App != oldTConfig.App {
		return fmt.Errorf(crdMeta.FiledImmutableError, "tconfig", "/app")
	}
	if newTConfig.Server != oldTConfig.Server {
		return fmt.Errorf(crdMeta.FiledImmutableError, "tconfig", "/server")
	}
	if !equality.Semantic.DeepEqual(newTConfig.PodSeq, oldTConfig.PodSeq) {
		return fmt.Errorf(crdMeta.FiledImmutableError, "tconfig", "/podSeq")
	}
	if newTConfig.ConfigName != oldTConfig.ConfigName {
		return fmt.Errorf(crdMeta.FiledImmutableError, "tconfig", "/configName")
	}
	if newTConfig.ConfigContent != oldTConfig.ConfigContent {
		return fmt.Errorf(crdMeta.FiledImmutableError, "tconfig", "/configContent")
	}
	if newTConfig.Version != oldTConfig.Version {
		return fmt.Errorf(crdMeta.FiledImmutableError, "tconfig", "/version")
	}
	if newTConfig.UpdateTime != oldTConfig.UpdateTime {
		return fmt.Errorf(crdMeta.FiledImmutableError, "TConfig", "/updateTime")
	}
	if newTConfig.UpdatePerson != oldTConfig.UpdatePerson {
		return fmt.Errorf(crdMeta.FiledImmutableError, "TConfig", "/updatePerson")
	}
	if newTConfig.UpdateReason != oldTConfig.UpdateReason {
		return fmt.Errorf(crdMeta.FiledImmutableError, "TConfig", "/updateReason")
	}

	if !newTConfig.Activated && oldTConfig.Activated {
		return fmt.Errorf("only use authorized account can update /activated from true to false")
	}

	if !oldTConfig.Activated && newTConfig.Activated {
		return prepareActiveTConfig(newTConfig, clients, informers)
	}

	return nil
}

func validDeleteTConfig(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {

	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == crdMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}

	if view.Request.UserInfo.Username == crdMeta.GarbageCollectorAccount {
		return nil
	}

	tconfig := &crdV1beta2.TConfig{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, tconfig)

	if !tconfig.Activated {
		return nil
	}

	if _, ok := tconfig.Labels[crdMeta.TConfigDeactivateLabel]; ok {
		return nil
	}

	if _, ok := tconfig.Labels["tars.io/Deleting"]; ok {
		return nil
	}

	return prepareDeleteTConfig(tconfig, clients, informers)
}

func validTTemplate(newTTemplate *crdV1beta2.TTemplate, oldTTemplate *crdV1beta2.TTemplate, clients *controller.Clients, informers *controller.Informers) error {

	parentName := newTTemplate.Spec.Parent
	if parentName == "" {
		return fmt.Errorf(crdMeta.ResourceInvalidError, "ttemplate", "value of filed \"/spec/parent\" should not empty ")
	}

	if newTTemplate.Name == newTTemplate.Spec.Parent {
		return nil
	}

	namespace := newTTemplate.Namespace

	if _, err := informers.TTemplateInformer.Lister().TTemplates(namespace).Get(parentName); err != nil {
		return fmt.Errorf(crdMeta.ResourceGetError, "ttemplate", namespace, parentName, err.Error())
	}

	return nil
}

func validCreateTTemplate(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	newTTemplate := &crdV1beta2.TTemplate{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTTemplate)
	return validTTemplate(newTTemplate, nil, clients, informers)
}

func validUpdateTTemplate(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	newTTemplate := &crdV1beta2.TTemplate{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTTemplate)

	oldTTemplate := &crdV1beta2.TTemplate{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTTemplate)

	return validTTemplate(newTTemplate, oldTTemplate, clients, informers)
}

func validDeleteTTemplate(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {

	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == crdMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}

	ttemplate := &crdV1beta2.TTemplate{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, ttemplate)
	requirement, _ := labels.NewRequirement(crdMeta.TemplateLabel, selection.DoubleEquals, []string{ttemplate.Name})
	namespace := ttemplate.Namespace
	tServers, err := informers.TServerInformer.Lister().TServers(namespace).List(labels.NewSelector().Add(*requirement))
	if err != nil {
		utilRuntime.HandleError(err)
		return err
	}
	if tServers != nil && len(tServers) != 0 {
		return fmt.Errorf("cannot delete ttemplate %s/%s because it is reference by some tserver", namespace, view.Request.Name)
	}
	return nil
}

func validTTree(newTTree *crdV1beta2.TTree, oldTTree *crdV1beta2.TTree, clients *controller.Clients, informers *controller.Informers) error {
	namespace := newTTree.Namespace

	businessMap := make(map[string]interface{}, len(newTTree.Businesses))
	for _, business := range newTTree.Businesses {
		if _, ok := businessMap[business.Name]; ok {
			return fmt.Errorf(crdMeta.ResourceInvalidError, "ttree", fmt.Sprintf("duplicate business name : %s", business.Name))
		}
		businessMap[business.Name] = nil
	}

	appMap := make(map[string]interface{}, len(newTTree.Apps))
	for _, app := range newTTree.Apps {
		if _, ok := appMap[app.Name]; ok {
			return fmt.Errorf(crdMeta.ResourceInvalidError, "ttree", fmt.Sprintf("duplicate app name : %s", app.Name))
		}
		if app.BusinessRef != "" {
			if _, ok := businessMap[app.BusinessRef]; !ok {
				return fmt.Errorf(crdMeta.ResourceInvalidError, "ttree", fmt.Sprintf("business/%s not exist", app.BusinessRef))
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
			requirement, _ := labels.NewRequirement(crdMeta.TServerAppLabel, selection.DoubleEquals, []string{appName})
			tservers, err := informers.TServerInformer.Lister().TServers(namespace).List(labels.NewSelector().Add(*requirement))
			if err != nil {
				utilRuntime.HandleError(err)
				return err
			}
			if tservers != nil && len(tservers) != 0 {
				return fmt.Errorf(crdMeta.ResourceInvalidError, "ttree", fmt.Sprintf("cannot delete ttree/apps[%s] because it is reference by some tserver", appName))
			}
		}
	}
	return nil
}

func validCreateTTree(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	newTTree := &crdV1beta2.TTree{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTTree)

	if newTTree.Name != crdMeta.FixedTTreeResourceName {
		return fmt.Errorf("create ttree operation is defined")
	}

	namespace := newTTree.Namespace

	_, err := informers.TTreeInformer.Lister().TTrees(namespace).Get(crdMeta.FixedTTreeResourceName)
	if err == nil {
		return fmt.Errorf("create ttree operation is defined")
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("create ttree operation is defined")
	}

	return validTTree(newTTree, nil, clients, informers)
}

func validUpdateTTree(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == crdMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}
	newTTree := &crdV1beta2.TTree{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTTree)

	oldTTree := &crdV1beta2.TTree{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTTree)

	return validTTree(newTTree, oldTTree, clients, informers)
}

func validDeleteTTree(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == crdMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}

	ttree := &crdV1beta2.TTree{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, ttree)

	if ttree.Name == crdMeta.FixedTTreeResourceName {
		return fmt.Errorf("delete ttree operation is defined")
	}
	return nil
}

func validTAccount(newTAccount *crdV1beta2.TAccount, oldTAccount *crdV1beta2.TAccount, client *controller.Clients, informers *controller.Informers) error {
	expectedResourceName := fmt.Sprintf("%x", md5.Sum([]byte(newTAccount.Spec.Username)))
	if newTAccount.Name != expectedResourceName {
		return fmt.Errorf(crdMeta.ResourceInvalidError, "taccount", "unexpected resource name")
	}
	return nil
}

func validCreateTAccount(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	newTAccount := &crdV1beta2.TAccount{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTAccount)
	return validTAccount(newTAccount, nil, clients, informers)
}

func validUpdateTAccount(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == crdMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}

	newTAccount := &crdV1beta2.TAccount{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTAccount)

	oldTAccount := &crdV1beta2.TAccount{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTAccount)

	return validTAccount(newTAccount, oldTAccount, clients, informers)
}

func validDeleteTAccount(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}

func validTFrameworkConfig(newTFC, oldTFC *crdV1beta2.TFrameworkConfig, clients *controller.Clients, informers *controller.Informers) error {
	return nil
}

func validCreateTFrameworkConfig(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	newTFC := &crdV1beta2.TFrameworkConfig{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTFC)

	if newTFC.Name != crdMeta.FixedTFrameworkConfigResourceName {
		return fmt.Errorf("create tframeworkconfig operation is defined")
	}

	namespace := newTFC.Namespace

	_, err := informers.TFrameworkConfigInformer.Lister().TFrameworkConfigs(namespace).Get(crdMeta.FixedTFrameworkConfigResourceName)
	if err == nil {
		return fmt.Errorf("create tframeworkconfig operation is defined")
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("create tframeworkconfig operation is defined")
	}

	return validTFrameworkConfig(newTFC, nil, clients, informers)
}

func validUpdateTFrameworkConfig(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == crdMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}
	newTFC := &crdV1beta2.TFrameworkConfig{}
	_ = json.Unmarshal(view.Request.Object.Raw, newTFC)

	oldTFC := &crdV1beta2.TFrameworkConfig{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, oldTFC)

	return validTFrameworkConfig(newTFC, oldTFC, clients, informers)
}

func validDeleteTFrameworkConfig(clients *controller.Clients, informers *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == crdMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}

	tfc := &crdV1beta2.TFrameworkConfig{}
	_ = json.Unmarshal(view.Request.OldObject.Raw, tfc)

	if tfc.Name == crdMeta.FixedTFrameworkConfigResourceName {
		return fmt.Errorf("delete tframeworkconfig operation is defined")
	}
	return nil
}

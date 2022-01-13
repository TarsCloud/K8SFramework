package v1

import (
	"encoding/json"
	"fmt"
	k8sAdmissionV1 "k8s.io/api/admission/v1"
	k8sAppsV1 "k8s.io/api/apps/v1"
	crdMeta "k8s.tars.io/api/meta"
	"tarscontroller/controller"
	crdV1beta2 "tarscontroller/reconcile/v1beta2"
)

var functions = map[string]func(*controller.Clients, *controller.Informers, *k8sAdmissionV1.AdmissionReview) error{}

func init() {
	functions = map[string]func(*controller.Clients, *controller.Informers, *k8sAdmissionV1.AdmissionReview) error{
		"CREATE/StatefulSet": validCreateStatefulSet,
		"UPDATE/StatefulSet": validUpdateStatefulSet,
		"DELETE/StatefulSet": validDeleteStatefulSet,

		"CREATE/DaemonSet": validCreateDaemonSet,
		"UPDATE/DaemonSet": validUpdateDaemonSet,
		"DELETE/DaemonSet": validDeleteDaemonSet,
	}
}

type Handler struct {
	clients  *controller.Clients
	informer *controller.Informers
}

func New(clients *controller.Clients, informers *controller.Informers) *Handler {
	return &Handler{clients: clients, informer: informers}
}

func (v *Handler) Handle(view *k8sAdmissionV1.AdmissionReview) error {
	key := fmt.Sprintf("%s/%s", view.Request.Operation, view.Request.Kind.Kind)
	if fun, ok := functions[key]; ok {
		return fun(v.clients, v.informer, view)
	}
	return fmt.Errorf("unsupported validating %s %s.%s", view.Request.Operation, view.Request.Kind.Version, view.Request.Kind.Kind)
}

func validStatefulSet(newStatefulset *k8sAppsV1.StatefulSet, oldStatefulset *k8sAppsV1.StatefulSet, clients *controller.Clients, informer *controller.Informers) error {
	namespace := newStatefulset.Namespace
	tserver, err := informer.TServerInformer.Lister().TServers(namespace).Get(newStatefulset.Name)
	if err != nil {
		return fmt.Errorf(crdMeta.ResourceGetError, "tserver", namespace, newStatefulset.Name, err.Error())
	}
	if !crdV1beta2.EqualTServerAndStatefulSet(tserver, newStatefulset) {
		return fmt.Errorf("resource should be modified through tserver")
	}
	return nil
}

func validCreateStatefulSet(clients *controller.Clients, informer *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == crdMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}
	return fmt.Errorf("only use authorized account can create statefulset")
}

func validUpdateStatefulSet(clients *controller.Clients, informer *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == crdMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}
	newStatefulset := &k8sAppsV1.StatefulSet{}
	_ = json.Unmarshal(view.Request.Object.Raw, newStatefulset)
	return validStatefulSet(newStatefulset, nil, clients, informer)
}

func validDeleteStatefulSet(clients *controller.Clients, informer *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}

func validDaemonset(newDaemonset, oldDaemonset *k8sAppsV1.DaemonSet, clients *controller.Clients, informer *controller.Informers) error {
	namespace := newDaemonset.Namespace
	tserver, err := informer.TServerInformer.Lister().TServers(namespace).Get(newDaemonset.Name)
	if err != nil {
		return fmt.Errorf(crdMeta.ResourceGetError, "tserver", namespace, newDaemonset.Name, err.Error())
	}
	if !crdV1beta2.EqualTServerAndDaemonSet(tserver, newDaemonset) {
		return fmt.Errorf("this resource should be modified through tserver")
	}
	return nil
}

func validCreateDaemonSet(clients *controller.Clients, informer *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == crdMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}
	return fmt.Errorf("only use authorized account can create daemonset")
}

func validUpdateDaemonSet(clients *controller.Clients, informer *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	controllerUserName := controller.GetControllerUsername()
	if controllerUserName == view.Request.UserInfo.Username || controllerUserName == crdMeta.DefaultUnlawfulAndOnlyForDebugUserName {
		return nil
	}
	newDaemonset := &k8sAppsV1.DaemonSet{}
	_ = json.Unmarshal(view.Request.Object.Raw, newDaemonset)
	return validDaemonset(newDaemonset, nil, clients, informer)
}

func validDeleteDaemonSet(clients *controller.Clients, informer *controller.Informers, view *k8sAdmissionV1.AdmissionReview) error {
	return nil
}

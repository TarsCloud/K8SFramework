package v1beta3

import (
	"context"
	"fmt"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	patchTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sWatchV1 "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/integer"
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
	tarsMetaTools "k8s.tars.io/meta/tools"
	tarsMetaV1beta3 "k8s.tars.io/meta/v1beta3"
	"strings"
	"tarscontroller/controller"
	"tarscontroller/reconcile"
	"time"
)

type TExitedRecordReconciler struct {
	clients   *controller.Clients
	informers *controller.Informers
	threads   int
	workQueue workqueue.RateLimitingInterface
}

func NewTExitedPodReconciler(clients *controller.Clients, informers *controller.Informers, threads int) *TExitedRecordReconciler {
	reconciler := &TExitedRecordReconciler{
		clients:   clients,
		informers: informers,
		threads:   threads,
		workQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), ""),
	}
	informers.Register(reconciler)
	return reconciler
}

func (r *TExitedRecordReconciler) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *tarsCrdV1beta3.TServer:
		tserver := resourceObj.(*tarsCrdV1beta3.TServer)
		key := fmt.Sprintf("%s/%s", tserver.Namespace, tserver.Name)
		r.workQueue.Add(key)
	case *tarsCrdV1beta3.TExitedRecord:
		texitedRecord := resourceObj.(*tarsCrdV1beta3.TExitedRecord)
		key := fmt.Sprintf("%s/%s", texitedRecord.Namespace, texitedRecord.Name)
		r.workQueue.Add(key)
	case *k8sCoreV1.Pod:
		pod := resourceObj.(*k8sCoreV1.Pod)
		if pod.DeletionTimestamp != nil && pod.UID != "" && pod.Labels != nil {
			app, appExist := pod.Labels[tarsMetaV1beta3.TServerAppLabel]
			server, serverExist := pod.Labels[tarsMetaV1beta3.TServerNameLabel]
			if appExist && serverExist {
				tExitedEvent := &tarsCrdV1beta3.TExitedRecord{
					App:    app,
					Server: server,
					Pods: []tarsCrdV1beta3.TExitedPod{
						{
							UID:        string(pod.UID),
							Name:       pod.Name,
							ID:         pod.Labels[tarsMetaV1beta3.TServerIdLabel],
							NodeIP:     pod.Status.HostIP,
							PodIP:      pod.Status.PodIP,
							CreateTime: pod.CreationTimestamp,
							DeleteTime: *pod.DeletionTimestamp,
						},
					},
				}
				bs, _ := json.Marshal(tExitedEvent)
				key := fmt.Sprintf("%s/event/%s", pod.Namespace, bs)
				r.workQueue.Add(key)
				return
			}
		}
	default:
		return
	}
}

func (r *TExitedRecordReconciler) splitKey(key string) []string {
	return strings.Split(key, "/")
}

func (r *TExitedRecordReconciler) processItem() bool {

	obj, shutdown := r.workQueue.Get()

	if shutdown {
		return false
	}

	defer r.workQueue.Done(obj)

	key, ok := obj.(string)
	if !ok {
		utilRuntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
		r.workQueue.Forget(obj)
		return true
	}

	var res reconcile.Result
	v := r.splitKey(key)
	if len(v) == 2 {
		res = r.reconcileBaseTServer(v[0], v[1])
	} else {
		res = r.reconcileBasePod(v[0], v[2])
	}

	switch res {
	case reconcile.AllOk:
		r.workQueue.Forget(obj)
		return true
	case reconcile.RateLimit:
		r.workQueue.AddRateLimited(obj)
		return true
	case reconcile.FatalError:
		r.workQueue.ShutDown()
		return false
	default:
		//code should not reach here
		utilRuntime.HandleError(fmt.Errorf("should not reach place"))
		return false
	}
}

func (r *TExitedRecordReconciler) Start(stopCh chan struct{}) {
	for i := 0; i < r.threads; i++ {
		workFun := func() {
			for r.processItem() {
			}
			r.workQueue.ShutDown()
		}
		go wait.Until(workFun, time.Second, stopCh)
	}
}

func (r *TExitedRecordReconciler) reconcileBaseTServer(namespace string, name string) reconcile.Result {
	tserver, err := r.informers.TServerInformer.Lister().TServers(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMetaV1beta3.ResourceGetError, "tserver", namespace, name, err.Error()))
			return reconcile.RateLimit
		}
		err = r.clients.CrdClient.CrdV1beta3().TExitedRecords(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMetaV1beta3.ResourceDeleteError, "texitedrecord", namespace, name, err.Error()))
			return reconcile.RateLimit
		}
		return reconcile.AllOk
	}

	if tserver.DeletionTimestamp != nil {
		err = r.clients.CrdClient.CrdV1beta3().TExitedRecords(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMetaV1beta3.ResourceDeleteError, "texitedrecord", namespace, name, err.Error()))
			return reconcile.RateLimit
		}
		return reconcile.AllOk
	}

	tExitedRecord, err := r.informers.TExitedRecordInformer.Lister().TExitedRecords(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMetaV1beta3.ResourceGetError, "texitedrecord", namespace, name, err.Error()))
			return reconcile.RateLimit
		}
		tExitedRecord = buildTExitedRecord(tserver)
		tExitedPodInterface := r.clients.CrdClient.CrdV1beta3().TExitedRecords(namespace)
		if _, err = tExitedPodInterface.Create(context.TODO(), tExitedRecord, k8sMetaV1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMetaV1beta3.ResourceCreateError, "texitedrecord", namespace, name, err.Error()))
			return reconcile.RateLimit
		}
		return reconcile.AllOk
	}
	return reconcile.AllOk
}

func (r *TExitedRecordReconciler) reconcileBasePod(namespace string, tExitedPodSpecString string) reconcile.Result {
	var tExitedEvent tarsCrdV1beta3.TExitedRecord
	_ = json.Unmarshal([]byte(tExitedPodSpecString), &tExitedEvent)

	tExitedRecordName := fmt.Sprintf("%s-%s", strings.ToLower(tExitedEvent.App), strings.ToLower(tExitedEvent.Server))
	tExitedRecord, err := r.informers.TExitedRecordInformer.Lister().TExitedRecords(namespace).Get(tExitedRecordName)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(tarsMetaV1beta3.ResourceGetError, "texitedrecord", namespace, tExitedRecordName, err.Error()))
			return reconcile.RateLimit
		}
		return reconcile.AllOk
	}

	recordedPodsLen := len(tExitedRecord.Pods)

	const DefaultMaxCheckLen = 12
	maxCheckLen := integer.IntMin(DefaultMaxCheckLen, recordedPodsLen)

	for i := 0; i < maxCheckLen; i++ {
		if tExitedRecord.Pods[i].UID == tExitedEvent.Pods[0].UID {
			// means exited events had recorded
			return reconcile.AllOk
		}
	}

	jsonPatch := tarsMetaTools.JsonPatch{
		{
			OP:    tarsMetaTools.JsonPatchAdd,
			Path:  "/pods/0",
			Value: tExitedEvent.Pods[0],
		},
	}

	recordsLimit := tarsMetaV1beta3.DefaultMaxRecordLen

	if tfc := controller.GetTFrameworkConfig(namespace); tfc != nil {
		recordsLimit = tfc.RecordLimit.TExitedPod
	}

	if recordedPodsLen >= recordsLimit {
		jsonPatch = append(jsonPatch, tarsMetaTools.JsonPatchItem{
			OP:   tarsMetaTools.JsonPatchRemove,
			Path: fmt.Sprintf("/pods/%d", recordedPodsLen),
		})
	}

	patchContent, _ := json.Marshal(jsonPatch)
	_, err = r.clients.CrdClient.CrdV1beta3().TExitedRecords(namespace).Patch(context.TODO(), tExitedRecordName, patchTypes.JSONPatchType, patchContent, k8sMetaV1.PatchOptions{})

	if err != nil {
		utilRuntime.HandleError(fmt.Errorf(tarsMetaV1beta3.ResourcePatchError, "texitedrecord", namespace, tExitedRecordName, err.Error()))
		return reconcile.RateLimit
	}

	return reconcile.AllOk
}

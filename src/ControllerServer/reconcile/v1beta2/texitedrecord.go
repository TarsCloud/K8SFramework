package v1beta2

import (
	"context"
	"encoding/json"
	"fmt"
	k8sCoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	patchTypes "k8s.io/apimachinery/pkg/types"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sWatchV1 "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/integer"
	crdV1beta2 "k8s.tars.io/api/crd/v1beta2"
	crdMeta "k8s.tars.io/api/meta"
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
	case *crdV1beta2.TServer:
		tserver := resourceObj.(*crdV1beta2.TServer)
		key := fmt.Sprintf("%s/%s", tserver.Namespace, tserver.Name)
		r.workQueue.Add(key)
	case *crdV1beta2.TExitedRecord:
		texitedRecord := resourceObj.(*crdV1beta2.TExitedRecord)
		key := fmt.Sprintf("%s/%s", texitedRecord.Namespace, texitedRecord.Name)
		r.workQueue.Add(key)
	case *k8sCoreV1.Pod:
		pod := resourceObj.(*k8sCoreV1.Pod)
		if pod.DeletionTimestamp != nil && pod.UID != "" && pod.Labels != nil {
			app, appExist := pod.Labels[crdMeta.TServerAppLabel]
			server, serverExist := pod.Labels[crdMeta.TServerNameLabel]
			if appExist && serverExist {
				tExitedEvent := &crdV1beta2.TExitedRecord{
					App:    app,
					Server: server,
					Pods: []crdV1beta2.TExitedPod{
						{
							UID:        string(pod.UID),
							Name:       pod.Name,
							ID:         pod.Labels[crdMeta.TServerIdLabel],
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
			utilRuntime.HandleError(fmt.Errorf(crdMeta.ResourceGetError, "tserver", namespace, name, err.Error()))
			return reconcile.RateLimit
		}
		err = r.clients.CrdClient.CrdV1beta2().TExitedRecords(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(crdMeta.ResourceDeleteError, "texitedrecord", namespace, name, err.Error()))
			return reconcile.RateLimit
		}
		return reconcile.AllOk
	}

	if tserver.DeletionTimestamp != nil {
		err = r.clients.CrdClient.CrdV1beta2().TExitedRecords(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(crdMeta.ResourceDeleteError, "texitedrecord", namespace, name, err.Error()))
			return reconcile.RateLimit
		}
		return reconcile.AllOk
	}

	tExitedRecord, err := r.informers.TExitedRecordInformer.Lister().TExitedRecords(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(crdMeta.ResourceGetError, "texitedrecord", namespace, name, err.Error()))
			return reconcile.RateLimit
		}
		tExitedRecord = buildTExitedRecord(tserver)
		tExitedPodInterface := r.clients.CrdClient.CrdV1beta2().TExitedRecords(namespace)
		if _, err = tExitedPodInterface.Create(context.TODO(), tExitedRecord, k8sMetaV1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
			utilRuntime.HandleError(fmt.Errorf(crdMeta.ResourceCreateError, "texitedrecord", namespace, name, err.Error()))
			return reconcile.RateLimit
		}
		return reconcile.AllOk
	}
	return reconcile.AllOk
}

func (r *TExitedRecordReconciler) reconcileBasePod(namespace string, tExitedPodSpecString string) reconcile.Result {
	var tExitedEvent crdV1beta2.TExitedRecord
	_ = json.Unmarshal([]byte(tExitedPodSpecString), &tExitedEvent)

	tExitedRecordName := fmt.Sprintf("%s-%s", strings.ToLower(tExitedEvent.App), strings.ToLower(tExitedEvent.Server))
	tExitedRecord, err := r.informers.TExitedRecordInformer.Lister().TExitedRecords(namespace).Get(tExitedRecordName)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(crdMeta.ResourceGetError, "texitedrecord", namespace, tExitedRecordName, err.Error()))
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

	jsonPatch := crdMeta.JsonPatch{
		{
			OP:    crdMeta.JsonPatchAdd,
			Path:  "/pods/0",
			Value: tExitedEvent.Pods[0],
		},
	}

	recordsLimit := crdMeta.DefaultMaxRecordLen

	if tfc := controller.GetTFrameworkConfig(namespace); tfc != nil {
		recordsLimit = tfc.RecordLimit.TExitedPod
	}

	if recordedPodsLen >= recordsLimit {
		jsonPatch = append(jsonPatch, crdMeta.JsonPatchItem{
			OP:   crdMeta.JsonPatchRemove,
			Path: "/pods/-",
		})
	}

	patchContent, _ := json.Marshal(jsonPatch)
	_, err = r.clients.CrdClient.CrdV1beta2().TExitedRecords(namespace).Patch(context.TODO(), tExitedRecordName, patchTypes.JSONPatchType, patchContent, k8sMetaV1.PatchOptions{})

	if err != nil {
		utilRuntime.HandleError(fmt.Errorf(crdMeta.ResourcePatchError, "texitedrecord", namespace, tExitedRecordName, err.Error()))
		return reconcile.RateLimit
	}

	return reconcile.AllOk
}

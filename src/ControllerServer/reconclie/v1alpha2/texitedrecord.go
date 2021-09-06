package v1alpha2

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
	crdV1alpha2 "k8s.tars.io/api/crd/v1alpha2"
	"strings"
	"tarscontroller/meta"
	"tarscontroller/reconclie"
	"time"
)

type TExitedRecordReconciler struct {
	clients   *meta.Clients
	informers *meta.Informers
	threads   int
	workQueue workqueue.RateLimitingInterface
}

func NewTExitedPodReconciler(clients *meta.Clients, informers *meta.Informers, threads int) *TExitedRecordReconciler {
	reconcile := &TExitedRecordReconciler{
		clients:   clients,
		informers: informers,
		threads:   threads,
		workQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), ""),
	}
	informers.Register(reconcile)
	return reconcile
}

func (r *TExitedRecordReconciler) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *crdV1alpha2.TServer:
		tserver := resourceObj.(*crdV1alpha2.TServer)
		key := fmt.Sprintf("%s/%s", tserver.Namespace, tserver.Name)
		r.workQueue.Add(key)
	case *crdV1alpha2.TExitedRecord:
		texitedRecord := resourceObj.(*crdV1alpha2.TExitedRecord)
		key := fmt.Sprintf("%s/%s", texitedRecord.Namespace, texitedRecord.Name)
		r.workQueue.Add(key)
	case *k8sCoreV1.Pod:
		if !meta.IsControllerLeader() {
			return
		}
		pod := resourceObj.(*k8sCoreV1.Pod)
		if pod.DeletionTimestamp != nil && pod.UID != "" && pod.Labels != nil {
			app, appExist := pod.Labels[meta.TServerAppLabel]
			server, serverExist := pod.Labels[meta.TServerNameLabel]
			if appExist && serverExist {
				tExitedEvent := &crdV1alpha2.TExitedRecord{
					App:    app,
					Server: server,
					Pods: []crdV1alpha2.TExitedPod{
						{
							UID:        string(pod.UID),
							Name:       pod.Name,
							ID:         pod.Labels[meta.TServerIdLabel],
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

	var res reconclie.ReconcileResult
	v := r.splitKey(key)
	if len(v) == 2 {
		res = r.reconcileBaseTServer(v[0], v[1])
	} else {
		res = r.reconcileBasePod(v[0], v[2])
	}

	switch res {
	case reconclie.AllOk:
		r.workQueue.Forget(obj)
		return true
	case reconclie.RateLimit:
		r.workQueue.AddRateLimited(obj)
		return true
	case reconclie.FatalError:
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

func (r *TExitedRecordReconciler) reconcileBaseTServer(namespace string, name string) reconclie.ReconcileResult {
	tserver, err := r.informers.TServerInformer.Lister().TServers(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceGetError, "tserver", namespace, name, err.Error()))
			return reconclie.RateLimit
		}
		err = r.clients.CrdClient.CrdV1alpha2().TExitedRecords(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceDeleteError, "texitedrecord", namespace, name, err.Error()))
			return reconclie.RateLimit
		}
		return reconclie.AllOk
	}

	if tserver.DeletionTimestamp != nil {
		err = r.clients.CrdClient.CrdV1alpha2().TExitedRecords(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceDeleteError, "texitedrecord", namespace, name, err.Error()))
			return reconclie.RateLimit
		}
		return reconclie.AllOk
	}

	tExitedRecord, err := r.informers.TExitedRecordInformer.Lister().TExitedRecords(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceGetError, "texitedrecord", namespace, name, err.Error()))
			return reconclie.RateLimit
		}
		tExitedRecord = meta.BuildTExitedRecord(tserver)
		tExitedPodInterface := r.clients.CrdClient.CrdV1alpha2().TExitedRecords(namespace)
		if _, err = tExitedPodInterface.Create(context.TODO(), tExitedRecord, k8sMetaV1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceCreateError, "texitedrecord", namespace, name, err.Error()))
			return reconclie.RateLimit
		}
		return reconclie.AllOk
	}
	return reconclie.AllOk
}

func (r *TExitedRecordReconciler) reconcileBasePod(namespace string, tExitedPodSpecString string) reconclie.ReconcileResult {
	var tExitedEvent crdV1alpha2.TExitedRecord
	_ = json.Unmarshal([]byte(tExitedPodSpecString), &tExitedEvent)

	tExitedRecordName := fmt.Sprintf("%s-%s", strings.ToLower(tExitedEvent.App), strings.ToLower(tExitedEvent.Server))
	tExitedRecord, err := r.informers.TExitedRecordInformer.Lister().TExitedRecords(namespace).Get(tExitedRecordName)
	if err != nil {
		if !errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf(meta.ResourceGetError, "texitedrecord", namespace, tExitedRecordName, err.Error()))
			return reconclie.RateLimit
		}
		return reconclie.AllOk
	}

	recordedPodsLen := len(tExitedRecord.Pods)

	const DefaultMaxCheckLen = 12
	maxCheckLen := DefaultMaxCheckLen

	if recordedPodsLen <= maxCheckLen {
		maxCheckLen = recordedPodsLen
	}

	for i := 0; i < maxCheckLen; i++ {
		if tExitedRecord.Pods[i].UID == tExitedEvent.Pods[0].UID {
			// means exited events had recorded
			return reconclie.AllOk
		}
	}

	const DefaultMaxRecordLen = 60
	var patchString string

	exitedPodString, _ := json.Marshal(tExitedEvent.Pods[0])

	if recordedPodsLen < DefaultMaxRecordLen {
		patchString = fmt.Sprintf("[{\"op\":\"add\",\"path\":\"/pods/0\",\"value\":%s}]", exitedPodString)
	} else {
		patchString = fmt.Sprintf("[{\"op\":\"remove\",\"path\":\"/pods/%d\"},{\"op\":\"add\",\"path\":\"/pods/0\",\"value\":%s}]", recordedPodsLen-1, exitedPodString)
	}

	_, err = r.clients.CrdClient.CrdV1alpha2().TExitedRecords(namespace).Patch(context.TODO(), tExitedRecordName, patchTypes.JSONPatchType, []byte(patchString), k8sMetaV1.PatchOptions{})

	if err != nil {
		utilRuntime.HandleError(fmt.Errorf(meta.ResourcePatchError, "texitedrecord", namespace, tExitedRecordName, err.Error()))
		return reconclie.RateLimit
	}

	return reconclie.AllOk
}

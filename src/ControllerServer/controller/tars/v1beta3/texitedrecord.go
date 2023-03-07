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
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"k8s.io/utils/integer"
	tarsV1beta3 "k8s.tars.io/apis/tars/v1beta3"
	tarsListerV1beta3 "k8s.tars.io/client-go/listers/tars/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"
	tarsTool "k8s.tars.io/tool"
	"strings"
	"tarscontroller/controller"
	"time"
)

type TExitedRecordReconciler struct {
	teLister tarsListerV1beta3.TExitedRecordLister
	tsLister tarsListerV1beta3.TServerLister
	threads  int
	queue    workqueue.RateLimitingInterface
	synced   []cache.InformerSynced
}

func NewTExitedPodController(threads int) *TExitedRecordReconciler {
	podInformer := tarsRuntime.Factories.K8SInformerFactoryWithTarsFilter.Core().V1().Pods()
	teInformer := tarsRuntime.Factories.TarsInformerFactory.Tars().V1beta3().TExitedRecords()
	tsInformer := tarsRuntime.Factories.TarsInformerFactory.Tars().V1beta3().TServers()
	c := &TExitedRecordReconciler{
		teLister: teInformer.Lister(),
		tsLister: tsInformer.Lister(),
		threads:  threads,
		queue:    workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		synced:   []cache.InformerSynced{podInformer.Informer().HasSynced, teInformer.Informer().HasSynced, tsInformer.Informer().HasSynced},
	}
	controller.RegistryInformerEventHandle(tarsMeta.KPodKind, podInformer.Informer(), c)
	controller.RegistryInformerEventHandle(tarsMeta.TExitedRecordKind, teInformer.Informer(), c)
	controller.RegistryInformerEventHandle(tarsMeta.TServerKind, tsInformer.Informer(), c)
	return c
}

func (r *TExitedRecordReconciler) EnqueueResourceEvent(resourceKind string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *tarsV1beta3.TServer:
		tserver := resourceObj.(*tarsV1beta3.TServer)
		key := fmt.Sprintf("%s/%s", tserver.Namespace, tserver.Name)
		r.queue.Add(key)
	case *tarsV1beta3.TExitedRecord:
		texitedRecord := resourceObj.(*tarsV1beta3.TExitedRecord)
		key := fmt.Sprintf("%s/%s", texitedRecord.Namespace, texitedRecord.Name)
		r.queue.Add(key)
	case *k8sCoreV1.Pod:
		pod := resourceObj.(*k8sCoreV1.Pod)
		if pod.DeletionTimestamp != nil && pod.UID != "" && pod.Labels != nil {
			app := pod.Labels[tarsMeta.TServerAppLabel]
			server := pod.Labels[tarsMeta.TServerNameLabel]
			if app != "" && server != "" {
				tExitedEvent := &tarsV1beta3.TExitedRecord{
					App:    app,
					Server: server,
					Pods: []tarsV1beta3.TExitedPod{
						{
							UID:        string(pod.UID),
							Name:       pod.Name,
							ID:         pod.Labels[tarsMeta.TServerIdLabel],
							NodeIP:     pod.Status.HostIP,
							PodIP:      pod.Status.PodIP,
							CreateTime: pod.CreationTimestamp,
							DeleteTime: *pod.DeletionTimestamp,
						},
					},
				}
				bs, _ := json.Marshal(tExitedEvent)
				key := fmt.Sprintf("%s/event/%s", pod.Namespace, bs)
				r.queue.Add(key)
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

	obj, shutdown := r.queue.Get()

	if shutdown {
		return false
	}

	defer r.queue.Done(obj)

	key, ok := obj.(string)
	if !ok {
		klog.Errorf("expected string in workqueue but got %#v", obj)
		r.queue.Forget(obj)
		return true
	}

	var res controller.Result
	v := r.splitKey(key)
	if len(v) == 2 {
		res = r.reconcileBaseTServer(v[0], v[1])
	} else {
		res = r.reconcileBasePod(v[0], v[2])
	}

	switch res {
	case controller.Done:
		r.queue.Forget(obj)
		return true
	case controller.Retry:
		r.queue.AddRateLimited(obj)
		return true
	case controller.FatalError:
		r.queue.ShutDown()
		return false
	default:
		//code should not reach here
		klog.Errorf("should not reach place")
		return false
	}
}

func (r *TExitedRecordReconciler) Run(stopCh chan struct{}) {
	defer utilRuntime.HandleCrash()
	defer r.queue.ShutDown()

	if !cache.WaitForNamedCacheSync("texitedrecord controller", stopCh, r.synced...) {
		return
	}

	for i := 0; i < r.threads; i++ {
		worker := func() {
			for r.processItem() {
			}
			r.queue.ShutDown()
		}
		go wait.Until(worker, time.Second, stopCh)
	}

	<-stopCh
}

func (r *TExitedRecordReconciler) reconcileBaseTServer(namespace string, name string) controller.Result {
	tserver, err := r.tsLister.TServers(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf(tarsMeta.ResourceGetError, "tserver", namespace, name, err.Error())
			return controller.Retry
		}
		err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TExitedRecords(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			klog.Errorf(tarsMeta.ResourceDeleteError, "texitedrecord", namespace, name, err.Error())
			return controller.Retry
		}
		return controller.Done
	}

	if tserver.DeletionTimestamp != nil {
		err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TExitedRecords(namespace).Delete(context.TODO(), name, k8sMetaV1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			klog.Errorf(tarsMeta.ResourceDeleteError, "texitedrecord", namespace, name, err.Error())
			return controller.Retry
		}
		return controller.Done
	}

	tExitedRecord, err := r.teLister.TExitedRecords(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf(tarsMeta.ResourceGetError, "texitedrecord", namespace, name, err.Error())
			return controller.Retry
		}
		tExitedRecord = tarsRuntime.TarsTranslator.BuildTExitedRecord(tserver)
		tExitedPodInterface := tarsRuntime.Clients.CrdClient.TarsV1beta3().TExitedRecords(namespace)
		if _, err = tExitedPodInterface.Create(context.TODO(), tExitedRecord, k8sMetaV1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
			klog.Errorf(tarsMeta.ResourceCreateError, "texitedrecord", namespace, name, err.Error())
			return controller.Retry
		}
		return controller.Done
	}
	return controller.Done
}

func (r *TExitedRecordReconciler) reconcileBasePod(namespace string, tExitedPodSpecString string) controller.Result {
	var tExitedEvent tarsV1beta3.TExitedRecord
	_ = json.Unmarshal([]byte(tExitedPodSpecString), &tExitedEvent)

	tExitedRecordName := fmt.Sprintf("%s-%s", strings.ToLower(tExitedEvent.App), strings.ToLower(tExitedEvent.Server))
	tExitedRecord, err := r.teLister.TExitedRecords(namespace).Get(tExitedRecordName)
	if err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf(tarsMeta.ResourceGetError, "texitedrecord", namespace, tExitedRecordName, err.Error())
			return controller.Retry
		}
		return controller.Done
	}

	recordedPodsLen := len(tExitedRecord.Pods)

	const DefaultMaxCheckLen = 12
	maxCheckLen := integer.IntMin(DefaultMaxCheckLen, recordedPodsLen)

	for i := 0; i < maxCheckLen; i++ {
		if tExitedRecord.Pods[i].UID == tExitedEvent.Pods[0].UID {
			// means exited events had recorded
			return controller.Done
		}
	}

	jsonPatch := tarsTool.JsonPatch{
		{
			OP:    tarsTool.JsonPatchAdd,
			Path:  "/pods/0",
			Value: tExitedEvent.Pods[0],
		},
	}

	recordsLimit := tarsMeta.DefaultMaxRecordLen

	if tfc := tarsRuntime.TFCConfig.GetTFrameworkConfig(namespace); tfc != nil {
		recordsLimit = tfc.RecordLimit.TExitedPod
	}

	if recordedPodsLen >= recordsLimit {
		jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
			OP:   tarsTool.JsonPatchRemove,
			Path: fmt.Sprintf("/pods/%d", recordedPodsLen),
		})
	}

	patchContent, _ := json.Marshal(jsonPatch)
	_, err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TExitedRecords(namespace).Patch(context.TODO(), tExitedRecordName, patchTypes.JSONPatchType, patchContent, k8sMetaV1.PatchOptions{})

	if err != nil {
		klog.Errorf(tarsMeta.ResourcePatchError, "texitedrecord", namespace, tExitedRecordName, err.Error())
		return controller.Retry
	}

	return controller.Done
}

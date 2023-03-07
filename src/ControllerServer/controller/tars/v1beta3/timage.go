package v1beta3

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sWatchV1 "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	tarsV1beta3 "k8s.tars.io/apis/tars/v1beta3"
	tarsListerV1beta3 "k8s.tars.io/client-go/listers/tars/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"
	tarsTool "k8s.tars.io/tool"
	"strings"
	"tarscontroller/controller"
	"time"
)

const reconcileTargetCheckImageBuildOvertime = "CHECK_BUILD_OVERTIME"

type TImageReconciler struct {
	tiLister tarsListerV1beta3.TImageLister
	threads  int
	queue    workqueue.RateLimitingInterface
	synced   []cache.InformerSynced
}

func NewTImageController(threads int) *TImageReconciler {
	tiInformer := tarsRuntime.Factories.TarsInformerFactory.Tars().V1beta3().TImages()
	c := &TImageReconciler{
		tiLister: tiInformer.Lister(),
		threads:  threads,
		queue:    workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		synced:   []cache.InformerSynced{tiInformer.Informer().HasSynced},
	}
	controller.RegistryInformerEventHandle(tarsMeta.TServerKind, tiInformer.Informer(), c)
	return c
}

func (r *TImageReconciler) EnqueueResourceEvent(resourceKind string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *tarsV1beta3.TImage:
		timage := resourceObj.(*tarsV1beta3.TImage)
		if timage.ImageType == "node" {
		}
		if timage.ImageType == "server" {
			if timage.Build != nil && timage.Build.Running != nil {
				maxBuildTime := tarsMeta.DefaultMaxImageBuildTime
				if tfc := tarsRuntime.TFCConfig.GetTFrameworkConfig(timage.Namespace); tfc != nil {
					maxBuildTime = tfc.ImageBuild.MaxBuildTime
				}
				key := fmt.Sprintf("%s/%s/%s/%s", timage.Namespace, timage.Name, reconcileTargetCheckImageBuildOvertime, timage.Build.Running.ID)
				r.queue.AddAfter(key, time.Duration(maxBuildTime)*time.Second)
			}
		}
	default:
		return
	}
}

func (r *TImageReconciler) processItem() bool {

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

	res := r.reconcile(key)
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

func (r *TImageReconciler) Run(stopCh chan struct{}) {
	defer utilRuntime.HandleCrash()
	defer r.queue.ShutDown()

	if !cache.WaitForNamedCacheSync("timage controller", stopCh, r.synced...) {
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

func (r *TImageReconciler) splitKey(key string) (namespace, name, target, value string) {
	v := strings.Split(key, "/")
	return v[0], v[1], v[2], v[3]
}

func (r *TImageReconciler) reconcile(key string) controller.Result {
	namespace, name, target, value := r.splitKey(key)
	timage, err := r.tiLister.TImages(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return controller.Done
		}
		klog.Errorf(tarsMeta.ResourceGetError, "timage", namespace, name, err.Error())
		return controller.Retry
	}

	var jsonPatch tarsTool.JsonPatch

	switch target {
	case reconcileTargetCheckImageBuildOvertime:
		if timage.Build == nil || timage.Build.Running == nil || value != timage.Build.Running.ID {
			return controller.Done
		}
		buildState := tarsV1beta3.TImageBuild{
			Last: &tarsV1beta3.TImageBuildState{
				ID:              timage.Build.Running.ID,
				BaseImage:       timage.Build.Running.BaseImage,
				BaseImageSecret: timage.Build.Running.BaseImageSecret,
				Image:           timage.Build.Running.Image,
				Secret:          timage.Build.Running.Secret,
				ServerType:      timage.Build.Running.ServerType,
				CreatePerson:    timage.Build.Running.CreatePerson,
				CreateTime:      timage.Build.Running.CreateTime,
				Mark:            timage.Build.Running.Mark,
				Phase:           "Failed",
				Message:         "task overtime",
			},
		}
		jsonPatch = append(jsonPatch, tarsTool.JsonPatchItem{
			OP:    tarsTool.JsonPatchAdd,
			Path:  "/build",
			Value: buildState,
		})
	}

	if jsonPatch != nil {
		bs, _ := json.Marshal(jsonPatch)
		_, err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TImages(namespace).Patch(context.TODO(), name, types.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		if err != nil {
			klog.Errorf(err.Error())
			return controller.Retry
		}
	}
	return controller.Done
}

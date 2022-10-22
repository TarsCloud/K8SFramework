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
	tarsCrdListerV1beta3 "k8s.tars.io/client-go/listers/crd/v1beta3"
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"strings"
	"tarscontroller/controller"
	"tarscontroller/util"
	"time"
)

const reconcileTargetCheckImageBuildOvertime = "CHECK_BUILD_OVERTIME"

type TImageReconciler struct {
	clients  *util.Clients
	tiLister tarsCrdListerV1beta3.TImageLister
	threads  int
	queue    workqueue.RateLimitingInterface
	synced   []cache.InformerSynced
}

func NewTImageController(clients *util.Clients, factories *util.InformerFactories, threads int) *TImageReconciler {
	tiInformer := factories.TarsInformerFactory.Crd().V1beta3().TImages()
	tic := &TImageReconciler{
		clients:  clients,
		tiLister: tiInformer.Lister(),
		threads:  threads,
		queue:    workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		synced:   []cache.InformerSynced{tiInformer.Informer().HasSynced},
	}
	controller.SetInformerHandlerEvent(tarsMeta.TServerKind, tiInformer.Informer(), tic)
	return tic
}

func (r *TImageReconciler) EnqueueResourceEvent(resourceKind string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *tarsCrdV1beta3.TImage:
		timage := resourceObj.(*tarsCrdV1beta3.TImage)
		if timage.ImageType == "node" {
		}
		if timage.ImageType == "server" {
			if timage.Build != nil && timage.Build.Running != nil {
				maxBuildTime := tarsMeta.DefaultMaxImageBuildTime
				if tfc := util.GetTFrameworkConfig(timage.Namespace); tfc != nil {
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
		utilRuntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
		r.queue.Forget(obj)
		return true
	}

	res := r.sync(key)
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
		utilRuntime.HandleError(fmt.Errorf("should not reach place"))
		return false
	}
}

func (r *TImageReconciler) StartController(stopCh chan struct{}) {
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

func (r *TImageReconciler) sync(key string) controller.Result {
	namespace, name, target, value := r.splitKey(key)
	timage, err := r.tiLister.TImages(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return controller.Done
		}
		utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceGetError, "timage", namespace, name, err.Error()))
		return controller.Retry
	}

	var jsonPatch tarsMeta.JsonPatch

	switch target {
	case reconcileTargetCheckImageBuildOvertime:
		if timage.Build == nil || timage.Build.Running == nil || value != timage.Build.Running.ID {
			return controller.Done
		}
		buildState := tarsCrdV1beta3.TImageBuild{
			Last: &tarsCrdV1beta3.TImageBuildState{
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
		jsonPatch = append(jsonPatch, tarsMeta.JsonPatchItem{
			OP:    tarsMeta.JsonPatchAdd,
			Path:  "/build",
			Value: buildState,
		})
	}

	if jsonPatch != nil {
		bs, _ := json.Marshal(jsonPatch)
		_, err = r.clients.CrdClient.CrdV1beta3().TImages(namespace).Patch(context.TODO(), name, types.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		if err != nil {
			utilRuntime.HandleError(err)
			return controller.Retry
		}
	}
	return controller.Done
}

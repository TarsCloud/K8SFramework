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
	"k8s.io/client-go/util/workqueue"
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"strings"
	"tarscontroller/controller"
	"tarscontroller/reconcile"
	"time"
)

const reconcileTargetCheckImageBuildOvertime = "CHECK_BUILD_OVERTIME"

type TImageReconciler struct {
	clients   *controller.Clients
	informers *controller.Informers
	threads   int
	workQueue workqueue.RateLimitingInterface
}

func NewTImageReconciler(clients *controller.Clients, informers *controller.Informers, threads int) *TImageReconciler {
	reconciler := &TImageReconciler{
		clients:   clients,
		informers: informers,
		threads:   threads,
		workQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), ""),
	}
	informers.Register(reconciler)
	return reconciler
}

func (r *TImageReconciler) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *tarsCrdV1beta3.TImage:
		timage := resourceObj.(*tarsCrdV1beta3.TImage)

		if timage.ImageType == "node" {

		}

		if timage.ImageType == "server" {
			if timage.Build != nil && timage.Build.Running != nil {
				maxBuildTime := tarsMeta.DefaultMaxImageBuildTime
				if tfc := controller.GetTFrameworkConfig(timage.Namespace); tfc != nil {
					maxBuildTime = tfc.ImageBuild.MaxBuildTime
				}
				key := fmt.Sprintf("%s/%s/%s/%s", timage.Namespace, timage.Name, reconcileTargetCheckImageBuildOvertime, timage.Build.Running.ID)
				r.workQueue.AddAfter(key, time.Duration(maxBuildTime)*time.Second)
			}
		}

	default:
		return
	}
}

func (r *TImageReconciler) processItem() bool {

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

	res := r.reconcile(key)
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

func (r *TImageReconciler) Start(stopCh chan struct{}) {
	for i := 0; i < r.threads; i++ {
		workFun := func() {
			for r.processItem() {
			}
			r.workQueue.ShutDown()
		}
		go wait.Until(workFun, time.Second, stopCh)
	}
}

func (r *TImageReconciler) splitKey(key string) (namespace, name, target, value string) {
	v := strings.Split(key, "/")
	return v[0], v[1], v[2], v[3]
}

func (r *TImageReconciler) reconcile(key string) reconcile.Result {
	namespace, name, target, value := r.splitKey(key)
	timage, err := r.informers.TImageInformer.Lister().TImages(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.AllOk
		}
		utilRuntime.HandleError(fmt.Errorf(tarsMeta.ResourceGetError, "timage", namespace, name, err.Error()))
		return reconcile.RateLimit
	}

	var jsonPatch tarsMeta.JsonPatch

	switch target {
	case reconcileTargetCheckImageBuildOvertime:
		if timage.Build == nil || timage.Build.Running == nil || value != timage.Build.Running.ID {
			return reconcile.AllOk
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
			return reconcile.RateLimit
		}
	}
	return reconcile.AllOk
}

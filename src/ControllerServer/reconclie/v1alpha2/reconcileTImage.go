package v1alpha2

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

type TImageReconciler struct {
	clients   *meta.Clients
	informers *meta.Informers
	threads   int
	workQueue workqueue.RateLimitingInterface
}

func NewTImageReconciler(clients *meta.Clients, informers *meta.Informers, threads int) *TImageReconciler {
	reconcile := &TImageReconciler{
		clients:   clients,
		informers: informers,
		threads:   threads,
		workQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), ""),
	}
	informers.Register(reconcile)
	return reconcile
}

func (r *TImageReconciler) EnqueueObj(resourceName string, resourceEvent k8sWatchV1.EventType, resourceObj interface{}) {
	switch resourceObj.(type) {
	case *crdV1alpha2.TImage:

		timage := resourceObj.(*crdV1alpha2.TImage)

		//if timage.ImageType == "node" {
		//	for _, release := range timage.Releases {
		//		if strings.HasPrefix(release.ID, "default") {
		//			meta.DefaultNodeImage = release.Image
		//			meta.DefaultNodeImageSecret = release.Secret
		//		}
		//	}
		//	return
		//}

		if timage.ImageType == "server" {
			if timage.Build != nil && timage.Build.Running != nil {
				key := fmt.Sprintf("%s/%s/%s", timage.Namespace, timage.Name, timage.Build.Running.ID)
				r.workQueue.AddAfter(key, time.Minute*10)
			}
			return
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

func (r *TImageReconciler) splitKey(key string) (namespace, name, id string) {
	v := strings.Split(key, "/")
	return v[0], v[1], v[2]
}

func (r *TImageReconciler) reconcile(key string) reconclie.ReconcileResult {
	namespace, name, id := r.splitKey(key)
	timage, err := r.informers.TImageInformer.Lister().TImages(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconclie.AllOk
		}
		utilRuntime.HandleError(fmt.Errorf(meta.ResourceGetError, "timage", namespace, name, err.Error()))
		return reconclie.RateLimit
	}

	if timage.Build == nil || timage.Build.Running == nil || id != timage.Build.Running.ID {
		return reconclie.AllOk
	}

	buildState := crdV1alpha2.TImageBuild{
		Last: &crdV1alpha2.TImageBuildState{
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
			Message:         "running task over time (10m)",
		},
	}

	bs, _ := json.Marshal(buildState)

	patchContent := fmt.Sprintf("[{\"op\":\"add\",\"path\":\"/build\",\"value\":%s}]", bs)

	_, err = r.clients.CrdClient.CrdV1alpha2().TImages(namespace).Patch(context.TODO(), name, types.JSONPatchType, []byte(patchContent), k8sMetaV1.PatchOptions{})

	if err != nil {
		utilRuntime.HandleError(err)
		return reconclie.RateLimit
	}

	return reconclie.AllOk
}

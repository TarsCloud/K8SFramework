package controller

import (
	"context"
	"fmt"
	"io/ioutil"
	k8sCoreV1 "k8s.io/api/core/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	k8sSchema "k8s.io/client-go/kubernetes/scheme"
	k8sCoreTypedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	k8sMetadata "k8s.io/client-go/metadata"
	k8sClientCmd "k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	crdV1beta2 "k8s.tars.io/api/crd/v1beta2"
	crdMeta "k8s.tars.io/api/meta"
	crdVersioned "k8s.tars.io/client-go/clientset/versioned"
	crdScheme "k8s.tars.io/client-go/clientset/versioned/scheme"
	"os"
	"strings"
	"sync"
	"time"
)

var k8sClient kubernetes.Interface
var crdClient crdVersioned.Interface
var k8sMetadataClient k8sMetadata.Interface
var informers *Informers

var tframeworkConfigs map[string]*crdV1beta2.TFrameworkConfig
var tframeworkRWLock sync.RWMutex

var controllerServiceAccount string
var controllerNamespace string
var recorder record.EventRecorder

const TControllerServiceAccount = "tars-controller"

func GetDefaultNodeImage(namespace string) (image string, secret string) {

	var tfc *crdV1beta2.TFrameworkConfig
	var timage *crdV1beta2.TImage
	var err error
	if informers.synced {
		if tfc = GetTFrameworkConfig(namespace); tfc != nil {
			return tfc.NodeImage.Image, tfc.NodeImage.Secret
		}

		timage, err = informers.TImageInformer.Lister().TImages(namespace).Get("node")
		if err == nil && timage != nil {
			for _, release := range timage.Releases {
				if strings.HasPrefix(release.ID, "default") {
					return release.Image, release.Secret
				}
			}
		}
		if err != nil {
			utilRuntime.HandleError(fmt.Errorf("get timage/node err: %s", err.Error()))
		}

		utilRuntime.HandleError(fmt.Errorf("no default node image set"))
		return crdMeta.ServiceImagePlaceholder, ""
	}

	tfc, _ = crdClient.CrdV1beta2().TFrameworkConfigs(namespace).Get(context.TODO(), crdMeta.FixedTFrameworkConfigResourceName, k8sMetaV1.GetOptions{})
	if tfc != nil {
		return tfc.NodeImage.Image, tfc.NodeImage.Secret
	}

	timage, _ = crdClient.CrdV1beta2().TImages(namespace).Get(context.TODO(), "node", k8sMetaV1.GetOptions{})
	if timage != nil {
		for _, release := range timage.Releases {
			if strings.HasPrefix(release.ID, "default") {
				return release.Image, release.Secret
			}
		}
	}

	utilRuntime.HandleError(fmt.Errorf("no default node image set"))
	return crdMeta.ServiceImagePlaceholder, ""
}

func GetControllerUsername() string {
	return controllerServiceAccount
}

func createRecorder(namespace string) {
	if recorder == nil {
		eventBroadcaster := record.NewBroadcaster()
		eventBroadcaster.StartRecordingToSink(&k8sCoreTypedV1.EventSinkImpl{Interface: k8sClient.CoreV1().Events(namespace)})
		recorder = eventBroadcaster.NewRecorder(k8sSchema.Scheme, k8sCoreV1.EventSource{
			Component: "tarscontroller",
			Host:      "",
		})
	}
}

func Event(namespace string, object runtime.Object, eventType, reason, message string) {
	if recorder == nil {
		createRecorder(namespace)
	}
	if recorder != nil {
		recorder.Event(object, eventType, reason, message)
	}
}

func CreateContext(masterUrl, kubeConfigPath string) (*Clients, *Informers, error) {

	clusterConfig, err := k8sClientCmd.BuildConfigFromFlags(masterUrl, kubeConfigPath)

	if err != nil {
		return nil, nil, err
	}

	k8sClient = kubernetes.NewForConfigOrDie(clusterConfig)

	crdClient = crdVersioned.NewForConfigOrDie(clusterConfig)

	k8sMetadataClient = k8sMetadata.NewForConfigOrDie(clusterConfig)

	utilRuntime.Must(crdScheme.AddToScheme(k8sSchema.Scheme))

	clients := &Clients{
		K8sClient:         k8sClient,
		CrdClient:         crdClient,
		K8sMetadataClient: k8sMetadataClient,
	}

	informers = newInformers(clients)

	const namespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	bs, err := ioutil.ReadFile(namespaceFile)
	if err == nil {
		controllerNamespace = string(bs)
	} else {
		utilRuntime.HandleError(fmt.Errorf("cannot read namespace file : %s", err.Error()))
		controllerNamespace = crdMeta.DefaultControllerNamespace
	}

	if masterUrl != "" || kubeConfigPath != "" {
		controllerServiceAccount = crdMeta.DefaultUnlawfulAndOnlyForDebugUserName
	} else {
		controllerServiceAccount = fmt.Sprintf("system:serviceaccount:%s:%s", controllerNamespace, TControllerServiceAccount)
	}
	return clients, informers, nil
}

func getEventRecorder(namespace string) record.EventRecorder {
	if recorder == nil {
		createRecorder(namespace)
	}
	return recorder
}

func LeaderElectAndRun(callbacks leaderelection.LeaderCallbacks) {
	id, err := os.Hostname()
	if err != nil {
		fmt.Printf("GetHostName Error: %s\n", err.Error())
		return
	}
	id = id + "_" + string(uuid.NewUUID())

	rl, err := resourcelock.New("leases",
		controllerNamespace,
		"tarscontroller",
		k8sClient.CoreV1(),
		k8sClient.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: getEventRecorder(controllerNamespace),
		})

	if err != nil {
		fmt.Printf("Create ResourceLock Error: %s\n", err.Error())
		return
	}

	leaderelection.RunOrDie(context.TODO(), leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				if callbacks.OnStartedLeading != nil {
					callbacks.OnStartedLeading(ctx)
				}
			},
			OnStoppedLeading: func() {
				if callbacks.OnStoppedLeading != nil {
					callbacks.OnStoppedLeading()
				}
			},
			OnNewLeader: callbacks.OnNewLeader,
		},
		Name: "tarscontroller",
	})
}

func GetTFrameworkConfig(namespace string) *crdV1beta2.TFrameworkConfig {
	tframeworkRWLock.RLock()
	defer tframeworkRWLock.RUnlock()
	if tframeworkConfigs == nil {
		return nil
	}
	tfc, _ := tframeworkConfigs[namespace]
	return tfc
}

func SetTFrameworkConfig(namespace string, tfc *crdV1beta2.TFrameworkConfig) {
	tframeworkRWLock.Lock()
	defer tframeworkRWLock.Unlock()
	if tframeworkConfigs != nil {
		tframeworkConfigs[namespace] = tfc.DeepCopy()
		return
	}

	tframeworkConfigs = map[string]*crdV1beta2.TFrameworkConfig{
		namespace: tfc.DeepCopy(),
	}
}

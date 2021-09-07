package meta

import (
	"context"
	"fmt"
	"io/ioutil"
	k8sCoreV1 "k8s.io/api/core/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	k8sInformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	k8sSchema "k8s.io/client-go/kubernetes/scheme"
	k8sCoreTypedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	k8sMetadata "k8s.io/client-go/metadata"
	"k8s.io/client-go/metadata/metadatainformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	crdV1beta1 "k8s.tars.io/api/crd/v1beta1"
	crdVersioned "k8s.tars.io/client-go/clientset/versioned"
	crdScheme "k8s.tars.io/client-go/clientset/versioned/scheme"
	crdInformers "k8s.tars.io/client-go/informers/externalversions"
	"os"
	"strings"
	"time"
)

var k8sClient kubernetes.Interface
var crdClient crdVersioned.Interface
var k8sMetadataClient k8sMetadata.Interface
var informers *Informers
var isControllerLeader bool

var controllerServiceAccount string
var recorders map[string]record.EventRecorder

func loadClients() (*Clients, error) {

	//clusterConfig, err := k8sClientCmd.BuildConfigFromFlags("", "/root/.kube/config")

	clusterConfig, err := rest.InClusterConfig()

	if err != nil {
		return nil, err
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

	return clients, nil
}

func newInformers(clients *Clients) *Informers {

	k8sInformerFactory := k8sInformers.NewSharedInformerFactory(clients.K8sClient, 0)

	k8sInformerFactoryWithFilter := k8sInformers.NewSharedInformerFactoryWithOptions(clients.K8sClient, 0, k8sInformers.WithTweakListOptions(
		func(options *k8sMetaV1.ListOptions) {
			options.LabelSelector = fmt.Sprintf("%s,%s", TServerAppLabel, TServerNameLabel)
		}))

	crdInformerFactory := crdInformers.NewSharedInformerFactoryWithOptions(clients.CrdClient, 0)

	metadataInformerFactory := metadatainformer.NewSharedInformerFactory(clients.K8sMetadataClient, 0)

	nodeInformer := k8sInformerFactory.Core().V1().Nodes()

	serviceInformer := k8sInformerFactoryWithFilter.Core().V1().Services()
	podInformer := k8sInformerFactoryWithFilter.Core().V1().Pods()
	persistentVolumeClaimInformer := k8sInformerFactoryWithFilter.Core().V1().PersistentVolumeClaims()
	daemonSetInformer := k8sInformerFactoryWithFilter.Apps().V1().DaemonSets()
	statefulSetInformer := k8sInformerFactoryWithFilter.Apps().V1().StatefulSets()

	tserverInformer := crdInformerFactory.Crd().V1beta1().TServers()
	tendpointInformer := crdInformerFactory.Crd().V1beta1().TEndpoints()
	ttemplateInformer := crdInformerFactory.Crd().V1beta1().TTemplates()
	timageInformer := crdInformerFactory.Crd().V1beta1().TImages()
	ttreeInformer := crdInformerFactory.Crd().V1beta1().TTrees()
	texitedRecordInformer := crdInformerFactory.Crd().V1beta1().TExitedRecords()
	tdeployInformer := crdInformerFactory.Crd().V1beta1().TDeploys()
	taccountInformer := crdInformerFactory.Crd().V1beta1().TAccounts()

	tconfigInformer := metadataInformerFactory.ForResource(crdV1beta1.SchemeGroupVersion.WithResource("tconfigs"))

	informers = &Informers{
		k8sInformerFactory:           k8sInformerFactory,
		k8sInformerFactoryWithFilter: k8sInformerFactoryWithFilter,
		k8sMetadataInformerFactor:    metadataInformerFactory,
		crdInformerFactory:           crdInformerFactory,

		NodeInformer:                  nodeInformer,
		ServiceInformer:               serviceInformer,
		PodInformer:                   podInformer,
		PersistentVolumeClaimInformer: persistentVolumeClaimInformer,

		DaemonSetInformer:   daemonSetInformer,
		StatefulSetInformer: statefulSetInformer,

		TServerInformer:       tserverInformer,
		TEndpointInformer:     tendpointInformer,
		TTemplateInformer:     ttemplateInformer,
		TImageInformer:        timageInformer,
		TTreeInformer:         ttreeInformer,
		TExitedRecordInformer: texitedRecordInformer,
		TDeployInformer:       tdeployInformer,
		TAccountInformer:      taccountInformer,

		TConfigInformer: tconfigInformer,

		synced: false,
		synceds: []cache.InformerSynced{
			nodeInformer.Informer().HasSynced,
			serviceInformer.Informer().HasSynced,
			podInformer.Informer().HasSynced,
			persistentVolumeClaimInformer.Informer().HasSynced,

			statefulSetInformer.Informer().HasSynced,
			daemonSetInformer.Informer().HasSynced,

			tserverInformer.Informer().HasSynced,
			tendpointInformer.Informer().HasSynced,
			ttemplateInformer.Informer().HasSynced,
			timageInformer.Informer().HasSynced,
			ttreeInformer.Informer().HasSynced,
			texitedRecordInformer.Informer().HasSynced,
			tdeployInformer.Informer().HasSynced,
			taccountInformer.Informer().HasSynced,

			tconfigInformer.Informer().HasSynced,
		},
	}

	setEventHandler("node", informers.NodeInformer.Informer(), informers)
	setEventHandler("service", informers.ServiceInformer.Informer(), informers)
	setEventHandler("pod", informers.PodInformer.Informer(), informers)
	setEventHandler("persistentvolumeclaim", persistentVolumeClaimInformer.Informer(), informers)

	setEventHandler("statefulset", informers.StatefulSetInformer.Informer(), informers)
	setEventHandler("daemonset", informers.DaemonSetInformer.Informer(), informers)

	setEventHandler("tserver", informers.TServerInformer.Informer(), informers)
	setEventHandler("tendpoint", informers.TEndpointInformer.Informer(), informers)
	setEventHandler("ttemplate", informers.TTemplateInformer.Informer(), informers)
	setEventHandler("timage", informers.TImageInformer.Informer(), informers)
	setEventHandler("ttree", informers.TTreeInformer.Informer(), informers)
	setEventHandler("texitedrecord", informers.TExitedRecordInformer.Informer(), informers)
	setEventHandler("tdeploy", informers.TDeployInformer.Informer(), informers)
	setEventHandler("taccount", informers.TAccountInformer.Informer(), informers)

	setEventHandler("tconfig", informers.TConfigInformer.Informer(), informers)

	return informers
}

func GetNodeImage(namespace string) (image string, secret *string) {
	timage, err := informers.TImageInformer.Lister().TImages(namespace).Get("node")
	if err == nil && timage != nil {
		for _, release := range timage.Releases {
			if strings.HasPrefix(release.ID, "default") {
				return release.Image, release.Secret
			}
		}
		utilRuntime.HandleError(fmt.Errorf("read timage/node err: %s", "no defalut release set"))
	}
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf("get timage/node err: %s", err.Error()))
	}
	return ServiceImagePlaceholder, nil
}

func GetHelmFinalizerUsername(namespace string) string {
	return fmt.Sprintf("system:serviceaccount:%s:%s", namespace, HelmFinalizerAccount)
}

func GetControllerUsername() string {
	return controllerServiceAccount
}

func Event(namespace string, object runtime.Object, eventType, reason, message string) {
	recorder, ok := recorders[namespace]
	if !ok {
		eventBroadcaster := record.NewBroadcaster()
		eventBroadcaster.StartRecordingToSink(&k8sCoreTypedV1.EventSinkImpl{Interface: k8sClient.CoreV1().Events(namespace)})
		recorder = eventBroadcaster.NewRecorder(k8sSchema.Scheme, k8sCoreV1.EventSource{
			Component: "tarscontroller",
			Host:      "",
		})
		recorders[namespace] = recorder
	}
	if recorder != nil {
		recorder.Event(object, eventType, reason, message)
	}
}

func GetControllerContext() (*Clients, *Informers, error) {
	clients, err := loadClients()
	if err != nil {
		return nil, nil, err
	}

	informers = newInformers(clients)

	const namespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	if bs, err := ioutil.ReadFile(namespaceFile); err != nil {
		return nil, nil, fmt.Errorf("cannot read namespace value : %s \n", err.Error())
	} else {
		namespace := string(bs)
		controllerServiceAccount = fmt.Sprintf("system:serviceaccount:%s:%s", namespace, TControllerServiceAccount)
	}

	return clients, informers, nil
}

func GetEventRecorder(namespace string) record.EventRecorder {
	recorder, ok := recorders[namespace]
	if !ok {
		eventBroadcaster := record.NewBroadcaster()
		eventBroadcaster.StartRecordingToSink(&k8sCoreTypedV1.EventSinkImpl{Interface: k8sClient.CoreV1().Events(namespace)})
		recorder = eventBroadcaster.NewRecorder(k8sSchema.Scheme, k8sCoreV1.EventSource{
			Component: "tarscontroller",
			Host:      "",
		})
		if recorders == nil {
			recorders = make(map[string]record.EventRecorder, 0)
		}
		recorders[namespace] = recorder
		return recorder
	}
	return recorder
}

func IsControllerLeader() bool {
	return isControllerLeader
}

func LeaderElectAndRun(callbacks leaderelection.LeaderCallbacks) {
	isControllerLeader = false
	id, err := os.Hostname()
	if err != nil {
		fmt.Printf("GetHostName Error: %s\n", err.Error())
		return
	}
	id = id + "_" + string(uuid.NewUUID())

	rl, err := resourcelock.New("leases",
		"tars-system",
		"tars-tarscontroller",
		k8sClient.CoreV1(),
		k8sClient.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: GetEventRecorder("tars-system"),
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
				isControllerLeader = true
				if callbacks.OnStartedLeading != nil {
					callbacks.OnStartedLeading(ctx)
				}
			},
			OnStoppedLeading: func() {
				isControllerLeader = false
				if callbacks.OnStoppedLeading != nil {
					callbacks.OnStoppedLeading()
				}
			},
			OnNewLeader: callbacks.OnNewLeader,
		},
		Name: "tars-tarscontroller",
	})
}

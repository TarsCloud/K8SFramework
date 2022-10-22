package util

import (
	"context"
	"fmt"
	"io/ioutil"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	k8sSchema "k8s.io/client-go/kubernetes/scheme"
	k8sMetadata "k8s.io/client-go/metadata"
	k8sClientCmd "k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	crdVersioned "k8s.tars.io/client-go/clientset/versioned"
	crdScheme "k8s.tars.io/client-go/clientset/versioned/scheme"
	tarsCrdV1beta3 "k8s.tars.io/crd/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	"os"
	"time"
)

var k8sClient kubernetes.Interface
var crdClient crdVersioned.Interface
var k8sMetadataClient k8sMetadata.Interface
var factories *InformerFactories

var controllerServiceAccount string
var controllerNamespace string

const TControllerServiceAccount = "tars-controller"

func GetControllerUsername() string {
	return controllerServiceAccount
}

func CreateContext(masterUrl, kubeConfigPath string) (*Clients, *InformerFactories, error) {

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

	factories = newInformerFactories(clients)

	const namespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	bs, err := ioutil.ReadFile(namespaceFile)
	if err == nil {
		controllerNamespace = string(bs)
	} else {
		utilRuntime.HandleError(fmt.Errorf("cannot read namespace file : %s", err.Error()))
		controllerNamespace = tarsMeta.DefaultControllerNamespace
	}

	if masterUrl != "" || kubeConfigPath != "" {
		controllerServiceAccount = tarsMeta.DefaultUnlawfulAndOnlyForDebugUserName
	} else {
		controllerServiceAccount = fmt.Sprintf("system:serviceaccount:%s:%s", controllerNamespace, TControllerServiceAccount)
	}

	setupTFCWatch(factories)

	return clients, factories, nil
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
			EventRecorder: nil,
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

func GetDefaultNodeImage(namespace string) (image string, secret string) {
	var tfc *tarsCrdV1beta3.TFrameworkConfig
	if tfcInformerSynced() {
		if tfc = GetTFrameworkConfig(namespace); tfc != nil {
			return tfc.NodeImage.Image, tfc.NodeImage.Secret
		}

		utilRuntime.HandleError(fmt.Errorf("no default node image set"))
		return tarsMeta.ServiceImagePlaceholder, ""
	}

	tfc, _ = crdClient.CrdV1beta3().TFrameworkConfigs(namespace).Get(context.TODO(), tarsMeta.FixedTFrameworkConfigResourceName, k8sMetaV1.GetOptions{})
	if tfc != nil {
		return tfc.NodeImage.Image, tfc.NodeImage.Secret
	}

	utilRuntime.HandleError(fmt.Errorf("no default node image set"))
	return tarsMeta.ServiceImagePlaceholder, ""
}

package runtime

import (
	"context"
	"fmt"
	"io/ioutil"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	k8sSchema "k8s.io/client-go/kubernetes/scheme"
	k8sMetadata "k8s.io/client-go/metadata"
	k8sClientCmd "k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"
	crdVersioned "k8s.tars.io/client-go/clientset/versioned"
	crdScheme "k8s.tars.io/client-go/clientset/versioned/scheme"
	tarsTranslatorV1beta3 "k8s.tars.io/translator/tars/v1beta3"
	"os"
	"time"
)

var k8sClient kubernetes.Interface
var tarsClient crdVersioned.Interface
var k8sMetadataClient k8sMetadata.Interface

var Clients *Client
var Factories *InformerFactories
var TFCConfig *TFrameworkConfig
var TarsTranslator *tarsTranslatorV1beta3.Translator

var Username string
var Namespace string

func CreateContext(masterUrl, kubeConfigPath string, namespace bool) error {
	const namespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	bs, err := ioutil.ReadFile(namespaceFile)
	if err == nil {
		Namespace = string(bs)
	} else {
		utilRuntime.HandleError(fmt.Errorf("cannot read namespace file : %s", err.Error()))
		Namespace = "default"
	}

	clusterConfig, err := k8sClientCmd.BuildConfigFromFlags(masterUrl, kubeConfigPath)
	if err != nil {
		return err
	}

	Username = clusterConfig.Username

	k8sClient = kubernetes.NewForConfigOrDie(clusterConfig)

	tarsClient = crdVersioned.NewForConfigOrDie(clusterConfig)

	k8sMetadataClient = k8sMetadata.NewForConfigOrDie(clusterConfig)

	err = crdScheme.AddToScheme(k8sSchema.Scheme)
	if err != nil {
		return err
	}

	Clients = &Client{
		K8sClient:         k8sClient,
		CrdClient:         tarsClient,
		K8sMetadataClient: k8sMetadataClient,
	}

	Factories = newInformerFactories(Clients, namespace)

	TFCConfig = &TFrameworkConfig{}
	TFCConfig.setupTFCWatch(Factories, namespace)

	TarsTranslator = tarsTranslatorV1beta3.NewTranslator(TFCConfig)

	return nil
}

func LeaderElectAndRun(callbacks leaderelection.LeaderCallbacks, namespace, name string) {
	id, err := os.Hostname()
	if err != nil {
		klog.Errorf("GetHostName Error: %s\n", err.Error())
		return
	}
	id = id + "_" + string(uuid.NewUUID())

	rl, err := resourcelock.New(resourcelock.LeasesResourceLock,
		namespace,
		name,
		k8sClient.CoreV1(),
		k8sClient.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: nil,
		})

	if err != nil {
		klog.Errorf("Create ResourceLock Error: %s\n", err.Error())
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
		Name: name,
	})
}

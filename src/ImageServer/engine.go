package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	k8sCoreV1 "k8s.io/api/core/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sWatchV1 "k8s.io/apimachinery/pkg/watch"
	tarsV1beta3 "k8s.tars.io/apis/tars/v1beta3"
	tarsMeta "k8s.tars.io/meta"
	tarsRuntime "k8s.tars.io/runtime"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type TaskUserParams struct {
	Timage          string `json:"timage"`
	ServerApp       string `json:"serverApp" validate:"required,hostname_rfc1123"`
	ServerName      string `json:"serverName" validate:"required,hostname_rfc1123"`
	ServerType      string `json:"serverType" validate:"required,oneof=cpp java-jar java-war nodejs go php python"`
	ServerTag       string `json:"serverTag" validate:"hostname_rfc1123"`
	ServerFile      string `json:"serverFile"`
	Secret          string `json:"secret" validate:"required,hostname_rfc1123"`
	BaseImage       string `json:"baseImage" validate:"required,hostname_rfc1123"`
	BaseImageSecret string `json:"baseImageSecret" validate:"required,hostname_rfc1123"`
	CreatePerson    string `json:"createPerson"`
	Mark            string `json:"mark"`
}

type TaskPaths struct {
	buildDirInHost   string
	buildDirInPod    string
	buildDirInKaniko string

	uploadDirInPod string

	cacheDirInHost   string
	cacheDirInKaniko string

	dockerConfDirInHost   string
	dockerConfDirInKaniko string
	dockerConfDirInPod    string

	dockerConfFileInPod    string
	dockerConfFileInKaniko string

	dockerfileInPod    string
	dockerfileInHost   string
	dockerfileInKaniko string
}

type Task struct {
	id                    string
	createTime            k8sMetaV1.Time
	image                 string
	userParams            TaskUserParams
	paths                 TaskPaths
	taskBuildRunningState tarsV1beta3.TImageBuildState
	waitChan              chan error
	handler               string
	kanikoPodName         string
	timage                *tarsV1beta3.TImage
	repository            string
	registrySecret        string
	executorImage         string
	executorSecret        string
}

type Engine struct {
	buildChan chan *Task
}

func setup(task *Task) {
	task.paths.buildDirInPod = fmt.Sprintf("%s/%s.%s-%s", glPodBuildDir, task.userParams.ServerApp, task.userParams.ServerName, task.id)

	task.paths.buildDirInHost = fmt.Sprintf("%s/%s.%s-%s", glHostBuildDir, task.userParams.ServerApp, task.userParams.ServerName, task.id)
	task.paths.buildDirInKaniko = "/workspace"

	task.paths.dockerConfDirInPod = fmt.Sprintf("%s/docker", task.paths.buildDirInPod)

	task.paths.dockerConfDirInKaniko = fmt.Sprintf("%s/docker", task.paths.buildDirInKaniko)

	task.paths.dockerConfFileInPod = fmt.Sprintf("%s/config.json", task.paths.dockerConfDirInPod)

	task.paths.dockerfileInPod = fmt.Sprintf("%s/Dockerfile", task.paths.buildDirInPod)
	task.paths.dockerfileInKaniko = fmt.Sprintf("%s/Dockerfile", task.paths.buildDirInKaniko)

	task.paths.cacheDirInHost = glHostCacheDir
	task.paths.cacheDirInKaniko = "/cache"
}

func prepare(task *Task) error {
	log.Printf("task|%s: preparing...\n", task.id)

	serverBinDir := fmt.Sprintf("%s/%s", task.paths.buildDirInPod, "root/usr/local/server/bin")
	var err error
	if err = os.RemoveAll(task.paths.buildDirInPod); err != nil && !os.IsNotExist(err) {
		err = fmt.Errorf("remove dir(%s) error: %s", task.paths.buildDirInPod, err.Error())
		return err
	}

	if err = os.MkdirAll(serverBinDir, 0766); err != nil {
		err = fmt.Errorf("create dir(%s) error: %s", serverBinDir, err.Error())
		return err
	}

	go func() {
		time.Sleep(AutoDeleteServerBuildDirDuration)
		_ = os.RemoveAll(task.paths.buildDirInPod)
	}()

	ext := filepath.Ext(task.userParams.ServerFile)
	switch ext {
	case ".tgz":
		log.Printf("task|%s: decompressing...\n", task.id)
		if err = handleTarFile(task.userParams.ServerFile, serverBinDir); err != nil {
			err = fmt.Errorf("decompress file(%s) err: %s", task.userParams.ServerFile, err.Error())
			return err
		}
	case ".war":
		log.Printf("task|%s: decompressing...\n", task.id)
		if err = handleWarFile(task.userParams.ServerFile, serverBinDir); err != nil {
			err = fmt.Errorf("decompress file(%s) err: %s", task.userParams.ServerFile, err.Error())
			return err
		}
	case ".jar":
		log.Printf("task|%s: copying...\n", task.id)
		if err = handleJarFile(task.userParams.ServerFile, serverBinDir); err != nil {
			err = fmt.Errorf("copy file(%s) err: %s", task.userParams.ServerFile, err.Error())
			return err
		}
	default:
		err = fmt.Errorf("unknown file(%s) type", task.userParams.ServerFile)
		return err
	}

	dockerFileContent := fmt.Sprintf("FROM %s\nENV ServerType=%s\nCOPY root /\n", task.userParams.BaseImage, task.userParams.ServerType)
	if err = ioutil.WriteFile(task.paths.dockerfileInPod, []byte(dockerFileContent), 0666); err != nil {
		err = fmt.Errorf("create dockerfile(%s) error: %s", task.paths.dockerfileInPod, err.Error())
		return err
	}

	return nil
}

func secret(task *Task) error {
	log.Printf("task|%s: decrypting...\n", task.id)
	if err := os.Mkdir(task.paths.dockerConfDirInPod, 0777); err != nil {
		err = fmt.Errorf("create dir(%s) error: %s", task.paths.dockerConfDirInPod, err.Error())
		return err
	}

	var (
		secretResources     = map[string]interface{}{}
		dockerConfigContent []byte
	)

	if task.registrySecret != "" {
		secretResources[task.registrySecret] = nil
	}

	if task.userParams.BaseImageSecret != "" {
		secretResources[task.userParams.BaseImageSecret] = nil
	}

	if task.userParams.Secret != "" {
		secretResources[task.userParams.Secret] = nil
	}

	var secretContents [][]byte
	for resource := range secretResources {
		if resource == "" {
			continue
		}
		secretSnap, err := tarsRuntime.Clients.K8sClient.CoreV1().Secrets(tarsRuntime.Namespace).Get(context.TODO(), resource, k8sMetaV1.GetOptions{})
		if err != nil {
			err = fmt.Errorf("get resource %s %s/%s error: %s", "secrets", tarsRuntime.Namespace, resource, err.Error())
			return err
		}
		if v, ok := secretSnap.Data[".dockerconfigjson"]; ok {
			secretContents = append(secretContents, v)
		}
	}

	type Auth struct {
		Auth string `json:"auth,omitempty"`
	}

	type Auths struct {
		Auths map[string]Auth `json:"auths,omitempty"`
	}

	dockerConfigContent = func(contents [][]byte) []byte {
		base := Auths{
			Auths: map[string]Auth{},
		}
		for _, bs := range contents {
			var auths Auths
			err := json.Unmarshal(bs, &auths)
			if err != nil {
				continue
			}
			for k, v := range auths.Auths {
				base.Auths[k] = v
			}
		}

		if len(base.Auths) == 0 {
			return []byte{}
		}
		bs, _ := json.Marshal(base)
		return bs
	}(secretContents)

	if err := ioutil.WriteFile(task.paths.dockerConfFileInPod, dockerConfigContent, 0666); err != nil {
		err = fmt.Errorf("writeto file(%s) error: %s", task.paths.dockerConfFileInPod, err.Error())
		return err
	}

	return nil
}

func submit(task *Task) error {
	log.Printf("task|%s: submitting...\n", task.id)
	task.kanikoPodName = fmt.Sprintf("timage-builder-%s", task.id)

	hostPathDirectory := k8sCoreV1.HostPathDirectoryOrCreate
	podLayout := &k8sCoreV1.Pod{
		ObjectMeta: k8sMetaV1.ObjectMeta{
			Name:      task.kanikoPodName,
			Namespace: tarsRuntime.Namespace,
			Labels: map[string]string{
				tarsMeta.TServerAppLabel:  "tars",
				tarsMeta.TServerNameLabel: "tarskaniko",
				tarsMeta.TServerIdLabel:   task.id,
			},
		},
		Spec: k8sCoreV1.PodSpec{
			Containers: []k8sCoreV1.Container{
				{
					Name:            "kaniko",
					Image:           task.executorImage,
					ImagePullPolicy: k8sCoreV1.PullAlways,
					Args: []string{
						fmt.Sprintf("--timage=%s", task.userParams.Timage),
						fmt.Sprintf("--id=%s", task.id),
						fmt.Sprintf("--dockerfile=%s", task.paths.dockerfileInKaniko),
						fmt.Sprintf("--context=dir:/%s", task.paths.buildDirInKaniko),
						fmt.Sprintf("--destination=%s", task.image),
					},
					VolumeMounts: []k8sCoreV1.VolumeMount{
						{
							Name:      "kaniko-workdir",
							MountPath: task.paths.buildDirInKaniko,
						},
						{
							Name:      "kaniko-cache",
							MountPath: task.paths.cacheDirInKaniko,
						},
					},
					Env: []k8sCoreV1.EnvVar{
						{
							Name:  "DOCKER_CONFIG",
							Value: task.paths.dockerConfDirInKaniko,
						},
					},
					TerminationMessagePath: "/kaniko/status",
				},
			},
			RestartPolicy: k8sCoreV1.RestartPolicyNever,

			Volumes: []k8sCoreV1.Volume{
				{
					Name: "kaniko-workdir",
					VolumeSource: k8sCoreV1.VolumeSource{
						HostPath: &k8sCoreV1.HostPathVolumeSource{
							Path: task.paths.buildDirInHost,
							Type: &hostPathDirectory,
						},
					},
				},
				{
					Name: "kaniko-cache",
					VolumeSource: k8sCoreV1.VolumeSource{
						HostPath: &k8sCoreV1.HostPathVolumeSource{
							Path: task.paths.cacheDirInHost,
							Type: &hostPathDirectory,
						},
					},
				},
			},
			Affinity: &k8sCoreV1.Affinity{PodAffinity: &k8sCoreV1.PodAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []k8sCoreV1.PodAffinityTerm{
					{
						LabelSelector: &k8sMetaV1.LabelSelector{
							MatchLabels: map[string]string{
								"statefulset.kubernetes.io/pod-name": glPodName,
							},
						},
						Namespaces:  []string{tarsRuntime.Namespace},
						TopologyKey: tarsMeta.K8SHostNameLabel,
					},
				},
			}},
			ServiceAccountName: "tars-tarsimage",
		},
	}

	if task.executorSecret != "" {
		podLayout.Spec.ImagePullSecrets = []k8sCoreV1.LocalObjectReference{
			{
				task.executorSecret,
			},
		}
	}

	_, err := tarsRuntime.Clients.K8sClient.CoreV1().Pods(tarsRuntime.Namespace).Create(context.TODO(), podLayout, k8sMetaV1.CreateOptions{})
	if err != nil {
		err = fmt.Errorf("create pod failed: %s", err.Error())
		return err
	}

	go func() {
		time.Sleep(time.Minute * 15)
		_ = tarsRuntime.Clients.K8sClient.CoreV1().Pods(tarsRuntime.Namespace).Delete(context.TODO(), task.kanikoPodName, k8sMetaV1.DeleteOptions{})
	}()

	return nil
}

func watch(task *Task) error {
	log.Printf("task|%s: watching...\n", task.id)
	watchInterface, err := tarsRuntime.Clients.K8sClient.CoreV1().Pods(tarsRuntime.Namespace).Watch(context.TODO(), k8sMetaV1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", task.kanikoPodName),
		Watch:         true,
	})

	if err != nil {
		return err
	}

	for {
		event, ok := <-watchInterface.ResultChan()
		if !ok {
			return nil
		}
		switch event.Type {
		case k8sWatchV1.Added, k8sWatchV1.Bookmark:
			continue
		case k8sWatchV1.Error:
			errSnap := event.Object.(*k8sMetaV1.Status)
			err = fmt.Errorf("pod|%s failed: %s\n", task.kanikoPodName, errSnap.Message)
			return err
		case k8sWatchV1.Deleted:
			err = fmt.Errorf("pod|%s deleted: %s\n", task.kanikoPodName, "unknown reason")
			return err
		case k8sWatchV1.Modified:
			podSnap := event.Object.(*k8sCoreV1.Pod)
			switch podSnap.Status.Phase {
			case k8sCoreV1.PodPending:
				log.Printf("task|%s: pod|%s pending\n", task.id, task.kanikoPodName)
				continue
			case k8sCoreV1.PodRunning:
				log.Printf("task|%s: pod|%s running\n", task.id, task.kanikoPodName)
				continue
			case k8sCoreV1.PodFailed:
				var exitMessage = "failed but unknown state"
				if podSnap.Status.ContainerStatuses[0].State.Terminated != nil {
					exitMessage = podSnap.Status.ContainerStatuses[0].State.Terminated.Message
				}
				message := fmt.Sprintf("pod|%s %s", task.kanikoPodName, exitMessage)
				log.Printf("task|%s: %s\n", task.id, message)
				err = fmt.Errorf(exitMessage)
				return err
			case k8sCoreV1.PodSucceeded:
				log.Printf("task|%s: pod|%s success\n", task.id, task.kanikoPodName)
				return nil
			}
		}
	}
}

func handleTarFile(sourceFile string, dstDir string) error {
	cmd := exec.Command("tar", "zxvf", sourceFile, "--strip-components=1", "-C", dstDir)
	err := cmd.Run()
	if err != nil {
		fmt.Print(err.Error())
	}
	return nil
}

func handleWarFile(sourceFile string, dstDir string) error {
	cmd := exec.Command("unzip", sourceFile, "-d", dstDir)
	err := cmd.Run()
	if err != nil {
		fmt.Print(err.Error())
	}
	return nil
}

func handleJarFile(sourceFile string, dstDir string) error {
	cmd := exec.Command("cp", sourceFile, dstDir)
	err := cmd.Run()
	if err != nil {
		fmt.Print(err.Error())
	}
	return nil
}

func NewEngine() *Engine {
	worker := &Engine{
		buildChan: make(chan *Task),
	}
	return worker
}

func pushBuildRunningState(task *Task) error {
	task.timage.Build.Running = &task.taskBuildRunningState
	var err error
	task.timage, err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TImages(tarsRuntime.Namespace).Update(context.TODO(), task.timage, k8sMetaV1.UpdateOptions{})
	if err != nil {
		var message = fmt.Sprintf("update running state failed: %s", err.Error())
		log.Printf("task|%s: %s\n", task.id, message)
		return fmt.Errorf(message)
	}
	return err
}

func (e *Engine) onBuildFailed(task *Task, err error) {

	if task.waitChan != nil {
		task.waitChan <- err
	}

	task.taskBuildRunningState.Phase = BuildPhaseFailed
	task.taskBuildRunningState.Message = err.Error()

	buildState := tarsV1beta3.TImageBuild{
		Running: nil,
		Last:    &task.taskBuildRunningState,
	}

	jsonPatch := tarsMeta.JsonPatch{
		{
			OP:    tarsMeta.JsonPatchAdd,
			Path:  "/build",
			Value: buildState,
		},
	}

	bs, _ := json.Marshal(jsonPatch)

	for i := 1; i < 3; i++ {
		_, err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TImages(tarsRuntime.Namespace).Patch(context.TODO(), task.timage.Name, types.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		if err != nil {
			time.Sleep(time.Second * 1)
			continue
		}
		break
	}

	if err != nil {
		log.Printf("task|%s: update running state failed: %s\n", task.id, err.Error())
	}

	return
}

func (e *Engine) onBuildSuccess(task *Task) {

	task.taskBuildRunningState.Phase = BuildPhaseDone
	task.taskBuildRunningState.Message = "Success"

	buildState := tarsV1beta3.TImageBuild{
		Running: nil,
		Last:    &task.taskBuildRunningState,
	}

	releases := []*tarsV1beta3.TImageRelease{
		{
			ID:           task.taskBuildRunningState.ID,
			Image:        task.taskBuildRunningState.Image,
			Secret:       task.taskBuildRunningState.Secret,
			CreatePerson: &task.taskBuildRunningState.CreatePerson,
			CreateTime:   task.taskBuildRunningState.CreateTime,
			Mark:         &task.taskBuildRunningState.Mark,
		},
	}

	if task.timage.Releases != nil {
		releases = append(releases, task.timage.Releases...)
		max := getMaxReleases()
		if len(releases) <= max {
			task.timage.Releases = releases
		} else {
			task.timage.Releases = releases[0:max]
		}
	}

	jsonPatch := tarsMeta.JsonPatch{
		{
			OP:    tarsMeta.JsonPatchAdd,
			Path:  "/build",
			Value: buildState,
		},
		{
			OP:    tarsMeta.JsonPatchAdd,
			Path:  "/releases",
			Value: releases,
		},
	}

	var err error
	bs, _ := json.Marshal(jsonPatch)

	for i := 1; i < 3; i++ {
		_, err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TImages(tarsRuntime.Namespace).Patch(context.TODO(), task.timage.Name, types.JSONPatchType, bs, k8sMetaV1.PatchOptions{})
		if err != nil {
			time.Sleep(time.Second * 1)
			continue
		}
		break
	}

	if err != nil {
		message := fmt.Sprintf("update running state failed: %s", err.Error())
		log.Printf("task|%s: %s\n", task.id, message)
		err = fmt.Errorf(message)
	}

	if task.waitChan != nil {
		task.waitChan <- err
	}

	return
}

func (e *Engine) PostTask(task *Task) (string, error) {

	if task.userParams.ServerTag == "" {
		task.userParams.ServerTag = task.id
	}

	task.repository, task.registrySecret = getRepository()
	if task.repository == "" {
		return "", fmt.Errorf("no default repository value set")
	}
	task.image = fmt.Sprintf("%s/%s.%s:%s", task.repository, strings.ToLower(task.userParams.ServerApp), strings.ToLower(task.userParams.ServerName), task.userParams.ServerTag)

	tfc := tarsRuntime.TFCConfig.GetTFrameworkConfig(tarsRuntime.Namespace)
	if tfc == nil {
		return "", fmt.Errorf("no execute image value set")
	}

	task.executorImage, task.executorSecret = tfc.ImageBuild.Executor.Image, tfc.ImageBuild.Executor.Secret
	if task.executorImage == "" {
		return "", fmt.Errorf("no execute image value set")
	}

	task.taskBuildRunningState = tarsV1beta3.TImageBuildState{
		ID:              task.id,
		BaseImage:       task.userParams.BaseImage,
		BaseImageSecret: task.userParams.BaseImageSecret,
		Image:           task.image,
		Secret:          task.userParams.Secret,
		ServerType:      task.userParams.ServerType,
		CreatePerson:    task.userParams.CreatePerson,
		CreateTime:      task.createTime,
		Mark:            task.userParams.Mark,
		Phase:           BuildPhasePending,
		Message:         "pending",
		Handler:         glPodName,
	}

	var err error
	task.timage, err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TImages(tarsRuntime.Namespace).Get(context.TODO(), task.userParams.Timage, k8sMetaV1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get resource %s %s/%s failed: %s", "timage", tarsRuntime.Namespace, task.userParams.Timage, err.Error())
	}

	if task.timage.ImageType != TImageTypeServer {
		return "", fmt.Errorf("unexcepted imageType value: %s", task.timage.ImageType)
	}

	if task.timage.Build != nil && task.timage.Build.Running != nil {
		return "", fmt.Errorf("another task is running")
	}

	if task.timage.Build == nil {
		task.timage.Build = &tarsV1beta3.TImageBuild{
			Running: &task.taskBuildRunningState,
		}
	} else {
		task.timage.Build.Running = &task.taskBuildRunningState
	}

	task.timage, err = tarsRuntime.Clients.CrdClient.TarsV1beta3().TImages(tarsRuntime.Namespace).Update(context.TODO(), task.timage, k8sMetaV1.UpdateOptions{})
	if err != nil {
		return "", fmt.Errorf("update running state failed: %s\n", err.Error())
	}

	e.buildChan <- task
	return task.image, nil
}

func (e *Engine) Start(stopChan chan struct{}, threads int) {
	for i := 0; i < threads; i++ {
		go func() {
			for true {
				select {
				case task := <-e.buildChan:
					e.build(task)
				case <-stopChan:
					return
				}
			}
		}()
	}
}

func (e *Engine) build(task *Task) {

	defer log.Printf("task|%s: stopped\n", task.id)

	var err error
	for true {

		setup(task)

		task.taskBuildRunningState.Phase = BuildPhasePreparing
		task.taskBuildRunningState.Message = "preparing context"
		_ = pushBuildRunningState(task)

		if err = prepare(task); err != nil {
			message := fmt.Sprintf("prepare context failed, %s", err.Error())
			log.Printf("task|%s: %s\n", task.id, message)
			err = fmt.Errorf(message)
			break
		}

		task.taskBuildRunningState.Phase = BuildPhasePreparing
		task.taskBuildRunningState.Message = "preparing secret"
		_ = pushBuildRunningState(task)

		if err = secret(task); err != nil {
			message := fmt.Sprintf("prepare secret failed, %s", err.Error())
			log.Printf("task|%s: %s\n", task.id, message)
			err = fmt.Errorf(message)
			break
		}

		task.taskBuildRunningState.Phase = BuildPhaseSubmitting
		task.taskBuildRunningState.Message = "submitting task"
		_ = pushBuildRunningState(task)

		if err = submit(task); err != nil {
			message := fmt.Sprintf("submit task failed, %s", err.Error())
			log.Printf("task|%s: %s\n", task.id, message)
			err = fmt.Errorf(message)
			break
		}

		if err = watch(task); err != nil {
			break
		}

		log.Printf("task|%s: %s\n", task.id, "Success")
		break
	}

	if err != nil {
		e.onBuildFailed(task, err)
		return
	}

	e.onBuildSuccess(task)
}

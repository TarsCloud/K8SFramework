package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	k8sCoreV1 "k8s.io/api/core/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	crdV1beta1 "k8s.tars.io/api/crd/v1beta1"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type TaskUserParams struct {
	Timage          string `json:"timage"`
	ServerApp       string `json:"serverApp"`
	ServerName      string `json:"serverName"`
	ServerType      string `json:"serverType"`
	ServerFile      string `json:"serverFile"`
	Secret          string `json:"secret"`
	BaseImage       string `json:"baseImage"`
	BaseImageSecret string `json:"baseImageSecret"`
	CreatePerson    string `json:"createPerson"`
	Mark            string `json:"mark"`
}

type Task struct {
	id                    string
	image                 string
	userParams            *TaskUserParams
	taskBuildRunningState *crdV1beta1.TImageBuildState
	taskBuildLastState    *crdV1beta1.TImageBuildState
	createTime            k8sMetaV1.Time
	timage                *crdV1beta1.TImage
	dockerInterface       *DockerClient
	waitChan              chan error
}

type TaskOption func(task *Task)

func withWaitChan(waitChan chan error) TaskOption {
	return func(task *Task) {
		task.waitChan = waitChan
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

type Builder struct {
	k8sOption *K8SOption
	tasksChan chan *Task
}

func (b *Builder) pushBuildRunningState(task *Task) error {
	buildState := crdV1beta1.TImageBuild{
		Running: task.taskBuildRunningState,
	}
	if task.timage.Build != nil && task.timage.Build.Last != nil {
		buildState.Last = task.timage.Build.Last
	}

	bs, _ := json.Marshal(buildState)

	patchContent := fmt.Sprintf("[{\"op\":\"add\",\"path\":\"/build\",\"value\":%s}]", bs)
	var err error
	task.timage, err = b.k8sOption.crdClientInterface.CrdV1beta1().TImages(b.k8sOption.namespace).Patch(context.TODO(), task.timage.Name, types.JSONPatchType, []byte(patchContent), k8sMetaV1.PatchOptions{})
	if err != nil {
		utilRuntime.HandleError(err)
	}
	return err
}

func (b *Builder) onBuildFailed(task *Task, err error) {

	if task.waitChan != nil {
		task.waitChan <- err
	}

	task.taskBuildRunningState.Phase = BuildPhaseFailed
	task.taskBuildRunningState.Message = err.Error()

	buildState := crdV1beta1.TImageBuild{
		Last: task.taskBuildRunningState,
	}

	bs, _ := json.Marshal(buildState)

	patchContent := fmt.Sprintf("[{\"op\":\"add\",\"path\":\"/build\",\"value\":%s}]", bs)
	task.timage, err = b.k8sOption.crdClientInterface.CrdV1beta1().TImages(b.k8sOption.namespace).Patch(context.TODO(), task.timage.Name, types.JSONPatchType, []byte(patchContent), k8sMetaV1.PatchOptions{})
	if err != nil {
		utilRuntime.HandleError(err)
	}
}

func (b *Builder) onBuildSuccess(task *Task) {
	newRelease := &crdV1beta1.TImageRelease{
		ID:           task.taskBuildRunningState.ID,
		Image:        task.taskBuildRunningState.Image,
		Secret:       &task.taskBuildRunningState.Secret,
		CreatePerson: &task.taskBuildRunningState.CreatePerson,
		CreateTime:   task.taskBuildRunningState.CreateTime,
		Mark:         &task.taskBuildRunningState.Mark,
	}

	task.taskBuildRunningState.Phase = BuildPhaseDone
	task.taskBuildRunningState.Message = "Success"

	releaseBS, _ := json.Marshal(newRelease)

	buildState := crdV1beta1.TImageBuild{
		Last: task.taskBuildRunningState,
	}
	stateBS, _ := json.Marshal(buildState)

	const MaxRecordLen = 120
	var patchContent string

	if len(task.timage.Releases) < MaxRecordLen {
		patchContent = fmt.Sprintf("[{\"op\":\"add\",\"path\":\"/releases/0\",\"value\":%s},{\"op\":\"add\",\"path\":\"/build\",\"value\":%s}]", releaseBS, stateBS)
	} else {
		patchContent = fmt.Sprintf("[{\"op\":\"add\",\"path\":\"/releases/0\",\"value\":%s},{\"op\":\"remove\",\"path\":\"/releases/-\"}},{\"op\":\"add\",\"path\":\"/build\",\"value\":%s}]", releaseBS, stateBS)
	}

	var err error
	task.timage, err = b.k8sOption.crdClientInterface.CrdV1beta1().TImages(b.k8sOption.namespace).Patch(context.TODO(), task.timage.Name, types.JSONPatchType, []byte(patchContent), k8sMetaV1.PatchOptions{})
	if err != nil {
		utilRuntime.HandleError(err)
	}

	if task.waitChan != nil {
		task.waitChan <- err
	}

	return
}

func (b *Builder) build(task *Task) {
	buildDir := fmt.Sprintf("%s/%s.%s-%s", AbsoluteBuildWorkPath, task.userParams.ServerApp, task.userParams.ServerName, task.id)
	serverBinDir := fmt.Sprintf("%s/%s", buildDir, "root/usr/local/server/bin")

	go func() {
		time.Sleep(AutoDeleteServerFileDuration)
		_ = os.Remove(buildDir)
	}()

	var err error
	for true {
		if err = os.RemoveAll(buildDir); err != nil && !os.IsNotExist(err) {
			err = fmt.Errorf("remove dir(%s) error : %s", buildDir, err.Error())
			break
		}

		go func() {
			time.Sleep(AutoDeleteServerBuildDirDuration)
			_ = os.RemoveAll(buildDir)
		}()

		if err = os.MkdirAll(serverBinDir, 0766); err != nil {
			err = fmt.Errorf("mkdir(%s) error : %s", serverBinDir, err.Error())
			utilRuntime.HandleError(err)
			break
		}

		task.taskBuildRunningState.Phase = BuildPhasePrepareFile
		task.taskBuildRunningState.Message = "BuildPhasePrepareFile"
		_ = b.pushBuildRunningState(task)

		ext := filepath.Ext(task.userParams.ServerFile)
		switch ext {
		case ".tgz":
			if err = handleTarFile(task.userParams.ServerFile, serverBinDir); err != nil {
				err = fmt.Errorf("decompressing file(%s) err : %s", task.userParams.ServerFile, err.Error())
				utilRuntime.HandleError(err)
				break
			}
		case ".war":
			if err = handleWarFile(task.userParams.ServerFile, serverBinDir); err != nil {
				err = fmt.Errorf("decompressing file(%s) err : %s", task.userParams.ServerFile, err.Error())
				utilRuntime.HandleError(err)
				break
			}
		case ".jar":
			if err = handleJarFile(task.userParams.ServerFile, serverBinDir); err != nil {
				err = fmt.Errorf("cp file(%s) err : %s", task.userParams.ServerFile, err.Error())
				utilRuntime.HandleError(err)
				break
			}
		default:
			err = fmt.Errorf("unknown file(%s) type", task.userParams.ServerFile)
			utilRuntime.HandleError(err)
			break
		}

		if task.dockerInterface, err = NewDockerClient(); err != nil {
			err = fmt.Errorf("create docker interface error: %s", err.Error())
			utilRuntime.HandleError(err)
			break
		}

		task.taskBuildRunningState.Phase = BuildPhaseReadingSecret
		task.taskBuildRunningState.Message = "BuildPhaseReadingSecret"
		_ = b.pushBuildRunningState(task)

		if task.taskBuildRunningState.Secret == "" {
			var defaultSecretBytes []byte
			if defaultSecretBytes, err = ioutil.ReadFile(RegistrySecretFile); err != nil {
				err = fmt.Errorf("get default secret value error: %s", err.Error())
				utilRuntime.HandleError(err)
				break
			}
			if defaultSecretBytes != nil && len(defaultSecretBytes) > 0 {
				task.taskBuildRunningState.Secret = string(defaultSecretBytes)
			}
		}

		var secrets []*k8sCoreV1.Secret

		if task.taskBuildRunningState.Secret != "" {
			secret, err := b.k8sOption.k8sClientInterface.CoreV1().Secrets(b.k8sOption.namespace).Get(context.TODO(), task.taskBuildRunningState.Secret, k8sMetaV1.GetOptions{})
			if err != nil {
				err = fmt.Errorf("get resource %s %s/%s error :%s", "secret", b.k8sOption.namespace, task.taskBuildRunningState.Secret, err.Error())
				utilRuntime.HandleError(err)
				break
			}
			if secret != nil {
				secrets = append(secrets, secret)
			}
		}

		if task.taskBuildRunningState.BaseImageSecret != "" {
			secret, err := b.k8sOption.k8sClientInterface.CoreV1().Secrets(b.k8sOption.namespace).Get(context.TODO(), task.taskBuildRunningState.BaseImageSecret, k8sMetaV1.GetOptions{})
			if err != nil {
				err = fmt.Errorf("get resource %s %s/%s error :%s", "secret", b.k8sOption.namespace, task.taskBuildRunningState.BaseImageSecret, err.Error())
				utilRuntime.HandleError(err)
				break
			}
			if secret != nil {
				secrets = append(secrets, secret)
			}
		}

		var dockerFile = fmt.Sprintf("%s/%s", buildDir, "Dockerfile")
		if err = task.dockerInterface.createDockerFile(task.userParams.BaseImage, task.userParams.ServerType, dockerFile); err != nil {
			err = fmt.Errorf("create dockerfile(%s) error : %s", dockerFile, err.Error())
			utilRuntime.HandleError(err)
			break
		}

		task.taskBuildRunningState.Phase = BuildPhasePullingBaseImage
		task.taskBuildRunningState.Message = "BuildPhasePullingBaseImage"
		_ = b.pushBuildRunningState(task)
		if err = task.dockerInterface.pullImage(task.userParams.BaseImage, time.Minute*5, secrets); err != nil {
			err = fmt.Errorf("pull image %s error :%s", task.userParams.BaseImage, err.Error())
			utilRuntime.HandleError(err)
			break
		}

		task.taskBuildRunningState.Phase = BuildPhaseBuilding
		task.taskBuildRunningState.Message = "BuildPhaseBuilding"
		_ = b.pushBuildRunningState(task)
		if err = task.dockerInterface.buildImage(task.image, buildDir); err != nil {
			err = fmt.Errorf("build image %s error :%s", task.image, err.Error())
			utilRuntime.HandleError(err)
			break
		}

		task.taskBuildRunningState.Phase = BuildPhasePushing
		task.taskBuildRunningState.Message = "BuildPhasePushing"
		_ = b.pushBuildRunningState(task)

		if err = task.dockerInterface.pushImage(task.image, time.Second*300, secrets); err != nil {
			err = fmt.Errorf("push image %s error :%s", task.image, err.Error())
			utilRuntime.HandleError(err)
			break
		}

		break
	}

	if err == nil {
		b.onBuildSuccess(task)
	} else {
		b.onBuildFailed(task, err)
	}
}

func (b *Builder) PostTask(id, timageName, serverApp, serverName, serverType, serverFile, baseImage, baseImageSecretName, secretName, createPerson, mark string, options ...TaskOption) (string, error) {
	timage, err := b.k8sOption.crdClientInterface.CrdV1beta1().TImages(b.k8sOption.namespace).Get(context.TODO(), timageName, k8sMetaV1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get resource %s %s/%s err : %s", "timage", b.k8sOption.namespace, timageName, err.Error())
	}

	if timage.ImageType != TImageTypeServer {
		return "", fmt.Errorf("unexcepted imageType value :%s", timage.ImageType)
	}

	if timage.Build != nil && timage.Build.Running != nil {
		return "", fmt.Errorf("another task is running")
	}

	defaultRegistry, err := ioutil.ReadFile(RegistryConfigFile)
	if err != nil {
		return "", fmt.Errorf("get default registry value error: %s", err.Error())
	}

	targetImage := fmt.Sprintf("%s/%s.%s:%s", defaultRegistry, strings.ToLower(serverApp), strings.ToLower(serverName), id)

	task := &Task{
		userParams: &TaskUserParams{
			Timage:          timageName,
			ServerApp:       serverApp,
			ServerName:      serverName,
			ServerType:      serverType,
			ServerFile:      serverFile,
			Secret:          secretName,
			BaseImage:       baseImage,
			BaseImageSecret: baseImageSecretName,
			CreatePerson:    createPerson,
			Mark:            mark,
		},
		id:    id,
		image: targetImage,
		taskBuildRunningState: &crdV1beta1.TImageBuildState{
			ID:              id,
			BaseImage:       baseImage,
			BaseImageSecret: baseImageSecretName,
			Image:           targetImage,
			Secret:          secretName,
			ServerType:      serverType,
			CreatePerson:    createPerson,
			CreateTime:      k8sMetaV1.Now(),
			Mark:            mark,
			Phase:           BuildPhasePending,
			Message:         "Pending",
		},
		createTime:      k8sMetaV1.Now(),
		timage:          timage,
		dockerInterface: nil,
	}
	if err = b.pushBuildRunningState(task); err != nil {
		return "", err
	}

	for _, option := range options {
		if option != nil {
			option(task)
		}
	}

	b.tasksChan <- task
	return task.image, nil
}

func (b *Builder) Start(stopChan chan struct{}, threads int) {
	for i := 0; i < threads; i++ {
		go func() {
			for true {
				select {
				case task := <-b.tasksChan:
					b.build(task)
				case <-stopChan:
					return
				}
			}
		}()
	}
}

var builder *Builder

func NewBuildWorker() *Builder {
	worker := &Builder{
		k8sOption: LoadK8SOption(),
		tasksChan: make(chan *Task),
	}
	return worker
}

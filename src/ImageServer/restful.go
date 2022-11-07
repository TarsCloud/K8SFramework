package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"hash/crc32"
	"io"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type BuildResult struct {
	ID     string `json:"id"`
	Image  string `json:"image"`
	Secret string `json:"secret"`
	Source string `json:"source"`
}

type RestfulResponse struct {
	Status  int          `json:"status"`
	Message string       `json:"message"`
	Result  *BuildResult `json:"result,omitempty"`
	Handler string       `json:"handler,omitempty"`
}

func Handler(engine *Engine, writer http.ResponseWriter, r *http.Request) {

	writer.Header().Add("Content-Type", "application/json")

	wait := r.URL.Query().Get("wait")

	response := &RestfulResponse{
		Handler: glPodName,
		Status:  http.StatusInternalServerError,
		Message: http.StatusText(http.StatusInternalServerError),
		Result:  nil,
	}

	var err error
	for true {
		vars := mux.Vars(r)

		if len(vars) == 0 {
			err = fmt.Errorf("unexcept request path")
			response.Status = http.StatusBadRequest
			response.Message = err.Error()
			utilRuntime.HandleError(err)
			break
		}

		timageName, ok := vars["timage"]
		if !ok {
			err = fmt.Errorf("unexcept request path")
			response.Status = http.StatusBadRequest
			response.Message = err.Error()
			utilRuntime.HandleError(err)
			break
		}

		if err = r.ParseMultipartForm(1 * 1024 * 1024 * 150); err != nil {
			err = fmt.Errorf("parse form error: %s", err.Error())
			response.Status = http.StatusBadRequest
			response.Message = err.Error()
			break
		}

		var multipartServerFile multipart.File
		var multipartFileHandler *multipart.FileHeader
		if multipartServerFile, multipartFileHandler, err = r.FormFile(ServerFileFormKey); err != nil {
			err = fmt.Errorf("parse form error: %s", err.Error())
			response.Status = http.StatusBadRequest
			response.Message = err.Error()
			utilRuntime.HandleError(err)
			break
		}

		task := &Task{
			id:         fmt.Sprintf("v%s-%x-%x", time.Now().Format("20060102030405"), crc32.ChecksumIEEE([]byte(glPodName)), rand.Intn(0xefff)+0x1000),
			createTime: k8sMetaV1.Now(),
			handler:    glPodName,
			userParams: TaskUserParams{
				Timage:          timageName,
				ServerApp:       r.FormValue(ServerAppFormKey),
				ServerName:      r.FormValue(ServerNameFormKey),
				ServerType:      r.FormValue(ServerTypeFormKey),
				ServerTag:       r.FormValue(ServerTagFormKey),
				ServerFile:      "",
				Secret:          r.FormValue(ServerSecretFormKey),
				BaseImage:       r.FormValue(BaseImageFormKey),
				BaseImageSecret: r.FormValue(BaseImageSecretFormKey),
				CreatePerson:    r.FormValue(CreatePersonFormKey),
				Mark:            r.FormValue(MarkFormKey),
			},
		}

		//if ok, err = validateUserParams(task.userParams); !ok {
		//	err = fmt.Errorf("invalid fields format %s", err.Error())
		//	response.Status = http.StatusBadRequest
		//	response.Message = err.Error()
		//	utilRuntime.HandleError(err)
		//	break
		//}

		var serverFile string
		multipartFileName := multipartFileHandler.Filename
		if strings.HasSuffix(multipartFileName, ".tar.gz") || strings.HasSuffix(multipartFileName, ".tgz") {
			serverFile = fmt.Sprintf("%s/%s.%s-%s%s", glPodUploadDir, task.userParams.ServerApp, task.userParams.ServerName, task.id, ".tgz")
		} else if strings.HasSuffix(multipartFileName, ".war") || strings.HasSuffix(multipartFileName, ".jar") {
			serverFile = fmt.Sprintf("%s/%s.%s-%s%s", glPodUploadDir, task.userParams.ServerApp, task.userParams.ServerName, task.id, filepath.Ext(multipartFileName))
		} else {
			err = fmt.Errorf("unsupported server file type %s", serverFile)
			response.Status = http.StatusBadRequest
			response.Message = err.Error()
			utilRuntime.HandleError(err)
			break
		}
		task.userParams.ServerFile = serverFile

		f, _ := os.OpenFile(serverFile, os.O_CREATE|os.O_WRONLY, 0666)
		if _, err = io.Copy(f, multipartServerFile); err != nil {
			_ = os.Remove(serverFile)
			err = fmt.Errorf("write file(%s) error: %s", serverFile, err.Error())
			response.Status = http.StatusInternalServerError
			response.Message = err.Error()
			utilRuntime.HandleError(err)
			break
		}

		go func() {
			time.Sleep(AutoDeleteServerFileDuration)
			_ = os.Remove(serverFile)
		}()

		var image string

		if wait != "1" && wait != "true" {
			if image, err = engine.PostTask(task); err != nil {
				response.Status = http.StatusInternalServerError
				response.Message = err.Error()
				break
			}
			response.Status = http.StatusCreated
			response.Message = http.StatusText(http.StatusCreated)
			response.Result = &BuildResult{
				ID:     task.id,
				Image:  image,
				Secret: task.userParams.Secret,
				Source: timageName,
			}
			break
		}

		task.waitChan = make(chan error, 1)
		if image, err = engine.PostTask(task); err != nil {
			response.Status = http.StatusInternalServerError
			response.Message = err.Error()
			break
		}

		select {
		case err = <-task.waitChan:
			if err != nil {
				response.Status = http.StatusInternalServerError
				response.Message = err.Error()
				break
			}
			response.Status = http.StatusOK
			response.Message = http.StatusText(http.StatusOK)
			response.Result = &BuildResult{
				ID:     task.id,
				Image:  image,
				Secret: task.userParams.Secret,
				Source: timageName,
			}
			break
		}

		break
	}
	bs, _ := json.Marshal(response)
	writer.WriteHeader(response.Status)
	_, _ = writer.Write(bs)
}

type RestfulServer struct {
}

func NewRestful() *RestfulServer {
	return &RestfulServer{}
}

func (s *RestfulServer) Start(stopCh chan struct{}) {
	router := mux.NewRouter()
	router.HandleFunc("/api/{version}/timage/{timage}/building", func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodPost:
			Handler(glEngine, writer, request)
		default:
			writer.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	srv := &http.Server{
		Addr:              ":80",
		Handler:           router,
		ReadTimeout:       300 * time.Second,
		ReadHeaderTimeout: 60 * time.Second,
		WriteTimeout:      600 * time.Second,
		IdleTimeout:       900 * time.Second,
	}
	go func() {
		err := srv.ListenAndServe()
		// ListenAndServe always returns a non-nil error. After Shutdown or Close,
		// the returned error is ErrServerClosed.
		if err != nil {
			utilRuntime.HandleError(fmt.Errorf("will exist because : %s \n", err.Error()))
			close(stopCh)
			return
		}
	}()
}

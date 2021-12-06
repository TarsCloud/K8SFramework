package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"math/rand"
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
}

func v1beta1Handler(writer http.ResponseWriter, r *http.Request) {

	writer.Header().Add("Content-Type", "application/json")

	wait := r.URL.Query().Get("wait")

	response := &RestfulResponse{
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

		if err = r.ParseMultipartForm(1 * 1024 * 1024 * 120); err != nil {
			err = fmt.Errorf("parse form error: %s", err.Error())
			response.Status = http.StatusBadRequest
			response.Message = err.Error()
			break
		}

		app := r.FormValue(ServerAppFormKey)
		name := r.FormValue(ServerNameFormKey)
		secret := r.FormValue(ServerSecretFormKey)
		serverType := r.FormValue(ServerTypeFormKey)

		basImage := r.FormValue(BaseImageFormKey)
		basImageSecret := r.FormValue(BaseImageSecretFormKey)

		person := r.FormValue(CreatePersonFormKey)
		mark := r.FormValue(MarkFormKey)

		var multipartServerFile multipart.File
		var multipartFileHandler *multipart.FileHeader
		if multipartServerFile, multipartFileHandler, err = r.FormFile(ServerFileFormKey); err != nil {
			err = fmt.Errorf("parse form error: %s", err.Error())
			response.Status = http.StatusBadRequest
			response.Message = err.Error()
			utilRuntime.HandleError(err)
			break
		}

		idString := fmt.Sprintf("v%s-%x", time.Now().Format("20060102030405"), rand.Intn(0xefffff)+0x100000)

		var serverFile string
		multipartFileName := multipartFileHandler.Filename
		if strings.HasSuffix(multipartFileName, ".tar.gz") || strings.HasSuffix(multipartFileName, ".tgz") {
			serverFile = fmt.Sprintf("%s/%s.%s-%s%s", AbsoluteServerFileSaveDir, app, name, idString, ".tgz")
		} else if strings.HasSuffix(multipartFileName, ".war") || strings.HasSuffix(multipartFileName, ".jar") {
			serverFile = fmt.Sprintf("%s/%s.%s-%s%s", AbsoluteServerFileSaveDir, app, name, idString, filepath.Ext(multipartFileName))
		} else {
			err = fmt.Errorf("unsupported server file type %s", serverFile)
			response.Status = http.StatusBadRequest
			response.Message = err.Error()
			utilRuntime.HandleError(err)
			break
		}

		f, _ := os.OpenFile(serverFile, os.O_CREATE|os.O_WRONLY, 0666)
		if _, err = io.Copy(f, multipartServerFile); err != nil {
			_ = os.Remove(serverFile)
			err = fmt.Errorf("write file(%s) error: %s", serverFile, err.Error())
			response.Status = http.StatusInternalServerError
			response.Message = err.Error()
			utilRuntime.HandleError(err)
			break
		}

		var image string

		if wait != "1" && wait != "true" {
			if image, err = builder.PostTask(idString, timageName, app, name, serverType, serverFile, basImage, basImageSecret, secret, person, mark); err != nil {
				response.Status = http.StatusInternalServerError
				response.Message = err.Error()
				break
			}
			response.Status = http.StatusCreated
			response.Message = http.StatusText(http.StatusCreated)
			response.Result = &BuildResult{
				ID:     idString,
				Image:  image,
				Secret: secret,
				Source: timageName,
			}
			break
		}

		waitChan := make(chan error, 1)
		if image, err = builder.PostTask(idString, timageName, app, name, serverType, serverFile, basImage, basImageSecret, secret, person, mark, withWaitChan(waitChan)); err != nil {
			response.Status = http.StatusInternalServerError
			response.Message = err.Error()
			break
		}

		select {
		case err = <-waitChan:
			if err != nil {
				response.Status = http.StatusInternalServerError
				response.Message = err.Error()
				break
			}
			response.Status = http.StatusOK
			response.Message = http.StatusText(http.StatusOK)
			response.Result = &BuildResult{
				ID:     idString,
				Image:  image,
				Secret: secret,
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

func NewRestfulServer() *RestfulServer {
	return &RestfulServer{}
}

func (s *RestfulServer) Start(stopCh chan struct{}) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1beta1/timage/{timage}/building", func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodPost:
			v1beta1Handler(writer, request)
		default:
			writer.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	srv := &http.Server{
		Addr:              ":80",
		Handler:           router,
		ReadTimeout:       400 * time.Second,
		ReadHeaderTimeout: 60 * time.Second,
		WriteTimeout:      300 * time.Second,
		IdleTimeout:       300 * time.Second,
	}
	go func() {
		err := srv.ListenAndServe()
		// ListenAndServe always returns a non-nil error. After Shutdown or Close,
		// the returned error is ErrServerClosed.
		if err != nil {
			utilRuntime.HandleError(fmt.Errorf("will exist because : %s \n", err.Error()))
			os.Exit(-1)
		}
	}()
}

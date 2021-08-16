package main

import (
	"bufio"
	"bytes"
	"context"
	"credentialprovider"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"io"
	"io/ioutil"
	k8sCoreV1 "k8s.io/api/core/v1"
	"time"
)

func base64EncodeAuth(auth dockerTypes.AuthConfig) string {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(auth); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(buf.Bytes())
}

type DockerClient struct {
	dockerInterface *client.Client
}

type ErrorLine struct {
	Error       string      `json:"error"`
	ErrorDetail ErrorDetail `json:"errorDetail"`
}

type ErrorDetail struct {
	Message string `json:"message"`
}

func (c *DockerClient) waitAndCheck(rd io.ReadCloser) error {
	defer func() { _ = rd.Close() }()
	var lastLine string
	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		lastLine = scanner.Text()
	}
	errLine := &ErrorLine{}
	_ = json.Unmarshal([]byte(lastLine), errLine)
	if errLine.Error != "" {
		return errors.New(errLine.Error)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func (c *DockerClient) pullImage(image string, deadline time.Duration, secrets []*k8sCoreV1.Secret) error {
	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()

	var responseIOReader io.ReadCloser
	var err error

	for true {
		if secrets == nil || len(secrets) == 0 {
			break
		}
		dockerKeyRing, err := credentialprovider.MakeDockerKeyring(secrets...)
		if err != nil {
			break
		}
		credentials, ok := dockerKeyRing.Lookup(image)
		if !ok {
			break
		}
		for _, credential := range credentials {
			auth := &dockerTypes.AuthConfig{
				Username:      credential.Username,
				Password:      credential.Password,
				Auth:          credential.Auth,
				ServerAddress: credential.ServerAddress,
				IdentityToken: credential.IdentityToken,
				RegistryToken: credential.RegistryToken,
			}
			responseIOReader, err = c.dockerInterface.ImagePull(ctx, image, dockerTypes.ImagePullOptions{RegistryAuth: base64EncodeAuth(*auth)})
			if err == nil {
				return c.waitAndCheck(responseIOReader)
			}
		}
		break
	}
	responseIOReader, err = c.dockerInterface.ImagePull(ctx, image, dockerTypes.ImagePullOptions{})
	if err != nil {
		return err
	}
	return c.waitAndCheck(responseIOReader)
}

func (c *DockerClient) buildImage(image string, rootPath string) error {
	tar, _ := archive.TarWithOptions(rootPath, &archive.TarOptions{})
	defer func() { _ = tar.Close() }()

	response, err := c.dockerInterface.ImageBuild(context.Background(), tar, dockerTypes.ImageBuildOptions{
		Tags:   []string{image},
		Remove: true,
	})
	if err != nil {
		return err
	}
	return c.waitAndCheck(response.Body)
}

func (c *DockerClient) pushImage(image string, deadline time.Duration, secrets []*k8sCoreV1.Secret) error {
	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()

	var responseIOReader io.ReadCloser
	var err error

	for true {
		if secrets == nil {
			break
		}
		dockerKeyRing, err := credentialprovider.MakeDockerKeyring(secrets...)
		if err != nil {
			break
		}
		credentials, ok := dockerKeyRing.Lookup(image)
		if !ok {
			break
		}
		for _, credential := range credentials {
			auth := &dockerTypes.AuthConfig{
				Username:      credential.Username,
				Password:      credential.Password,
				Auth:          credential.Auth,
				ServerAddress: credential.ServerAddress,
				IdentityToken: credential.IdentityToken,
				RegistryToken: credential.RegistryToken,
			}
			responseIOReader, err = c.dockerInterface.ImagePush(ctx, image, dockerTypes.ImagePushOptions{RegistryAuth: base64EncodeAuth(*auth)})
			if err == nil {
				return c.waitAndCheck(responseIOReader)
			}
		}
		break
	}
	responseIOReader, err = c.dockerInterface.ImagePush(ctx, image, dockerTypes.ImagePushOptions{})
	if err != nil {
		return err
	}
	return c.waitAndCheck(responseIOReader)
}

func (c *DockerClient) createDockerFile(baseImage string, serverType string, dockerFile string) error {
	dockerFileContent := fmt.Sprintf("FROM %s\nENV ServerType=%s\nCOPY root /\n", baseImage, serverType)
	return ioutil.WriteFile(dockerFile, []byte(dockerFileContent), 0666)
}

func NewDockerClient() (*DockerClient, error) {
	dockerClient := &DockerClient{}
	var err error
	dockerClient.dockerInterface, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return dockerClient, nil
}

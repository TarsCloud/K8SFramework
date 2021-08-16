package image

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"io"
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

func (c *DockerClient) pullImage(image string, deadline time.Duration, secrets *k8sCoreV1.Secret) error {
	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()

	var responseIOReader io.ReadCloser
	var err error

	if secrets == nil {
		responseIOReader, err = c.dockerInterface.ImagePull(ctx, image, dockerTypes.ImagePullOptions{})
		if err == nil {
			return c.waitAndCheck(responseIOReader)
		}
		return err
	}

	dockerKeyRing, err := MakeDockerKeyring(secrets)
	if err != nil {
		return err
	}
	credentials, ok := dockerKeyRing.Lookup(image)
	if !ok {
		return errors.New(fmt.Sprintf("Lookup image: %s error.\n", image))
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
	return nil
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

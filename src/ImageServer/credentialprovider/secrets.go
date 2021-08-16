/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package credentialprovider

import (
	"encoding/json"
	k8sCoreV1 "k8s.io/api/core/v1"
)

// MakeDockerKeyring inspects the passedSecrets to see if they contain any DockerConfig secrets.  If they do,
// then a DockerKeyring is built based on every hit and unioned with the defaultKeyring.
// If they do not, then the default keyring is returned
func MakeDockerKeyring(passedSecrets ...*k8sCoreV1.Secret) (*BasicDockerKeyring, error) {
	passedCredentials := []DockerConfig{}
	for _, passedSecret := range passedSecrets {
		if passedSecret == nil {
			continue
		}
		if dockerConfigJSONBytes, dockerConfigJSONExists := passedSecret.Data[k8sCoreV1.DockerConfigJsonKey]; (passedSecret.Type == k8sCoreV1.SecretTypeDockerConfigJson) && dockerConfigJSONExists && (len(dockerConfigJSONBytes) > 0) {
			dockerConfigJSON := DockerConfigJSON{}
			if err := json.Unmarshal(dockerConfigJSONBytes, &dockerConfigJSON); err != nil {
				return nil, err
			}
			passedCredentials = append(passedCredentials, dockerConfigJSON.Auths)
		} else if dockercfgBytes, dockercfgExists := passedSecret.Data[k8sCoreV1.DockerConfigKey]; (passedSecret.Type == k8sCoreV1.SecretTypeDockercfg) && dockercfgExists && (len(dockercfgBytes) > 0) {
			dockercfg := DockerConfig{}
			if err := json.Unmarshal(dockercfgBytes, &dockercfg); err != nil {
				return nil, err
			}
			passedCredentials = append(passedCredentials, dockercfg)
		}
	}

	if len(passedCredentials) > 0 {
		basicKeyring := &BasicDockerKeyring{}
		for _, currCredentials := range passedCredentials {
			basicKeyring.Add(currCredentials)
		}
		return basicKeyring, nil
	}
	return nil, nil
}

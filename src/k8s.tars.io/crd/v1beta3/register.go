/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Tag 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta3

import (
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupName is the group name use in this package
const GroupName = "k8s.tars.io"
const Version = "v1beta3"

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: Version}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)
	localSchemeBuilder = &SchemeBuilder
	AddToScheme        = localSchemeBuilder.AddToScheme
)

// Adds the list of known types to the given scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&TServer{},
		&TServerList{},
		&TEndpoint{},
		&TEndpointList{},
		&TTemplate{},
		&TTemplateList{},
		&TExitedRecord{},
		&TExitedRecordList{},
		&TTree{},
		&TTreeList{},
		&TConfig{},
		&TConfigList{},
		&TAccount{},
		&TAccountList{},
		&TImage{},
		&TImageList{},
		&TFrameworkConfig{},
		&TFrameworkConfigList{},
	)
	k8sMetaV1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

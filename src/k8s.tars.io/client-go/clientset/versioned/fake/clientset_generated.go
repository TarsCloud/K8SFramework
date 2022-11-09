/*
Copyright The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/testing"
	clientset "k8s.tars.io/client-go/clientset/versioned"
	tarsv1beta1 "k8s.tars.io/client-go/clientset/versioned/typed/tars/v1beta1"
	faketarsv1beta1 "k8s.tars.io/client-go/clientset/versioned/typed/tars/v1beta1/fake"
	tarsv1beta2 "k8s.tars.io/client-go/clientset/versioned/typed/tars/v1beta2"
	faketarsv1beta2 "k8s.tars.io/client-go/clientset/versioned/typed/tars/v1beta2/fake"
	tarsv1beta3 "k8s.tars.io/client-go/clientset/versioned/typed/tars/v1beta3"
	faketarsv1beta3 "k8s.tars.io/client-go/clientset/versioned/typed/tars/v1beta3/fake"
)

// NewSimpleClientset returns a clientset that will respond with the provided objects.
// It's backed by a very simple object tracker that processes creates, updates and deletions as-is,
// without applying any validations and/or defaults. It shouldn't be considered a replacement
// for a real clientset and is mostly useful in simple unit tests.
func NewSimpleClientset(objects ...runtime.Object) *Clientset {
	o := testing.NewObjectTracker(scheme, codecs.UniversalDecoder())
	for _, obj := range objects {
		if err := o.Add(obj); err != nil {
			panic(err)
		}
	}

	cs := &Clientset{tracker: o}
	cs.discovery = &fakediscovery.FakeDiscovery{Fake: &cs.Fake}
	cs.AddReactor("*", "*", testing.ObjectReaction(o))
	cs.AddWatchReactor("*", func(action testing.Action) (handled bool, ret watch.Interface, err error) {
		gvr := action.GetResource()
		ns := action.GetNamespace()
		watch, err := o.Watch(gvr, ns)
		if err != nil {
			return false, nil, err
		}
		return true, watch, nil
	})

	return cs
}

// Clientset implements clientset.Interface. Meant to be embedded into a
// struct to get a default implementation. This makes faking out just the method
// you want to test easier.
type Clientset struct {
	testing.Fake
	discovery *fakediscovery.FakeDiscovery
	tracker   testing.ObjectTracker
}

func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	return c.discovery
}

func (c *Clientset) Tracker() testing.ObjectTracker {
	return c.tracker
}

var (
	_ clientset.Interface = &Clientset{}
	_ testing.FakeClient  = &Clientset{}
)

// TarsV1beta1 retrieves the TarsV1beta1Client
func (c *Clientset) TarsV1beta1() tarsv1beta1.TarsV1beta1Interface {
	return &faketarsv1beta1.FakeTarsV1beta1{Fake: &c.Fake}
}

// TarsV1beta2 retrieves the TarsV1beta2Client
func (c *Clientset) TarsV1beta2() tarsv1beta2.TarsV1beta2Interface {
	return &faketarsv1beta2.FakeTarsV1beta2{Fake: &c.Fake}
}

// TarsV1beta3 retrieves the TarsV1beta3Client
func (c *Clientset) TarsV1beta3() tarsv1beta3.TarsV1beta3Interface {
	return &faketarsv1beta3.FakeTarsV1beta3{Fake: &c.Fake}
}

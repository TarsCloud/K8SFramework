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

package v1beta2

import (
	"context"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
	v1beta2 "k8s.tars.io/apis/tars/v1beta2"
	scheme "k8s.tars.io/client-go/clientset/versioned/scheme"
)

// TFrameworkConfigsGetter has a method to return a TFrameworkConfigInterface.
// A group's client should implement this interface.
type TFrameworkConfigsGetter interface {
	TFrameworkConfigs(namespace string) TFrameworkConfigInterface
}

// TFrameworkConfigInterface has methods to work with TFrameworkConfig resources.
type TFrameworkConfigInterface interface {
	Create(ctx context.Context, tFrameworkConfig *v1beta2.TFrameworkConfig, opts v1.CreateOptions) (*v1beta2.TFrameworkConfig, error)
	Update(ctx context.Context, tFrameworkConfig *v1beta2.TFrameworkConfig, opts v1.UpdateOptions) (*v1beta2.TFrameworkConfig, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1beta2.TFrameworkConfig, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1beta2.TFrameworkConfigList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta2.TFrameworkConfig, err error)
	TFrameworkConfigExpansion
}

// tFrameworkConfigs implements TFrameworkConfigInterface
type tFrameworkConfigs struct {
	client rest.Interface
	ns     string
}

// newTFrameworkConfigs returns a TFrameworkConfigs
func newTFrameworkConfigs(c *TarsV1beta2Client, namespace string) *tFrameworkConfigs {
	return &tFrameworkConfigs{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the tFrameworkConfig, and returns the corresponding tFrameworkConfig object, and an error if there is any.
func (c *tFrameworkConfigs) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta2.TFrameworkConfig, err error) {
	result = &v1beta2.TFrameworkConfig{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("tframeworkconfigs").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of TFrameworkConfigs that match those selectors.
func (c *tFrameworkConfigs) List(ctx context.Context, opts v1.ListOptions) (result *v1beta2.TFrameworkConfigList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1beta2.TFrameworkConfigList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("tframeworkconfigs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested tFrameworkConfigs.
func (c *tFrameworkConfigs) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("tframeworkconfigs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a tFrameworkConfig and creates it.  Returns the server's representation of the tFrameworkConfig, and an error, if there is any.
func (c *tFrameworkConfigs) Create(ctx context.Context, tFrameworkConfig *v1beta2.TFrameworkConfig, opts v1.CreateOptions) (result *v1beta2.TFrameworkConfig, err error) {
	result = &v1beta2.TFrameworkConfig{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("tframeworkconfigs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(tFrameworkConfig).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a tFrameworkConfig and updates it. Returns the server's representation of the tFrameworkConfig, and an error, if there is any.
func (c *tFrameworkConfigs) Update(ctx context.Context, tFrameworkConfig *v1beta2.TFrameworkConfig, opts v1.UpdateOptions) (result *v1beta2.TFrameworkConfig, err error) {
	result = &v1beta2.TFrameworkConfig{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("tframeworkconfigs").
		Name(tFrameworkConfig.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(tFrameworkConfig).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the tFrameworkConfig and deletes it. Returns an error if one occurs.
func (c *tFrameworkConfigs) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("tframeworkconfigs").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *tFrameworkConfigs) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("tframeworkconfigs").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched tFrameworkConfig.
func (c *tFrameworkConfigs) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta2.TFrameworkConfig, err error) {
	result = &v1beta2.TFrameworkConfig{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("tframeworkconfigs").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
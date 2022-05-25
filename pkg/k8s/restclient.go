// Copyright 2022 Antoine Martin
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package k8s

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type RESTClientGetter struct {
	clientconfig clientcmd.ClientConfig
}

func NewRESTClientForDistribution(distributionName string) (*RESTClientGetter, error) {
	kubeConfigFile := fmt.Sprintf(wslKubeconfigFormat, distributionName)

	config, err := clientcmd.LoadFromFile(kubeConfigFile)
	if err != nil {
		return nil, err
	}
	return NewRESTClientFromConfig(config), nil
}

func NewRESTClientFromConfig(config *api.Config) *RESTClientGetter {
	return &RESTClientGetter{clientcmd.NewDefaultClientConfig(api.Config(*config), nil)}
}

func (r *RESTClientGetter) ToRESTConfig() (*rest.Config, error) {
	return r.clientconfig.ClientConfig()
}

func (r *RESTClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	restconfig, err := r.clientconfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	dc, err := discovery.NewDiscoveryClientForConfig(restconfig)
	if err != nil {
		return nil, err
	}
	return memory.NewMemCacheClient(dc), nil
}

func (r *RESTClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	dc, err := r.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	return restmapper.NewDeferredDiscoveryRESTMapper(dc), nil
}

func (r *RESTClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return r.clientconfig
}

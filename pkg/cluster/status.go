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

package cluster

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kaweezle/kaweezle/pkg/k8s"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/polymorphichelpers"
)

type WorkloadState struct {
	Namespace string
	Name      string
	Ok        bool
	Message   string
}

func OkString(b bool) string {
	if b {
		return "🟩"
	}
	return "🟥"
}

func (r *WorkloadState) LongString() string {

	return fmt.Sprintf("%s %-20s %-54s %s", OkString(r.Ok), r.Namespace, r.Name, r.Message)
}

func (r *WorkloadState) String() string {

	return fmt.Sprintf("%s/%s:%s", r.Namespace, r.Name, OkString(r.Ok))
}

var ApplicationSchemaGroupVersionKind = schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "Application"}

type SyncStatus struct {
	Status string `json:"status" protobuf:"bytes,1,opt,name=status,casttype=SyncStatusCode"`
}

type HealthStatus struct {
	// Status holds the status code of the application or resource
	Status string `json:"status,omitempty" protobuf:"bytes,1,opt,name=status"`
	// Message is a human-readable informational message describing the health status
	Message string `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`
}

type ApplicationStatus struct {
	Sync   SyncStatus   `json:"sync,omitempty" protobuf:"bytes,2,opt,name=sync"`
	Health HealthStatus `json:"health,omitempty" protobuf:"bytes,3,opt,name=health"`
}

type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Status            ApplicationStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type ApplicationStatusViewer struct{}

func StatusViewerFor(kind schema.GroupKind) (polymorphichelpers.StatusViewer, error) {
	if kind == ApplicationSchemaGroupVersionKind.GroupKind() {
		return &ApplicationStatusViewer{}, nil
	}
	return polymorphichelpers.StatusViewerFor(kind)
}

func (s *ApplicationStatusViewer) Status(obj runtime.Unstructured, revision int64) (string, bool, error) {
	application := &Application{}

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), application)
	if err != nil {
		return "", false, fmt.Errorf("failed to convert %T to %T: %v", obj, application, err)
	}

	healthStatusString := application.Status.Health.Status
	syncStatusString := application.Status.Sync.Status

	msg := fmt.Sprintf("application \"%s\" sync status: %s, health status: %s", application.Name, syncStatusString, healthStatusString)
	return msg, healthStatusString == "Healthy" && syncStatusString == "Synced", nil
}

func HasApplications(client *k8s.RESTClientGetter) (has bool, err error) {
	var mapper meta.RESTMapper
	if mapper, err = client.ToRESTMapper(); err != nil {
		return
	}

	_, err = mapper.RESTMapping(ApplicationSchemaGroupVersionKind.GroupKind(), ApplicationSchemaGroupVersionKind.Version)
	if err != nil {
		if meta.IsNoMatchError(err) {
			err = nil
		} else {
			return
		}
	} else {
		has = true
	}
	return
}

func AllWorkloadStates(client *k8s.RESTClientGetter) (result []*WorkloadState, err error) {
	var _result []*WorkloadState

	var resourceTypes = "deployments,statefulsets,daemonsets"
	var hasApplications bool
	if hasApplications, err = HasApplications(client); err != nil {
		return
	}
	if hasApplications {
		resourceTypes += ",applications"
	}

	r := resource.NewBuilder(client).
		Unstructured().
		AllNamespaces(true).
		ResourceTypeOrNameArgs(true, resourceTypes).
		ContinueOnError().
		Flatten().
		Do()

	var infos []*resource.Info
	if infos, err = r.Infos(); err != nil {
		return
	}

	for _, info := range infos {
		var u map[string]interface{}

		if u, err = runtime.DefaultUnstructuredConverter.ToUnstructured(info.Object); err != nil {
			return
		}

		var v polymorphichelpers.StatusViewer
		if v, err = /* polymorphichelpers. */ StatusViewerFor(info.Object.GetObjectKind().GroupVersionKind().GroupKind()); err != nil {
			return
		}

		var msg string
		var ok bool
		if msg, ok, err = v.Status(&unstructured.Unstructured{Object: u}, 0); err != nil {
			return
		}
		_result = append(_result, &WorkloadState{info.Namespace, info.ObjectName(), ok, strings.TrimSuffix(msg, "\n")})
	}
	sort.SliceStable(_result, func(i, j int) bool {
		return _result[i].String() < _result[j].String()
	})
	result = _result
	return
}

type WorkloadStateCallbackFunc func(state bool, total int, ready []*WorkloadState, unready []*WorkloadState)

func AreWorkloadsReady(client *k8s.RESTClientGetter, callback WorkloadStateCallbackFunc) wait.ConditionFunc {
	return func() (bool, error) {
		states, err := AllWorkloadStates(client)
		if err != nil {
			return false, err
		}
		var result bool = true
		var ready, unready []*WorkloadState
		for _, state := range states {
			if !state.Ok {
				result = false
				unready = append(unready, state)
			} else {
				ready = append(ready, state)
			}
		}

		if callback != nil {
			callback(result, len(states), ready, unready)
		}

		return result, nil
	}
}

func WaitForWorkloads(client *k8s.RESTClientGetter, timeout time.Duration, callback WorkloadStateCallbackFunc) error {
	return wait.PollImmediate(time.Second*time.Duration(2), timeout, AreWorkloadsReady(client, callback))
}

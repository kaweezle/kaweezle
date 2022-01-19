/*
Copyright © 2021 Antoine Martin <antoine@openance.com>

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
package k8s

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

const (
	wslKubeconfigFormat = `\\wsl$\%s\root\.kube\config`
)

func MergeKubernetesConfig(distributionName string) (err error) {

	loadingRules := clientcmd.ClientConfigLoadingRules{
		Precedence: []string{fmt.Sprintf(wslKubeconfigFormat, distributionName), clientcmd.RecommendedHomeFile},
	}

	var mergedConfig *api.Config

	if mergedConfig, err = loadingRules.Load(); err == nil {
		log.WithFields(log.Fields{
			"distribution_name": distributionName,
			"kubeConfigFile":    clientcmd.RecommendedFileName,
		}).Trace("Writing config")
		err = clientcmd.WriteToFile(*mergedConfig, clientcmd.RecommendedHomeFile)
	} else {
		log.WithError(err).WithField("distribution_name", distributionName).Error("Loading configuration")
	}

	return err
}

func RemoveKubernetesConfig(distributionName string) (err error) {
	loadingRules := clientcmd.ClientConfigLoadingRules{
		Precedence: []string{clientcmd.RecommendedHomeFile},
	}

	var config *api.Config

	if config, err = loadingRules.Load(); err == nil {
		delete(config.Clusters, distributionName)
		delete(config.Contexts, distributionName)
		delete(config.AuthInfos, distributionName)
		log.WithFields(log.Fields{
			"distribution_name": distributionName,
			"kubeConfigFile":    clientcmd.RecommendedFileName,
		}).Trace("Writing config")
		err = clientcmd.WriteToFile(*config, clientcmd.RecommendedHomeFile)
	} else {
		log.WithError(err).WithField("distribution_name", distributionName).Error("Loading configuration")
	}
	return
}

func ClientSetForDistribution(distributionName string) (clientset *kubernetes.Clientset, err error) {

	kubeConfigFile := fmt.Sprintf(wslKubeconfigFormat, distributionName)

	log.WithFields(log.Fields{
		"distribution_name": distributionName,
		"kubeConfigFile":    kubeConfigFile,
	}).Trace("Loading config")

	var config *rest.Config

	if config, err = clientcmd.BuildConfigFromFlags("", kubeConfigFile); err == nil {
		clientset, err = kubernetes.NewForConfig(config)
	} else {
		log.WithError(err).WithField("distribution_name", distributionName).Error("Loading configuration")
	}

	return
}

// IsPodReady returns false if the Pod Status is nil
func IsPodReady(pod *v1.Pod) bool {
	condition := getPodReadyCondition(&pod.Status)
	return condition != nil && condition.Status == v1.ConditionTrue
}

func getPodReadyCondition(status *v1.PodStatus) *v1.PodCondition {
	for i := range status.Conditions {
		if status.Conditions[i].Type == v1.PodReady {
			return &status.Conditions[i]
		}
	}
	return nil
}

func GetPodsSeparatedByStatus(pods []v1.Pod) (active, unready, stopped []*v1.Pod) {
	for _, pod := range pods {
		switch pod.Status.Phase {
		case v1.PodRunning:
			if IsPodReady(&pod) {
				active = append(active, &pod)
			} else {
				unready = append(unready, &pod)
			}
		case v1.PodPending, v1.PodUnknown:
			unready = append(unready, &pod)
		default:
			stopped = append(stopped, &pod)
		}
	}

	return active, unready, stopped
}

func GetPodStatus(clientset kubernetes.Interface) (active, unready, stopped []*v1.Pod, err error) {
	var list *v1.PodList
	if list, err = clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{}); err != nil {
		return
	}
	active, unready, stopped = GetPodsSeparatedByStatus(list.Items)
	return
}

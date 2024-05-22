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
package cluster

import (
	"fmt"
	"time"

	"github.com/kaweezle/kaweezle/pkg/k8s"
	"github.com/kaweezle/kaweezle/pkg/logger"
	"github.com/kaweezle/kaweezle/pkg/wsl"
	log "github.com/sirupsen/logrus"
	"github.com/yuk7/wsllib-go"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type ClusterStatus int16

const (
	Undefined ClusterStatus = iota
	Uninstalled
	Installed
	Started
)

var startClusterFields = log.Fields{
	logger.TaskKey: "Start Cluster",
}

var stopClusterFields = log.Fields{
	logger.TaskKey: "Stop Cluster",
}

var waitClusterFields = log.Fields{
	logger.TaskKey: "Wait for cluster to settle",
}

func (s ClusterStatus) String() (r string) {

	switch s {
	case Undefined:
		r = "undefined"
	case Uninstalled:
		r = "uninstalled"
	case Installed:
		r = "installed"
	case Started:
		r = "started"
	}
	return
}

func GetClusterStatus(distributionName string) (status ClusterStatus, err error) {

	status = Uninstalled

	if wsllib.WslIsDistributionRegistered(distributionName) {
		status = Installed
		var distribution wsl.DistributionInformation
		if distribution, err = wsl.GetDistribution(distributionName); err == nil {
			log.WithFields(log.Fields{
				"distribution":      distribution,
				"distribution_name": distributionName,
			}).Trace("Found distribution")
			if distribution.State == wsl.Running {
				status = Started
			}
		} else {
			log.WithError(err).WithField("distribution_name", distributionName).Warning("Couldn't get distribution")
		}
	}

	return
}

func StartCluster(distributionName string, logLevel string) (err error) {
	startCommand := fmt.Sprintf("/sbin/iknite '--json' -v %s '--cluster-name' %s start", logLevel, distributionName)
	log.WithFields(startClusterFields).WithFields(log.Fields{
		"distribution_name": distributionName,
		"command":           startCommand,
	}).Info("Starting kubernetes...")

	var exitCode uint32
	exitCode, err = wsl.LaunchAndPipe(distributionName, startCommand, true, startClusterFields)
	if exitCode != 0 {
		err = fmt.Errorf("command %v exited with error code %d", startCommand, exitCode)
	}
	log.WithError(err).WithFields(startClusterFields).Info("Kubernetes started")

	return
}

func StopCluster(distributionName string) (err error) {
	log.WithFields(stopClusterFields).WithFields(log.Fields{
		"distribution_name": distributionName,
	}).Info("Stopping kubernetes...")
	var info wsl.DistributionInformation
	info, err = wsl.GetDistribution(distributionName)
	if err != nil {
		return
	}
	if info.State != wsl.Running {
		log.WithFields(stopClusterFields).Warnf("Distribution %s not running", distributionName)
		return
	} else {
		stopCommand := "/sbin/rc-service iknite stop"
		log.WithFields(stopClusterFields).WithFields(log.Fields{
			"distribution_name": distributionName,
			"command":           stopCommand,
		}).Info("Stopping kubernetes...")
		var exitCode uint32
		exitCode, _ = wsl.LaunchAndPipe(distributionName, stopCommand, true, stopClusterFields)
		if exitCode != 0 {
			err = fmt.Errorf("command %v exited with error code %d", stopCommand, exitCode)
			log.WithError(err).WithFields(stopClusterFields).Info("Kubernetes stopped")
		}
		err = wsl.StopDistribution(distributionName)
		log.WithError(err).WithFields(stopClusterFields).Info("Kubernetes stopped")
	}
	return
}

func arePodsReady(c kubernetes.Interface, fields *log.Fields) wait.ConditionFunc {
	return func() (bool, error) {

		active, unready, stopped, err := k8s.GetPodStatus(c)
		if err != nil {
			return false, err
		}
		log.WithFields(*fields).WithFields(log.Fields{
			"active":  len(active),
			"unready": len(unready),
			"stopped": len(stopped),
		}).Infof("active: %d, unready: %d, stopped:%d", len(active), len(unready), len(stopped))
		if len(stopped) > 0 {
			return false, fmt.Errorf("stopped pods: %d", len(stopped))
		}
		return len(active) > 0 && len(unready) == 0, nil
	}
}

func waitForPodsReady(c kubernetes.Interface, timeout time.Duration, fields *log.Fields) error {
	return wait.PollImmediate(time.Second, timeout, arePodsReady(c, fields))
}

func WaitForCluster(distributionName string, timeout time.Duration) (err error) {
	log.WithFields(waitClusterFields).WithFields(log.Fields{
		"distribution_name": distributionName,
	}).Info("Wait for kubernetes...")

	var client *k8s.RESTClientGetter
	if client, err = k8s.NewRESTClientForDistribution(distributionName); err != nil {
		return
	}

	err = WaitForWorkloads(client, timeout, func(state bool, total int, ready, unready []*WorkloadState) {
		log.WithFields(waitClusterFields).WithFields(log.Fields{
			"total":   total,
			"ready":   len(ready),
			"unready": len(unready),
		}).Infof("Workloads total: %d, ready: %d, unready: %d", total, len(ready), len(unready))
	})

	if err != nil {
		log.WithError(err).WithFields(waitClusterFields).Error("Kubernetes not ready")
	} else {
		log.WithError(err).WithFields(waitClusterFields).Info("Cluster ready")
	}

	return
}

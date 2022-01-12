package cluster

import (
	"fmt"
	"time"

	"github.com/antoinemartin/kaweezle/pkg/k8s"
	"github.com/antoinemartin/kaweezle/pkg/logger"
	"github.com/antoinemartin/kaweezle/pkg/wsl"
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
	startCommand := fmt.Sprintf("/kwsl --json -v %s --name %s start", logLevel, distributionName)
	log.WithFields(startClusterFields).WithFields(log.Fields{
		"distribution_name": distributionName,
		"command":           startCommand,
	}).Info("Starting kubernetes...")

	_, err = wsl.LaunchAndPipe(distributionName, startCommand, true, startClusterFields)
	log.WithError(err).WithFields(startClusterFields).Info("Kubernetes started")

	return
}

func StopCluster(distributionName string) (err error) {
	log.WithFields(stopClusterFields).WithFields(log.Fields{
		"distribution_name": distributionName,
	}).Info("Stopping kubernetes...")
	err = wsl.StopDistribution(distributionName)
	log.WithError(err).WithFields(stopClusterFields).Info("Kubernetes stopped")
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
		return len(unready) == 0, nil
	}
}

func waitForPodsReady(c kubernetes.Interface, timeout time.Duration, fields *log.Fields) error {
	return wait.PollImmediate(time.Second, timeout, arePodsReady(c, fields))
}

func WaitForCluster(distributionName string, timeout time.Duration) (err error) {
	log.WithFields(waitClusterFields).WithFields(log.Fields{
		"distribution_name": distributionName,
	}).Info("Wait for kubernetes...")

	var client *kubernetes.Clientset
	if client, err = k8s.ClientSetForDistribution(distributionName); err != nil {
		return
	}

	err = waitForPodsReady(client, timeout, &waitClusterFields)

	if err != nil {
		log.WithError(err).WithFields(waitClusterFields).Error("Kubernetes not ready")
	} else {
		log.WithError(err).WithFields(waitClusterFields).Info("Cluster ready")
	}

	return
}

package cluster

import (
	"fmt"

	"github.com/antoinemartin/kaweezle/pkg/logger"
	"github.com/antoinemartin/kaweezle/pkg/wsl"
	log "github.com/sirupsen/logrus"
	"github.com/yuk7/wsllib-go"
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

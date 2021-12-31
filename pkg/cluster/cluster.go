package cluster

import (
	"fmt"

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

func StartCluster(distributionName string, logLevel string, json bool) (err error) {
	jsonArg := ""
	k8sLevel := logLevel
	if json {
		jsonArg = " --json"
		if k8sLevel != "trace" {
			k8sLevel = "debug"
		}
	}
	startCommand := fmt.Sprintf("/k8wsl -v %s%s --name %s start", k8sLevel, jsonArg, distributionName)
	log.WithFields(log.Fields{
		"distribution_name": distributionName,
		"command":           startCommand,
	}).Info("Starting kubernetes...")

	_, err = wsllib.WslLaunchInteractive(distributionName, startCommand, true)

	return
}

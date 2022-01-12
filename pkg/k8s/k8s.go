package k8s

import (
	"fmt"

	log "github.com/sirupsen/logrus"
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

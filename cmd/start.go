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
package cmd

import (
	"os"
	"time"

	"github.com/kaweezle/kaweezle/pkg/cluster"
	"github.com/kaweezle/kaweezle/pkg/config"
	"github.com/kaweezle/kaweezle/pkg/k8s"
	"github.com/kaweezle/kaweezle/pkg/rootfs"
	"github.com/kaweezle/kaweezle/pkg/wsl"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/runtime"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	DefaultClusterWaitTimeout = 45
)

var (
	ClusterWaitTimeout   = DefaultClusterWaitTimeout
	ConfigurationOptions = config.NewConfigurationOptions()
)

// NewStartCommand creates a new start command
func NewStartCommand() *cobra.Command {
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the cluster",
		Long:  `Start the cluster when it is not started.`,
		Run:   performStart,
	}
	flags := startCmd.Flags()

	flags.StringVarP(&rootfs.TarFilePath, "root", "r", rootfs.DefaultTarFilePath, "The root file system to install")
	flags.IntVarP(&ClusterWaitTimeout, "timeout", "t", DefaultClusterWaitTimeout, "The time (in seconds) to wait for the cluster to settle")
	AddConfigurationFlags(flags, ConfigurationOptions)

	return startCmd
}

func AddConfigurationFlags(flags *pflag.FlagSet, options *config.ConfigurationOptions) {

	flags.StringVar(&options.AgeKeyFile, "age-key-file", options.AgeKeyFile, "The path to the age key file")
	flags.StringVar(&options.SshKeyFile, "ssh-key-file", options.SshKeyFile, "The path to the ssh key file")
	flags.StringVar(&options.KustomizeUrl, "kustomize-url", options.KustomizeUrl, "The URL to the kustomization to perform on start (default none)")
	flags.StringVar(&options.PersistentIPAddress, "ip-address", options.PersistentIPAddress, "The persistent IP address to use for the WSL distribution")
	flags.StringArrayVar(&options.DomainNames, "domain-name", options.DomainNames, "Domain names to associate locally with the cluster")
	flags.StringArrayVar(&options.SshHosts, "ssh-hosts", options.SshHosts, "Hosts to add to the ~/.ssh/known_hosts file")
}

func performStart(cmd *cobra.Command, args []string) {
	status, err := cluster.GetClusterStatus(DistributionName)
	cobra.CheckErr(err)
	if status != cluster.Started {
		if status == cluster.Uninstalled {
			if rootfs.TarFilePath == "" {
				rootfs.TarFilePath = rootfs.DefaultTarFilePath
				cobra.CheckErr(rootfs.EnsureRootFS(rootfs.TarFilePath, &UpdateRootFSFields))
			} else {
				// Check if rootfs.TarFilePath is a valid file path
				if _, err := os.Stat(rootfs.TarFilePath); os.IsNotExist(err) {
					cobra.CheckErr(errors.Wrapf(err, "rootfs file %s does not exist", rootfs.TarFilePath))
				}
			}

			installationDir, err := rootfs.EnsureWSLDirectory(rootfs.HomeDir, DistributionName)
			cobra.CheckErr(err)
			cobra.CheckErr(wsl.RegisterDistribution(DistributionName, rootfs.TarFilePath, installationDir))
			cobra.CheckErr(config.Configure(DistributionName, ConfigurationOptions))
			status = cluster.Installed
		}
		if status != cluster.Installed {
			log.Fatalf("Cluster %s in bad status: %v", DistributionName, status)
		}
		cobra.CheckErr(cluster.StartCluster(DistributionName, LogLevel))
		cobra.CheckErr(k8s.MergeKubernetesConfig(DistributionName))
	}
	if ClusterWaitTimeout > 0 {
		runtime.ErrorHandlers = runtime.ErrorHandlers[:0]
		err = cluster.WaitForCluster(DistributionName, time.Second*time.Duration(ClusterWaitTimeout))
		if err != nil {
			log.WithError(err).WithField("distrib_name", DistributionName).Infof("To continue waiting, issue the following command: %s status -w", commandName)
		}
	} else {
		log.WithField("distrib_name", DistributionName).Info("No wait for cluster settling")
	}

}

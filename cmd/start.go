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
	"path/filepath"
	"time"

	"github.com/antoinemartin/kaweezle/pkg/cluster"
	"github.com/antoinemartin/kaweezle/pkg/k8s"
	"github.com/antoinemartin/kaweezle/pkg/rootfs"
	"github.com/antoinemartin/kaweezle/pkg/wsl"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

const (
	DefaultClusterWaitTimeout = 45
)

var (
	ClusterWaitTimeout = DefaultClusterWaitTimeout
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the cluster",
	Long:  `Start the cluster when it is not started.`,
	Run:   performStart,
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().StringVarP(&rootfs.TarFilePath, "root", "r", rootfs.DefaultTarFilePath, "The root file system to install")
	startCmd.Flags().IntVarP(&ClusterWaitTimeout, "timeout", "t", DefaultClusterWaitTimeout, "The time (in seconds) to wait for the cluster to settle")
}

func performStart(cmd *cobra.Command, args []string) {
	status, err := cluster.GetClusterStatus(DistributionName)
	cobra.CheckErr(err)
	if status != cluster.Started {
		if status == cluster.Uninstalled {
			cobra.CheckErr(rootfs.EnsureRootFS(rootfs.TarFilePath, &UpdateRootFSFields))
			installationDir := filepath.Dir(rootfs.TarFilePath)
			cobra.CheckErr(wsl.RegisterDistribution(DistributionName, rootfs.TarFilePath, installationDir))
			status = cluster.Installed
		}
		if status != cluster.Installed {
			log.Fatalf("Cluster %s in bad status: %v", DistributionName, status)
		}
		cobra.CheckErr(cluster.StartCluster(DistributionName, LogLevel))
		cobra.CheckErr(k8s.MergeKubernetesConfig(DistributionName))
	}
	if ClusterWaitTimeout > 0 {
		cobra.CheckErr(cluster.WaitForCluster(DistributionName, time.Second*time.Duration(ClusterWaitTimeout)))
	} else {
		log.WithField("distrib_name", DistributionName).Info("No wait for cluster settling")
	}
}

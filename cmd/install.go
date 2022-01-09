/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

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
	"fmt"
	"path/filepath"

	"github.com/antoinemartin/kaweezle/pkg/cluster"
	"github.com/antoinemartin/kaweezle/pkg/k8s"
	"github.com/antoinemartin/kaweezle/pkg/rootfs"
	"github.com/antoinemartin/kaweezle/pkg/wsl"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/yuk7/wsllib-go"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Kaweezle distribution",
	Long: `Installs the kaweezle distribution and starts the cluster.

Examples:

> kaweezle install --root rootfs.tar.gz
`,
	Run: performInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().StringVarP(&rootfs.TarFilePath, "root", "r", rootfs.DefaultTarFilePath, "The root file system to install")
}

func performInstall(cmd *cobra.Command, args []string) {
	if wsllib.WslIsDistributionRegistered(DistributionName) {
		cobra.CheckErr(fmt.Sprintf("The distribution %s is already registered.", DistributionName))
	}

	cobra.CheckErr(rootfs.EnsureRootFS(rootfs.TarFilePath, &UpdateRootFSFields))

	installationDir := filepath.Dir(rootfs.TarFilePath)
	log.WithFields(log.Fields{
		"rootfs":       rootfs.TarFilePath,
		"distrib_name": DistributionName,
		"install_dir":  installationDir,
	}).Info("Registering distribution...")
	cobra.CheckErr(wsl.RegisterDistribution(DistributionName, rootfs.TarFilePath, installationDir))

	cobra.CheckErr(cluster.StartCluster(DistributionName, LogLevel))
	log.WithFields(log.Fields{
		"distrib_name": DistributionName,
		"install_dir":  installationDir,
	}).Info("Adding cluster to kubeconfig...")
	cobra.CheckErr(k8s.MergeKubernetesConfig(DistributionName))
}

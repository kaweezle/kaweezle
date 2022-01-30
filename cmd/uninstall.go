/*
Copyright © 2021 Antoine Martin antoinemartin@openance.com

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

	"github.com/kaweezle/kaweezle/pkg/k8s"
	"github.com/kaweezle/kaweezle/pkg/rootfs"
	"github.com/pterm/pterm"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/yuk7/wsllib-go"
)

// uninstallCmd represents the uninstall command
var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the distribution",
	Long: `Uninstalls the distribution. For example:

> kaweezle uninstall
`,
	Run: performUninstall,
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}

func performUninstall(cmd *cobra.Command, args []string) {
	if !wsllib.WslIsDistributionRegistered(DistributionName) {
		cobra.CheckErr(fmt.Sprintf("The distribution %s is not registered.", DistributionName))
	}

	log.WithFields(log.Fields{
		"distrib_name": DistributionName,
	}).Infof("Uninstall %s WSL distribution", pterm.Bold.Sprint(DistributionName))
	cobra.CheckErr(wsllib.WslUnregisterDistribution(DistributionName))
	log.WithFields(log.Fields{
		"distrib_name": DistributionName,
	}).Infof("Remove %s kube context", pterm.Bold.Sprint(DistributionName))
	cobra.CheckErr(k8s.RemoveKubernetesConfig(DistributionName))
	log.WithFields(log.Fields{
		"distrib_name": DistributionName,
	}).Infof("Remove %s directory", pterm.Bold.Sprint(DistributionName))
	cobra.CheckErr(rootfs.RemoveWSLDirectory(rootfs.HomeDir, DistributionName))
}

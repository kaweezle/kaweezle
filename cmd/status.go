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
	"github.com/kaweezle/kaweezle/pkg/cluster"
	"github.com/kaweezle/kaweezle/pkg/k8s"
	"github.com/pterm/pterm"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Current status of the cluster",
	Long: `Gives the status of the cluster. Example:

> kaweezle status
`,
	Run: performStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// statusCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// statusCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func performStatus(cmd *cobra.Command, args []string) {
	status, err := cluster.GetClusterStatus(DistributionName)
	cobra.CheckErr(err)
	log.Infof("Cluster %s is %v.", pterm.Bold.Sprint(DistributionName), pterm.Bold.Sprint(status))
	var client *kubernetes.Clientset
	client, err = k8s.ClientSetForDistribution(DistributionName)
	cobra.CheckErr(err)
	if status == cluster.Started {
		active, unready, stopped, err := k8s.GetPodStatus(client)
		cobra.CheckErr(err)
		log.WithFields(log.Fields{
			"active":  len(active),
			"unready": len(unready),
			"stopped": len(stopped),
		}).Info("Pods status")
	}
}

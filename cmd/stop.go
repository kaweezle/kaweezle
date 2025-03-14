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

	"github.com/kaweezle/kaweezle/pkg/cluster"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

func NewStopCommand() *cobra.Command {
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the cluster and the WSL distribution",
		Long:  `Currently this stops abruptly the distribution.`,
		Run: func(cmd *cobra.Command, args []string) {
			status, err := cluster.GetClusterStatus(DistributionName)
			cobra.CheckErr(err)
			if status != cluster.Started {
				log.Fatalf("Cluster %s in bad status: %v", DistributionName, status)
				os.Exit(1)
			}

			cobra.CheckErr(cluster.StopCluster(DistributionName))
		},
	}

	return stopCmd
}

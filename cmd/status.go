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
	"fmt"
	"os"

	"github.com/kaweezle/kaweezle/pkg/cluster"
	"github.com/kaweezle/kaweezle/pkg/k8s"
	"github.com/pterm/pterm"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/runtime"
)

// NewStatusCommand creates a new status command
func NewStatusCommand() *cobra.Command {
	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Current status of the cluster",
		Long: `Gives the status of the cluster. Example:
	
	> kaweezle status
	`,
		Run: performStatus,
	}

	statusCmd.Flags().BoolVarP(&waitReadiness, "wait", "w", waitReadiness, "Wait n seconds for all pods to settle")
	return statusCmd
}

// statusCmd represents the status command

var waitReadiness = false

var callbackCount = 0

func callback(ok bool, count int, ready []*cluster.WorkloadState, unready []*cluster.WorkloadState) {
	if callbackCount == 0 {
		fmt.Printf("\n%d workloads, %d ready, %d unready\n", count, len(ready), len(unready))
		for _, state := range ready {
			fmt.Println(state.LongString())
		}
	} else {
		if len(unready) > 0 {
			fmt.Printf("\n%d unready workloads remaining:\n", len(unready))
		} else {
			fmt.Printf("\n🎉 All workloads (%d) ready:\n", count)
			for _, state := range ready {
				fmt.Println(state.LongString())
			}
		}
	}

	for _, state := range unready {
		fmt.Println(state.LongString())
	}

	if !waitReadiness {
		os.Exit(0)
	}
	callbackCount++
}

func performStatus(cmd *cobra.Command, args []string) {
	status, err := cluster.GetClusterStatus(DistributionName)
	cobra.CheckErr(err)
	log.Infof("Cluster %s is %v.", pterm.Bold.Sprint(DistributionName), pterm.Bold.Sprint(status))
	if status == cluster.Started {
		runtime.ErrorHandlers = runtime.ErrorHandlers[:0]

		var client *k8s.RESTClientGetter

		client, err = k8s.NewRESTClientForDistribution(DistributionName)
		cobra.CheckErr(err)
		cluster.WaitForWorkloads(client, 0, callback)
	}
}

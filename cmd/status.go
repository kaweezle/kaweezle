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
	log "github.com/antoinemartin/kaweezle/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/yuk7/wsllib-go"
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
	if !wsllib.WslIsDistributionRegistered(DistributionName) {
		log.Warningf("The distribution %s is not registered.", DistributionName)
	} else {
		log.Infof("The distribution %s is registered.", DistributionName)
	}
}

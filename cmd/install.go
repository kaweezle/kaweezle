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
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/yuk7/wsllib-go"
)

var (
	rootfs           string
	defaultRootFiles = []string{"install.tar", "install.tar.gz", "rootfs.tar", "rootfs.tar.gz", "install.ext4.vhdx", "install.ext4.vhdx.gz"}
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Kaweezle distribution",
	Long: `Installs the kaweezle distribution and starts the cluster.

Examples:

> kaweezle install --root rootfs.tar.gz
`,
	Run: _perform,
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().StringVarP(&rootfs, "root", "r", detectRootfsFiles(), "The root file system to install")
}

func detectRootfsFiles() string {
	efPath, _ := os.Executable()
	efDir := filepath.Dir(efPath)
	for _, rootFile := range defaultRootFiles {
		rootPath := filepath.Join(efDir, rootFile)
		_, err := os.Stat(rootPath)
		if err == nil {
			return rootPath
		}
	}
	return "rootfs.tar.gz"
}

func _perform(cmd *cobra.Command, args []string) {
	if wsllib.WslIsDistributionRegistered(DistributionName) {
		cobra.CheckErr(fmt.Sprintf("The distribution %s is already registered.", DistributionName))
	}
	_, err := os.Stat(rootfs)
	cobra.CheckErr(errors.Wrapf(err, "Bad root filesystem: %s", rootfs))

	jsonArg := ""
	if JsonLogs {
		jsonArg = " --json"
	}
	log.WithFields(log.Fields{
		"rootfs":       rootfs,
		"distrib_name": DistributionName,
	}).Info("➜ Registering distribution...")
	cobra.CheckErr(wsllib.WslRegisterDistribution(DistributionName, rootfs))

	startCommand := fmt.Sprintf("/k8wsl -v %s%s start", LogLevel, jsonArg)
	log.WithFields(log.Fields{
		"distrib_name": DistributionName,
		"command":      startCommand,
	}).Info("➜ Starting kubernetes...")

	_, err = wsllib.WslLaunchInteractive(DistributionName, startCommand, true)
	cobra.CheckErr(err)
}

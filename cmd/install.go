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
	name             string
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
	Run: perform,
}

func init() {
	rootCmd.AddCommand(installCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// installCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// installCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	installCmd.Flags().StringVarP(&rootfs, "root", "r", detectRootfsFiles(), "The root file system to install")
	installCmd.Flags().StringVarP(&name, "name", "n", "kaweezle", "The name of the WSL distribution to install")
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

func perform(cmd *cobra.Command, args []string) {
	if wsllib.WslIsDistributionRegistered(name) {
		cobra.CheckErr(fmt.Sprintf("The distribution %s is already registered.", name))
	}
	_, err := os.Stat(rootfs)
	cobra.CheckErr(errors.Wrapf(err, "Bad root filesystem: %s", rootfs))

	log.WithFields(log.Fields{
		"rootfs": rootfs,
		"name":   name,
	}).Info("Registering distribution...")
	cobra.CheckErr(wsllib.WslRegisterDistribution(name, rootfs))
}

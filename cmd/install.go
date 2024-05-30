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
	"github.com/kaweezle/kaweezle/pkg/rootfs"

	"github.com/spf13/cobra"
)

// NewInstallCommand creates a new install command
func NewInstallCommand() *cobra.Command {
	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install Kaweezle distribution",
		Long: `Installs the kaweezle distribution and starts the cluster.
	
	Examples:
	
	> kaweezle install --root rootfs.tar.gz
	`,
		Run: performStart,
	}
	installCmd.Flags().StringVarP(&rootfs.TarFilePath, "root", "r", rootfs.DefaultTarFilePath, "The root file system to install")

	return installCmd
}

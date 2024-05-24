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

	"github.com/Microsoft/go-winio"
	"github.com/kaweezle/kaweezle/pkg/cluster"
	"github.com/kaweezle/kaweezle/pkg/config"
	"google.golang.org/grpc"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var RemoveDomains bool

// startCmd represents the start command
var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure the cluster",
	Long:  `Set the configuration properties of the cluster.`,
	Run:   performConfigure,
}

// routeCmd represents the route command
var routeCmd = &cobra.Command{
	Use:   "route [ip-address]",
	Args:  cobra.MaximumNArgs(1),
	Short: "Add route to the cluster ingress IP address",
	Long:  `Add route to the cluster ingress IP address.`,
	Run:   performRoute,
}

var ageCmd = &cobra.Command{
	Use:   "age [age-key-file]",
	Args:  cobra.ExactArgs(1),
	Short: "Add age key file to the cluster",
	Long: `Add age key file to the cluster. 
This command will copy the age key file to the cluster and update the configuration file to use it.`,
	Run: performAge,
}

var sshCmd = &cobra.Command{
	Use:   "ssh [ssh-key-file]",
	Args:  cobra.ExactArgs(1),
	Short: "Add ssh key file to the cluster",
	Long: `Add ssh key file to the cluster.
This command will copy the ssh key file to the cluster.`,
	Run: performSsh,
}

var kustomizeCmd = &cobra.Command{
	Use:   "kustomize [kustomize-url]",
	Args:  cobra.ExactArgs(1),
	Short: "Add kustomize url to the cluster",
	Long: `Add kustomize url to the cluster.
This command will set the kustomize url in the configuration file.`,
	Run: performKustomize,
}

var domainsCommand = &cobra.Command{
	Use:   "domains [domain...]",
	Args:  cobra.MinimumNArgs(0),
	Short: "Bind domain names to the cluster ingress IP address",
	Long: `Bind domain names to the cluster ingress IP address.
This command will bind or remove domain names to the cluster ingress IP address.`,
	Run: performDomains,
}

var elevateCommand = &cobra.Command{
	Use:   "elevate [pipe-name]",
	Args:  cobra.ExactArgs(1),
	Short: "Create an elevated server listening on the specified pipe",
	Long: `Create a GRPC Server listening on the specified windows pipe name.
The command MUST be run from an elevated prompt.
`,
	Run: performElevate,
}

func init() {
	rootCmd.AddCommand(configureCmd)
	flags := configureCmd.Flags()

	AddConfigurationFlags(flags, ConfigurationOptions)
	bindFlags(configureCmd, viper.GetViper())
	configureCmd.AddCommand(routeCmd)
	configureCmd.AddCommand(ageCmd)
	configureCmd.AddCommand(sshCmd)
	configureCmd.AddCommand(kustomizeCmd)
	configureCmd.AddCommand(domainsCommand)
	configureCmd.AddCommand(elevateCommand)

	domainsCommand.Flags().StringVar(&ConfigurationOptions.PersistentIPAddress, "ip-address", ConfigurationOptions.PersistentIPAddress, "The persistent IP address to use for the WSL distribution")
	domainsCommand.Flags().BoolVarP(&RemoveDomains, "remove", "r", false, "Remove the domain names")
}

func performConfigure(cmd *cobra.Command, args []string) {
	status, err := cluster.GetClusterStatus(DistributionName)
	cobra.CheckErr(err)
	if status == cluster.Uninstalled {
		cobra.CheckErr(fmt.Errorf("distribution %s is not installed", DistributionName))
	}
	config.Configure(DistributionName, ConfigurationOptions)
}

func performRoute(cmd *cobra.Command, args []string) {
	if len(args) == 1 {
		ConfigurationOptions.PersistentIPAddress = args[0]
	}
	cobra.CheckErr(config.RouteToWSL(DistributionName, ConfigurationOptions.PersistentIPAddress))
}

func performAge(cmd *cobra.Command, args []string) {
	if len(args) == 1 {
		ConfigurationOptions.AgeKeyFile = args[0]
	}
	cobra.CheckErr(config.ConfigureAgeKeyFile(DistributionName, ConfigurationOptions.AgeKeyFile))
}

func performSsh(cmd *cobra.Command, args []string) {
	if len(args) == 1 {
		ConfigurationOptions.SshKeyFile = args[0]
	}
	cobra.CheckErr(config.ConfigureSshKeyFile(DistributionName, ConfigurationOptions.SshKeyFile))
}

func performKustomize(cmd *cobra.Command, args []string) {
	if len(args) == 1 {
		ConfigurationOptions.KustomizeUrl = args[0]
	}
	cobra.CheckErr(config.ConfigureKustomizeUrl(DistributionName, ConfigurationOptions.KustomizeUrl))
}

func performDomains(cmd *cobra.Command, args []string) {
	var err error
	_, err = config.ConfigureDomains(DistributionName, ConfigurationOptions.PersistentIPAddress, args, RemoveDomains)

	cobra.CheckErr(err)
}

func performElevate(cmd *cobra.Command, args []string) {
	if !config.IsAdmin() {
		cobra.CheckErr(fmt.Errorf("not running from an elevated prompt"))
	}
	pipePath := args[0]

	pc := &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;AU)",
		InputBufferSize:    512,
		OutputBufferSize:   512,
	}

	l, err := winio.ListenPipe(pipePath, pc)
	//l, err := net.Listen("tcp", ":50005")
	if err != nil {
		log.Fatal("listen error:", err)
	}
	defer l.Close()
	log.Printf("Server listening op pipe %v\n", pipePath)

	s := grpc.NewServer()
	done := make(chan bool, 1)

	go func() {
		<-done
		s.Stop()
	}()

	impl := config.ElevatedConfigurationServerImpl{Done: done}
	config.RegisterElevatedConfigurationServer(s, impl)

	log.Println("start server")
	// and start...
	if err := s.Serve(l); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}

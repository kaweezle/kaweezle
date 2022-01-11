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
	"os"

	"github.com/antoinemartin/kaweezle/pkg/logger"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"

	"github.com/spf13/viper"
)

var (
	cfgFile          string
	LogLevel         string
	LogFile          string
	DistributionName string
	JSONLogs         bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kaweezle",
	Short: "Kubernetes for WSL 2",
	Long:  `Manages a local kubernetes cluster working on WSL2.`,
	Example: `kaweezeel install
kaweezle status
kaweezle -v debug start
`,
	Version: "v0.2.1", // <---VERSION--->
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {

		if level, err := log.ParseLevel(LogLevel); err == nil {
			log.SetLevel(level)
		} else {
			log.WithError(err).Fatal("Error while parsing log level")
		}

		if LogFile != "" {
			logger.InitFileLogging(LogFile, JSONLogs)
		} else {
			if JSONLogs {
				log.SetFormatter(&log.JSONFormatter{})
			} else {
				log.SetFormatter(&logger.PTermFormatter{
					Emoji:      true,
					ShowFields: true,
				})
			}
		}

		return nil
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kaweezle.yaml)")
	rootCmd.PersistentFlags().StringVarP(&LogLevel, "verbosity", "v", log.InfoLevel.String(), "Log level (debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().StringVarP(&LogFile, "logfile", "l", "", "Log file to save")
	rootCmd.PersistentFlags().BoolVar(&JSONLogs, "json", false, "Output JSON logs")
	rootCmd.PersistentFlags().StringVarP(&DistributionName, "name", "n", "kaweezle", "The name of the WSL distribution to manage")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".kaweezle" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".kaweezle")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

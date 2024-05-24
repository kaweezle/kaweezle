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
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kaweezle/kaweezle/pkg/config"
	"github.com/kaweezle/kaweezle/pkg/logger"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/spf13/viper"
)

const commandName = "kaweezle"

var (
	cfgFile          string
	LogLevel         string
	LogFile          string
	DistributionName string
	JSONLogs         bool
	envPrefix        = strings.ToUpper(commandName)
	afs              = &afero.Afero{Fs: afero.NewOsFs()}
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   commandName,
	Short: "Kubernetes for WSL 2",
	Long:  `Manages a local kubernetes cluster working on WSL2.`,
	Example: `kaweezle install
kaweezle status
kaweezle -v debug start
`,
	Version: "v0.3.14", // <---VERSION--->
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		config.ReleaseElevatedClient(context.TODO())
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig, initLogging)

	flags := rootCmd.PersistentFlags()

	flags.StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kaweezle.yaml)")
	flags.StringVarP(&LogLevel, "verbosity", "v", log.InfoLevel.String(), "Log level (debug, info, warn, error, fatal, panic)")
	flags.StringVarP(&LogFile, "logfile", "l", "", "Log file to save")
	flags.BoolVar(&JSONLogs, "json", false, "Output JSON logs")
	flags.StringVarP(&DistributionName, "name", "n", "kaweezle", "The name of the WSL distribution to manage")
}

// initLogging initializes logging
func initLogging() {
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
		viper.SetConfigName("." + commandName)
	}

	viper.AutomaticEnv() // read in environment variables that match
	viper.SetEnvPrefix("kaweezle")

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.WithField("config_file", viper.ConfigFileUsed()).Infof("Using config file: %s", viper.ConfigFileUsed())
	}
	bindFlags(rootCmd, viper.GetViper())
}

// Bind each cobra flag to its associated viper configuration (config file and environment variable)
func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		// Environment variables can't have dashes in them, so bind them to their equivalent
		// keys with underscores, e.g. --favorite-color to STING_FAVORITE_COLOR
		v.BindPFlag(strings.ReplaceAll(f.Name, "-", "_"), f)

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(f.Name) {
			val := v.Get(f.Name)
			cmd.PersistentFlags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})

	// Same with the command flags
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		v.BindPFlag(strings.ReplaceAll(f.Name, "-", "_"), f)

		if !f.Changed && v.IsSet(f.Name) {
			val := v.Get(f.Name)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}

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
	"path/filepath"
	"strings"

	"github.com/kaweezle/kaweezle/pkg/config"
	"github.com/kaweezle/kaweezle/pkg/logger"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/spf13/viper"
)

var (
	cfgFile          string
	LogLevel         string
	LogFile          string
	DistributionName string
	JSONLogs         bool
	commandName      = "kaweezle"
)

func NewKaweezleCommand() *cobra.Command {
	initConfig()
	// TODO: try pre-initializing the logger with the default values
	cobra.OnInitialize(initLogging)

	// rootCmd represents the base command when called without any subcommands
	rootCmd := &cobra.Command{
		Use:   commandName,
		Short: "Kubernetes for WSL 2",
		Long:  `Manages a local kubernetes cluster working on WSL2.`,
		Example: `kaweezle install
kaweezle status
kaweezle -v debug start
`,
		Version: "v0.3.17", // <---VERSION--->
		// Uncomment the following line if your bare application
		// has an action associated with it:
		// Run: func(cmd *cobra.Command, args []string) { },
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			config.ReleaseElevatedClient(context.TODO())
		},
	}

	// This is for automatic binding of flags
	rootCmd.SetGlobalNormalizationFunc(func(f *pflag.FlagSet, name string) pflag.NormalizedName {
		name = strings.ReplaceAll(name, "-", "_")
		return pflag.NormalizedName(name)
	})
	initializeRootFlags(rootCmd.PersistentFlags())

	rootCmd.AddCommand(NewInstallCommand())
	rootCmd.AddCommand(NewStartCommand())
	rootCmd.AddCommand(NewStopCommand())
	rootCmd.AddCommand(NewStatusCommand())
	rootCmd.AddCommand(NewConfigureCommand())
	rootCmd.AddCommand(NewUninstallCommand())
	rootCmd.AddCommand(NewVersionCommand())
	rootCmd.AddCommand(NewUpdateCommand())

	bindFlags(rootCmd, viper.GetViper())

	return rootCmd
}

func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Long:  `Print the version number of kaweezle`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(cmd.Root().Version)
		},
	}
}

func initializeRootFlags(flags *pflag.FlagSet) {

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
	commandName, _ = os.Executable()
	commandName = strings.TrimSuffix(filepath.Base(commandName), filepath.Ext(commandName))
	envPrefix := strings.ToUpper(commandName)
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".kaweezle" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath(filepath.Join(os.Getenv("APPDATA"), commandName))
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("." + commandName)
	}

	viper.AutomaticEnv() // read in environment variables that match
	viper.SetEnvPrefix(envPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// If a config file is found, read it in.
	viper.ReadInConfig()
}

func bindFlag(f *pflag.Flag, v *viper.Viper) {
	// Environment variables can't have dashes in them, so bind them to their equivalent
	// keys with underscores, e.g. --favorite-color to STING_FAVORITE_COLOR
	viperName := strings.ReplaceAll(f.Name, "-", "_")
	v.BindPFlag(viperName, f)

	// Apply the viper config value to the flag when the flag is not set and viper has a value
	if !f.Changed && v.IsSet(viperName) {
		val := v.Get(viperName)

		if vi, ok := f.Value.(pflag.SliceValue); ok {
			stringValues, _ := lo.FromAnySlice[string](val.([]interface{}))
			vi.Replace(stringValues)
		} else {
			f.Value.Set(fmt.Sprintf("%v", val))
		}
	}
}

// Bind each cobra flag to its associated viper configuration (config file and environment variable)
func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	persistent := cmd.PersistentFlags()
	persistent.VisitAll(func(f *pflag.Flag) {
		bindFlag(f, v)
	})

	flags := cmd.Flags()
	// Same with the command flags
	flags.VisitAll(func(f *pflag.Flag) {
		bindFlag(f, v)
	})
	for _, c := range cmd.Commands() {
		bindFlags(c, v)
	}
}

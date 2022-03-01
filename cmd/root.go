// Copyright © 2020 Kindly Ops, LLC <support@kindlyops.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/muesli/coral"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var cfgFile string

// Debug controls whether or not to enable debug level logging.
var Debug bool

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &coral.Command{
	Version: "dev",
	Use:     "vbs",
	Short:   "video broadcasting stuff",
	Long: `vbs helps work with video broadcast files and streams.
This tool depends on ffmpeg and ffprobe, which must be installed
separately.
Brought to you by

_  ___           _ _        ___
| |/ (_)_ __   __| | |_   _ / _ \ _ __  ___
| ' /| | '_ \ / _| | | | | | | | | '_ \/ __|
| . \| | | | | (_| | | |_| | |_| | |_) __ \
|_|\_\_|_| |_|\__,_|_|\__, |\___/| .__/|___/
                      |___/      |_|
use at your own risk.
`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

var configCmd = &coral.Command{
	Use:   "save-config",
	Short: "Save the current config.",
	Long:  `Save the current config after merge from files, arguments, and environment.`,
	Run:   saveConfig,
	Args:  coral.NoArgs,
}

func saveConfig(cmd *coral.Command, args []string) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Couldn't locate config dir")
	}

	configDir = filepath.Join(configDir, "vbs")

	os.MkdirAll(configDir, os.ModePerm)

	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Couldn't create config dir")
	}

	err = viper.SafeWriteConfig()

	if err != nil {
		log.Fatal().Err(err).Msgf("Error writing out config to %s.", viper.ConfigFileUsed())
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(v string) {
	rootCmd.SetVersionTemplate(v)

	if err := rootCmd.Execute(); err != nil {
		log.Error().Err(err).Msg("error running root command")
		os.Exit(1)
	}
}

func init() {
	zerolog.TimeFieldFormat = time.RFC3339

	if isatty.IsTerminal(os.Stdout.Fd()) {
		output := zerolog.ConsoleWriter{Out: os.Stderr}
		log.Logger = log.With().Caller().Logger().Output(output)
	} else {
		log.Logger = log.With().Caller().Logger()
	}

	coral.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "override config file")
	viper.BindPFlag("config_file", rootCmd.PersistentFlags().Lookup("config"))

	rootCmd.PersistentFlags().BoolVarP(&Debug, "debug", "d", false, "Print debug messages while working")

	rootCmd.AddCommand(configCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetConfigType("yaml")

	configFile := viper.GetString("config_file")

	if configFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(configFile)
	} else {
		// Find platform appropriate config directory.
		configDir, err := os.UserConfigDir()
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("Couldn't locate config dir")
		}

		configDir = filepath.Join(configDir, "vbs")

		// Search config in home directory with name ".vbs" (without extension).
		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Info().Msg(fmt.Sprintf("Using config file: %s", viper.ConfigFileUsed()))
	}

	if Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

// Copyright © 2022 Kindly Ops, LLC <support@kindlyops.com>
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
	"os"
	"path/filepath"

	"github.com/muesli/coral"
	"github.com/pocketbase/pocketbase"
	"github.com/rs/zerolog/log"
)

var flyServerCmd = &coral.Command{
	Use:   "serve",
	Short: "Run pocketbase http server",
	Long:  `Serve pocketbase web requests.`,
	Run:   flyServer,
	Args:  coral.ExactArgs(0),
}

func flyServer(cmd *coral.Command, args []string) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Couldn't locate config dir")
	}

	configDir = filepath.Join(configDir, "vbs")

	os.MkdirAll(configDir, os.ModePerm)

	log.Debug().Msgf("running pocketbase with data dir %s\n", configDir)
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: configDir,
	})

	if err := app.Start(); err != nil {
		log.Fatal().Err(err).Msg("error starting pocketbase")
	}

	// app.RootCmd.AddCommand(&cobra.Command{
	// 	Use: "fly",
	// 	Run: func(command *cobra.Command, args []string) {
	// 		log.Debug().Msgf("Pocketbase interceptor no-op")
	// 		if err := app.Execute(); err != nil {
	// 			log.Fatal().Err(err).Msg("error starting pocketbase")
	// 		}
	// 	},
	// })
}

// Port to listen for HTTP requests.
var httpTarget string

func init() {
	flyServerCmd.Flags().StringVarP(&httpTarget, "http", "a", "0.0.0.0:8080", "Address & port to listen for HTTP requests")
	//viper.BindPFlag("port", flyServerCmd.Flags().Lookup("port"))
	rootCmd.AddCommand(flyServerCmd)
}

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
	"github.com/muesli/coral"
	"github.com/pocketbase/pocketbase"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var flyServerCmd = &coral.Command{
	Use:   "serve",
	Short: "Run pocketbase http server",
	Long:  `Serve pocketbase web requests.`,
	Run:   flyServer,
	Args:  coral.ExactArgs(0),
}

func flyServer(cmd *coral.Command, args []string) {
	_ = "127.0.0.1:" + viper.GetString("port")

	log.Debug().Msgf("running pocketbase\n")
	app := pocketbase.New()
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
var FlyPort string

func init() {
	flyServerCmd.Flags().StringVarP(&FlyPort, "port", "p", "8080", "Port to listen for HTTP requests")
	viper.BindPFlag("port", flyServerCmd.Flags().Lookup("port"))
	rootCmd.AddCommand(flyServerCmd)
}
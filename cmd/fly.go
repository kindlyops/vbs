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
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var flyServerCmd = &coral.Command{
	Use:   "fly",
	Short: "Run http server for fly.io",
	Long:  `Serve web requests in fly.io.`,
	Run:   flyServer,
	Args:  coral.ExactArgs(0),
}

func flyServer(cmd *coral.Command, args []string) {
	addr := "127.0.0.1:" + viper.GetString("port")

	log.Debug().Msgf("Listening on port: '%s'\n", addr)
	//app := pocketbase.New()

	// if err := app.Start(); err != nil {
	// 	log.Fatal(err)
	// }

}

// Port to listen for HTTP requests.
var FlyPort string

func init() {
	flyServerCmd.Flags().StringVarP(&FlyPort, "port", "p", "8080", "Port to listen for HTTP requests")
	viper.BindPFlag("port", flyServerCmd.Flags().Lookup("port"))
	rootCmd.AddCommand(flyServerCmd)
}

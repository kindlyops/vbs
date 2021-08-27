// Copyright © 2021 Kindly Ops, LLC <support@kindlyops.com>
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
	"io/fs"

	"github.com/kindlyops/vbs/embeddy"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"net/http"
)

var lightingBridgeCmd = &cobra.Command{
	Use:   "lighting-bridge <companion-ip>",
	Short: "Serve embedded lighting control page",
	Long:  `Use OSC to send messages to Companion API for lighting control.`,
	Run:   lightingBridge,
	Args:  cobra.ExactArgs(1),
}

func lightingBridge(cmd *cobra.Command, args []string) {
	//companionAddr := args[0] // TODO: connect to companion
	serverAddr := "127.0.0.1:" + ServerPort
	dist, _ := fs.Sub(embeddy.GetNextFS(), "dist")

	fs.WalkDir(dist, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		log.Debug().Msgf("path=%q, isDir=%v\n", path, d.IsDir())
		return nil
	})

	http.Handle("/", http.FileServer(http.FS(dist)))
	// The API will be served under `/api`.
	http.HandleFunc("/api/on", handleON)
	http.HandleFunc("/api/off", handleOff)

	// Start HTTP server at :8080.
	log.Debug().Msgf("Starting HTTP server at: http://%s\n", serverAddr)
	err := http.ListenAndServe(serverAddr, nil)
	if err != nil {
		log.Error().Err(err).Msg("error from ListenAndServe")
	}
}

func handleON(w http.ResponseWriter, r *http.Request) {
	// TODO: only accept POST
	log.Debug().Msg("handleON")
	sendAPIResponse(w, r)
}

func handleOff(w http.ResponseWriter, r *http.Request) {
	// TODO: only accept POST
	log.Debug().Msg("handleOff")
	sendAPIResponse(w, r)
}

func sendAPIResponse(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

// Port to listen for HTTP.
var ServerPort string

func init() {
	lightingBridgeCmd.Flags().StringVarP(&ServerPort, "port", "p", "7007", "Port to serve website on")
	rootCmd.AddCommand(lightingBridgeCmd)
}

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

	"github.com/hypebeast/go-osc/osc"
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
	Args:  cobra.NoArgs,
}

func lightingBridge(cmd *cobra.Command, args []string) {
	listenAddr := "127.0.0.1:" + ServerPort
	dist, _ := fs.Sub(embeddy.GetNextFS(), "dist")

	fs.WalkDir(dist, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		log.Debug().Msgf("path=%q, isDir=%v\n", path, d.IsDir())
		return nil
	})

	// TODO: maybe switch to echo
	// https://echo.labstack.com/middleware/static/
	// https://echo.labstack.com/middleware/csrf/
	// https://echo.labstack.com/middleware/rate-limiter/
	// https://echo.labstack.com/middleware/secure/
	http.Handle("/", http.FileServer(http.FS(dist)))
	// The API will be served under `/api`.
	http.HandleFunc("/api/on", handleON)
	http.HandleFunc("/api/off", handleOff)
	http.HandleFunc("/api/ftb", handleFTB)
	http.HandleFunc("/api/dsk", handleDSK)

	// Start HTTP server at :8080.
	log.Debug().Msgf("Starting HTTP server at: http://%s\n", listenAddr)
	err := http.ListenAndServe(listenAddr, nil)
	if err != nil {
		log.Error().Err(err).Msg("error from ListenAndServe")
	}
}

func sendOSC(path string) {
	client := osc.NewClient(CompanionAddr, 12321) // TODO configurable port
	msg := osc.NewMessage(path)
	client.Send(msg)
}

func handleON(w http.ResponseWriter, r *http.Request) {
	// TODO: only accept POST
	log.Debug().Msg("handleON")
	// page 20, 2nd button
	sendOSC("/press/bank/20/2")
	sendAPIResponse(w, r)
}

func handleOff(w http.ResponseWriter, r *http.Request) {
	// TODO: only accept POST
	log.Debug().Msg("handleOff")
	// page 20, 3rd button
	sendOSC("/press/bank/20/3")
	sendAPIResponse(w, r)
}

func handleFTB(w http.ResponseWriter, r *http.Request) {
	// TODO: only accept POST
	log.Debug().Msg("handleFTB")
	// page 20, 4th button
	sendOSC("/press/bank/20/4")
	sendAPIResponse(w, r)
}

func handleDSK(w http.ResponseWriter, r *http.Request) {
	// TODO: only accept POST
	log.Debug().Msg("handleDSK")
	// page 20, 5th button
	sendOSC("/press/bank/20/5")
	sendAPIResponse(w, r)
}

func sendAPIResponse(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

// Port to listen for HTTP.
var ServerPort string
var CompanionAddr string

func init() {
	lightingBridgeCmd.Flags().StringVarP(&ServerPort, "port", "p", "7007", "Port to serve website on")
	lightingBridgeCmd.Flags().StringVarP(&CompanionAddr, "companion", "c", "127.0.0.1", "Address to send companion OSC commands")
	rootCmd.AddCommand(lightingBridgeCmd)
}

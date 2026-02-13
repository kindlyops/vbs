// Copyright © 2018 Kindly Ops, LLC <support@kindlyops.com>
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

//go:build windows
// +build windows

package cmd

import (
	"fmt"
	"net"
	"os"

	"github.com/Microsoft/go-winio"
	"github.com/rs/zerolog/log"
)

func GetIPCName() string {
	f, err := os.CreateTemp("", "vbs-player")
	if err != nil {
		log.Fatal().Err(err).Msg("Could not create temp file")
	}

	socketName := f.Name()
	f.Close()
	os.Remove(socketName)
	pipeName := fmt.Sprintf(`\\.\pipe\%s`, socketName)

	return pipeName
}

func ConnectIPC(socketName string) (net.Conn, error) {
	c, err := winio.DialPipe(socketName, nil)

	return c, err
}

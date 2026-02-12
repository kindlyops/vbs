// Copyright © 2025 Kindly Ops, LLC <support@kindlyops.com>
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
	"encoding/json"
	"net"
	"testing"
	"time"
)

func TestReadMpvEvents_GracefulExitOnClose(t *testing.T) {
	server, client := net.Pipe()
	sub := make(chan responseMsg, 10)
	done := make(chan struct{})

	go func() {
		readMpvEvents(client, sub)
		close(done)
	}()

	// Send one event so we know the goroutine is running.
	_, err := server.Write([]byte(`{"event":"property-change"}` + "\n"))
	if err != nil {
		t.Fatalf("failed to write to pipe: %v", err)
	}

	msg := <-sub
	if msg.event != `{"event":"property-change"}` {
		t.Errorf("expected event %q, got %q", `{"event":"property-change"}`, msg.event)
	}

	// Close the server side; the reader should exit gracefully.
	server.Close()

	select {
	case <-done:
		// goroutine exited — success
	case <-time.After(2 * time.Second):
		t.Fatal("readMpvEvents did not exit after connection was closed")
	}
}

func TestReadMpvEvents_FirstLineOnly(t *testing.T) {
	server, client := net.Pipe()
	sub := make(chan responseMsg, 10)

	go readMpvEvents(client, sub)
	defer server.Close()
	defer client.Close()

	// Send a multi-line payload in a single write.
	_, err := server.Write([]byte("first-line\nsecond-line\n"))
	if err != nil {
		t.Fatalf("failed to write to pipe: %v", err)
	}

	msg := <-sub
	if msg.event != "first-line" {
		t.Errorf("expected first line only, got %q", msg.event)
	}
}

func TestMpvIPCCommand_Marshal(t *testing.T) {
	tests := []struct {
		name string
		cmd  mpvIPCCommand
		want string
	}{
		{
			name: "simple command",
			cmd:  mpvIPCCommand{Command: []interface{}{"quit"}},
			want: `{"command":["quit"]}`,
		},
		{
			name: "command with int arg",
			cmd:  mpvIPCCommand{Command: []interface{}{"seek", 5}},
			want: `{"command":["seek",5]}`,
		},
		{
			name: "command with string args",
			cmd:  mpvIPCCommand{Command: []interface{}{"set_property", "fullscreen", "yes"}},
			want: `{"command":["set_property","fullscreen","yes"]}`,
		},
		{
			name: "command with bool arg",
			cmd:  mpvIPCCommand{Command: []interface{}{"set_property", "pause", true}},
			want: `{"command":["set_property","pause",true]}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.cmd)
			if err != nil {
				t.Fatalf("unexpected marshal error: %v", err)
			}

			if string(data) != tc.want {
				t.Errorf("got %s, want %s", string(data), tc.want)
			}
		})
	}
}

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

package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var playCmd = &cobra.Command{
	Use:   "play <videofile.mp4>",
	Short: "Play a videofile fullscreen with mpv player.",
	Long:  `Use mpv video player to play a video file fullscreen on the designated display.`,
	Run:   play,
	Args:  cobra.ExactArgs(1),
}

type model struct {
	currentItem   string
	remainingTime float32
	outputScreen  int8
	fullScreen    bool
	playing       bool
	debug         string
	controlSocket net.Conn
}

func initialModel(file string) model {
	return model{
		currentItem:   file,
		fullScreen:    false,
		remainingTime: 0,
		outputScreen:  0,
		playing:       false,
		debug:         "",
		controlSocket: nil,
	}
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

type vbsQuit int

func responseReader(c net.Conn) {
	bufferSize := 2048
	response := make([]byte, bufferSize)
	count, err := c.Read(response)

	if err != nil {
		log.Fatal().Err(err).Msg("Could not read response")
	}

	log.Debug().Msgf("Response: %s", response[:count])
}

func cmdQuitMpv(m model) tea.Cmd {
	return func() tea.Msg {
		_, err := m.controlSocket.Write([]byte("{ \"command\": [\"quit\"] }\n"))

		if err != nil {
			log.Fatal().Err(err).Msg("Could not write quit command")
		}

		return vbsQuit(0)
	}
}

func cmdPlayMpv(m model) tea.Cmd {
	return func() tea.Msg {
		cmd := fmt.Sprintf("{ \"command\": [\"set_property\", \"pause\", %v] }\n", m.playing)
		_, err := m.controlSocket.Write([]byte(cmd))

		if err != nil {
			log.Fatal().Err(err).Msg("Could not send pause state command")
		}

		return nil
	}
}

func cmdFullscreenMpv(m model) tea.Cmd {
	return func() tea.Msg {
		fullScreen := "no"
		if m.fullScreen {
			fullScreen = "yes"
		}

		cmd := fmt.Sprintf("{ \"command\": [\"set_property\", \"fullscreen\", \"%v\"] }\n", fullScreen)
		_, err := m.controlSocket.Write([]byte(cmd))

		if err != nil {
			log.Fatal().Err(err).Msg("Could not send fullscreen state command")
		}

		return nil
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case vbsQuit:
		// we finished our cleanup, exit bubbletea
		return m, tea.Quit
	// Is it a key press?
	case tea.KeyMsg:
		// Cool, what was the actual key pressed?
		switch msg.String() {
		// These keys should exit the program.
		case "ctrl+c", "q":
			// do our internal cleanup first
			return m, cmdQuitMpv(m)

		// The "enter" key and the spacebar (a literal space) toggle
		// the playing state
		case "enter", " ":
			m.playing = !m.playing

			return m, cmdPlayMpv(m)

		case "f", "F":
			m.fullScreen = !m.fullScreen

			return m, cmdFullscreenMpv(m)
		}
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func (m model) View() string {
	// The header
	s := "VBS player\n\n"

	// Render the row
	s += fmt.Sprintf("Current item: %v\n", m.currentItem)
	s += fmt.Sprintf("Remaining time: %v\n", m.remainingTime)
	s += fmt.Sprintf("Output screen: %v\n", m.outputScreen)
	s += fmt.Sprintf("Fullscreen: %v\n", m.fullScreen)
	s += fmt.Sprintf("Playing: %v\n\n\n", m.playing)
	s += fmt.Sprintf("Debug: %v\n", m.debug)

	// The footer
	s += "\nPress q to quit. Press <space> to play. f to fullscreen\n"

	// Send the UI for rendering
	return s
}

func play(cmd *cobra.Command, args []string) {
	_, err := exec.LookPath("mpv")

	if err != nil {
		log.Fatal().Err(err).Msg("Could not find mpv. Please install mpv player.")
	}

	target, _ := filepath.Abs(args[0])
	_, err = os.Stat(target)

	if err != nil {
		log.Fatal().Err(err).Msgf("Could not access video %s", target)
	}

	m := initialModel(target)

	f, err := ioutil.TempFile("", "vbs-player")
	if err != nil {
		log.Fatal().Err(err).Msg("Could not create temp file")
	}

	socketName := f.Name()

	os.Remove(socketName)
	defer os.Remove(socketName)

	m.debug = socketName

	go runMpvPlayer(m.outputScreen, socketName, m.currentItem)

	time.Sleep(1 * time.Second)

	c, err := net.Dial("unix", socketName)
	m.controlSocket = c

	// TODO: set up an event loop and turn them into messages
	// go responseEventReader(m.controlSocket)

	if err != nil {
		log.Fatal().Err(err).Msg("Could not open control socket")
	}

	p := tea.NewProgram(m)
	if err := p.Start(); err != nil {
		log.Fatal().Err(err).Msg("BUBBLETEA BROKED")
	}
}

func runMpvPlayer(outputScreen int8, controlSocket string, currentItem string) {
	whichScreen := fmt.Sprintf("--fs-screen=%v", outputScreen)
	ipcArgument := fmt.Sprintf("--input-ipc-server=%v", controlSocket)
	mpv := exec.Command("mpv",
		"--pause",
		"--keep-open=always",
		"--keepaspect-window=no",
		whichScreen,
		"--autofit=70%",
		"--no-osc",
		"--no-osd-bar",
		"--osd-on-seek=no",
		"--profile=low-latency",
		"--no-terminal",
		"--no-input-terminal",
		"--no-input-builtin-bindings",
		"--input-media-keys=no",
		"--no-input-cursor",
		"--cursor-autohide=always",
		"--ontop",
		"--no-focus-on-open",
		"--no-border",
		"--image-display-duration=5",
		"--no-resume-playback",
		"--force-window=yes",
		"--idle=yes",
		ipcArgument,
		currentItem,
	)

	stdout, err := mpv.StdoutPipe()
	if err != nil {
		log.Fatal().Err(err).Msg("Could not get stdout pipe")
	}

	stderr, err := mpv.StderrPipe()
	if err != nil {
		log.Fatal().Err(err).Msg("Could not get stdout pipe")
	}

	_, err = mpv.StdinPipe()
	if err != nil {
		log.Fatal().Err(err).Msg("Could not get stdin pipe")
	}

	if err = mpv.Start(); err != nil {
		log.Fatal().Err(err).Msg("Could not start mpv")
	}

	errSlurp, _ := io.ReadAll(stderr)
	outSlurp, _ := io.ReadAll(stdout)

	if err := mpv.Wait(); err != nil {
		message := fmt.Sprintf("PROBLEMS\n\nstdout: %s\nstderr: %s\n\n", outSlurp, errSlurp)
		log.Fatal().Err(err).Msg(message)
	}
}

func init() {
	rootCmd.AddCommand(playCmd)
}

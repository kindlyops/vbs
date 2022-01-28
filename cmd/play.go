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
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// keyMap defines a set of keybindings. To work for help it must satisfy
// key.Map. It could also very easily be a map[string]key.Binding.
type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
	Help       key.Binding
	Quit       key.Binding
	Fullscreen key.Binding
	Play       key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right}, // first column
		{k.Help, k.Quit},                // second column
		{k.Play, k.Fullscreen},          // third column
	}
}

var keys = keyMap{
	Play: key.NewBinding(
		key.WithKeys(" ", "space", "enter"),
		key.WithHelp("space/enter/p", "play/pause"),
	),
	Fullscreen: key.NewBinding(
		key.WithKeys("F", "f"),
		key.WithHelp("f", "fullscreen"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "move left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "move right"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

type model struct {
	currentItem   string
	remainingTime float32
	outputScreen  int8
	fullScreen    bool
	playing       bool
	debug         string
	controlSocket net.Conn
	sub           chan responseMsg // event channel
	responses     int              // how many responses we've received
	spinner       spinner.Model
	keys          keyMap
	help          help.Model
	inputStyle    lipgloss.Style
	quitting      bool
	percent       float64
	progress      progress.Model
}

func initialModel(file string) model {
	m := model{
		currentItem:   file,
		fullScreen:    false,
		remainingTime: 0,
		outputScreen:  0,
		playing:       false,
		debug:         "",
		controlSocket: nil,
		sub:           make(chan responseMsg),
		responses:     0,
		spinner:       spinner.New(),
		keys:          keys,
		help:          help.New(),
		inputStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("#FF75B7")),
		quitting:      false,
		percent:       0,
		progress:      progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C")),
	}
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	m.spinner.Spinner = spinner.Monkey

	return m
}

var (
	appStyle   = lipgloss.NewStyle().Padding(1, 2)
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		// b.Right = "├"

		return lipgloss.NewStyle().BorderStyle(b).Padding(1, 2).
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065"))
	}

	statusMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"}).
				Render
)

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		spinner.Tick,
		listenForMpvEvents(m.sub, m.controlSocket), // loop read on mpv socket
		waitForActivity(m.sub),                     // wait for mpv event to be sent on channel
	)
}

type vbsQuit int

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
		cmd := fmt.Sprintf("{ \"command\": [\"set_property\", \"pause\", %t] }\n", m.playing)
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
	case tea.WindowSizeMsg:
		// If we set a width on the help menu it can it can gracefully truncate
		// its view as needed.
		m.help.Width = msg.Width
		padding := 2
		m.progress.Width = msg.Width - padding*2 - 2*padding
		maxWidth := 80

		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}
	case vbsQuit:
		// we finished our cleanup, exit bubbletea
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)

		return m, cmd
	case responseMsg:
		m.responses++ // record external activity
		m.debug = msg.event
		full := 100
		// simulate progress bar activity by counting events until I can
		// implement property observation for playback remaining
		m.percent = float64((m.responses+full)%full) * float64(.01)

		return m, waitForActivity(m.sub) // wait for next event
	// Is it a key press?
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Play):
			m.playing = !m.playing

			return m, cmdPlayMpv(m)
		case key.Matches(msg, m.keys.Fullscreen):
			m.fullScreen = !m.fullScreen

			return m, cmdFullscreenMpv(m)
		case key.Matches(msg, m.keys.Up):
		case key.Matches(msg, m.keys.Down):
		case key.Matches(msg, m.keys.Left):
		case key.Matches(msg, m.keys.Right):
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true

			return m, cmdQuitMpv(m)
		}
	}
	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func (m model) View() string {
	// The header
	s := titleStyle().Render("VBS player")

	// Render the row
	s += fmt.Sprintf("\nCurrent item: %v\n", m.currentItem)
	s += fmt.Sprintf("Remaining time: %v\n", m.remainingTime)
	s += fmt.Sprintf("Output screen: %v\n", m.outputScreen)
	s += fmt.Sprintf("Fullscreen: %v\n", m.fullScreen)
	s += fmt.Sprintf("Playing: %v\n\n\n", m.playing)
	s += fmt.Sprintf("Debug: '%v'\n", m.debug)
	s += "-------------------------------------------------------\n"
	s += fmt.Sprintf("\n %s Events received: %d\n\n", m.spinner.View(), m.responses)
	s += "\n  " + m.progress.ViewAs(m.percent) + "\n\n"
	helpView := m.help.View(m.keys)
	s += fmt.Sprintf("%s\n", helpView)

	// The footer
	//s += statusMessageStyle("\nPress q to quit. Press ? for help on commands\n")

	// Send the UI for rendering
	return appStyle.Render(s)
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

	ipcName := GetIPCName()
	defer os.Remove(ipcName)

	m.debug = ipcName

	go runMpvPlayer(m.outputScreen, ipcName, m.currentItem)

	time.Sleep(1 * time.Second)

	c, err := ConnectIPC(ipcName)
	m.controlSocket = c

	if err != nil {
		log.Fatal().Err(err).Msg("Could not open control socket")
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if err := p.Start(); err != nil {
		log.Fatal().Err(err).Msg("BUBBLETEA BROKED")
	}
}

func listenForMpvEvents(sub chan responseMsg, c net.Conn) tea.Cmd {
	return func() tea.Msg {
		for {
			bufferSize := 4096
			response := make([]byte, bufferSize)
			count, err := c.Read(response)

			if err != nil {
				log.Error().Err(err).Msg("Could not read response")
			}

			//log.Debug().Msgf("Got %v byte response: %s", count, response[:count])
			responseEvent := string(response[:count])
			sub <- responseMsg{
				event: strings.Split(responseEvent, "\n")[0],
			}
		}
	}
}

type responseMsg struct {
	event string
}

// A command that waits for mpv responses.
func waitForActivity(sub chan responseMsg) tea.Cmd {
	return func() tea.Msg {
		return responseMsg(<-sub)
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

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
	"container/list"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/coral"
	"github.com/rs/zerolog/log"
)

var playCmd = &coral.Command{
	Use:   "play <videofile.mp4>",
	Short: "Play a videofile fullscreen with mpv player.",
	Long:  `Use mpv video player to play a video file fullscreen on the designated display.`,
	Run:   play,
	Args:  coral.ExactArgs(1),
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
	Debug      key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit, k.Debug}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right}, // first column
		{k.Help, k.Quit, k.Debug},       // second column
		{k.Play, k.Fullscreen},          // third column
	}
}

var keys = keyMap{
	Play: key.NewBinding(
		key.WithKeys(" ", "space", "enter"),
		key.WithHelp("space/enter/p", "play/pause"),
	),
	Debug: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "toggle debug view"),
	),
	Fullscreen: key.NewBinding(
		key.WithKeys("F", "f"),
		key.WithHelp("f", "fullscreen"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "forward 5 seconds"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "back 5 seconds"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "back one frame"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "forward one frame"),
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
	terminalWidth  int
	terminalHeight int
	currentItem    string
	remainingTime  float64
	outputScreen   int8
	fullScreen     bool
	playing        bool
	debug          *list.List
	showDebug      bool
	controlSocket  net.Conn
	sub            chan responseMsg // event channel
	responses      int              // how many responses we've received
	spinner        spinner.Model
	keys           keyMap
	help           help.Model
	inputStyle     lipgloss.Style
	quitting       bool
	percent        float64
	progress       progress.Model
	ipcName        string
}

func initialModel(file string) model {
	m := model{
		currentItem:    file,
		terminalWidth:  0,
		terminalHeight: 0,
		fullScreen:     false,
		remainingTime:  0,
		outputScreen:   0,
		playing:        false,
		debug:          list.New(),
		showDebug:      false,
		controlSocket:  nil,
		sub:            make(chan responseMsg),
		responses:      0,
		spinner:        spinner.New(),
		keys:           keys,
		help:           help.New(),
		inputStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("#FF75B7")),
		quitting:       false,
		percent:        0,
		ipcName:        "",
		progress:       progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C")),
	}
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	m.spinner.Spinner = spinner.Monkey

	// put debugSize empty messages in the debug queue so it fills up the widget
	// and doesn't resize later
	for i := 0; i < debugSize; i++ {
		m.debug.PushBack(" ")
	}

	return m
}

var debugSize = 20

func pushDebugList(m *model, msg string) {
	if m.showDebug {
		m.debug.PushBack(msg)

		if m.debug.Len() > debugSize {
			m.debug.Remove(m.debug.Front())
		}
	}
}

var (
	appStyle = lipgloss.NewStyle().
		// Foreground(lipgloss.Color("226")).
		// Background(lipgloss.Color("63")).
		PaddingTop(2).
		PaddingLeft(4)

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
		cmdInitializeControlSocket(&m),
		waitForActivity(m.sub), // wait for mpv event to be sent on channel
	)
}

type vbsQuit int

func writeMpvSocket(m model, msg string) {
	_, err := m.controlSocket.Write([]byte(msg))

	if err != nil {
		log.Fatal().Err(err).Msgf("Could not send %s command", msg)
	}
}

func genericMpvCommand(m model, command string, id int, msg tea.Msg, args ...string) tea.Cmd {
	return func() tea.Msg {
		cmdArgs := []interface{}{command}
		if id != 0 {
			cmdArgs = append(cmdArgs, id)
		}

		for _, arg := range args {
			cmdArgs = append(cmdArgs, arg)
		}

		data, err := json.Marshal(mpvIPCCommand{Command: cmdArgs})
		if err != nil {
			log.Error().Err(err).Msg("Could not marshal mpv command")
			return msg
		}

		cmdStr := string(data) + "\n"
		pushDebugList(&m, cmdStr)
		writeMpvSocket(m, cmdStr)

		return msg
	}
}

func cmdQuitMpv(m model) tea.Cmd {
	return genericMpvCommand(m, "quit", 0, vbsQuit(0))
}

func cmdObserveTimeRemainingMpv(m model) tea.Cmd {
	return genericMpvCommand(m, "observe_property_string", 1, nil, "percent-pos")
}

func cmdBackOneFrameMpv(m model) tea.Cmd {
	if m.playing {
		// discard frame step during playback
		return nil
	}

	return genericMpvCommand(m, "frame-back-step", 0, nil)
}

func cmdForwardOneFrameMpv(m model) tea.Cmd {
	if m.playing {
		// discard frame step during playback
		return nil
	}

	return genericMpvCommand(m, "frame-step", 0, nil)
}

func cmdForwardFiveSecondsMpv(m model) tea.Cmd {
	return genericMpvCommand(m, "seek", 5, nil)
}

func cmdBackwardFiveSecondsMpv(m model) tea.Cmd {
	return genericMpvCommand(m, "seek", -5, nil)
}

func cmdPlayMpv(m model) tea.Cmd {
	return func() tea.Msg {
		data, err := json.Marshal(mpvIPCCommand{
			Command: []interface{}{"set_property", "pause", m.playing},
		})
		if err != nil {
			log.Error().Err(err).Msg("Could not marshal play command")
			return nil
		}

		writeMpvSocket(m, string(data)+"\n")

		return nil
	}
}

// mpvIPCCommand represents a JSON-RPC command for the mpv IPC protocol.
type mpvIPCCommand struct {
	Command []interface{} `json:"command"`
}

type MpvCommandId int

const (
	Position MpvCommandId = iota
	PlaytimeRemaining
)

func cmdInitializeControlSocket(m *model) tea.Cmd {
	return func() tea.Msg {
		i := 0
		c, err := ConnectIPC(m.ipcName)

		for ; err != nil; i++ {
			time.Sleep(100 * time.Millisecond)

			c, err = ConnectIPC(m.ipcName)
			if err == nil {
				break
			}
		}

		if err != nil {
			pushDebugList(m, fmt.Sprintf("Could not connect to mpv: %s", err))
		} else {
			pushDebugList(m, "Connected to mpv")
			m.controlSocket = c
		}

		go func() {
			response := make([]byte, 4096)
			for {
				count, err := c.Read(response)
				if err != nil {
					log.Error().Err(err).Msg("mpv IPC read ended")
					return
				}

				responseEvent := string(response[:count])
				m.sub <- responseMsg{
					event: strings.Split(responseEvent, "\n")[0],
				}
			}
		}()

		// request notification of percent remaining events
		posData, _ := json.Marshal(mpvIPCCommand{
			Command: []interface{}{"observe_property_string", Position, "percent-pos"},
		})
		writeMpvSocket(*m, string(posData)+"\n")
		// request notification of time-remaining events
		timeData, _ := json.Marshal(mpvIPCCommand{
			Command: []interface{}{"observe_property_string", PlaytimeRemaining, "time-remaining"},
		})
		writeMpvSocket(*m, string(timeData)+"\n")

		return vbsSetControlSocket{Socket: c}
	}
}

func cmdFullscreenMpv(m model) tea.Cmd {
	fullScreen := "no"
	if m.fullScreen {
		fullScreen = "yes"
	}

	return genericMpvCommand(m, "set_property", 0, nil, "fullscreen", fullScreen)
}

type progressMessage struct {
	Event string
	Id    MpvCommandId
	Name  string
	Data  string
}

type vbsSetControlSocket struct {
	Socket net.Conn
}

func updateModelFromEvent(m *model, event string) {
	scaleFactor := float64(100.0)

	// parse the json unstructured
	// check if this is an event
	// else does it have a request_id?

	var p progressMessage
	err := json.Unmarshal([]byte(event), &p)

	if err != nil {
		// if not, lets log the message
		pushDebugList(m, event)
	} else {
		switch p.Id {
		case Position:
			if p.Name == "percent-pos" {
				position, _ := strconv.ParseFloat(p.Data, 64)
				m.percent = position / scaleFactor
				pushDebugList(m, event)
			} else {
				pushDebugList(m, fmt.Sprintf("Bad data match %s", event))
			}

		case PlaytimeRemaining:
			remaining, _ := strconv.ParseFloat(p.Data, 64)
			m.remainingTime = remaining
		default:
			pushDebugList(m, event)
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalHeight = msg.Height
		m.terminalWidth = msg.Width
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

	case vbsSetControlSocket:
		m.controlSocket = msg.Socket

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)

		return m, cmd
	case responseMsg:
		m.responses++ // record external activity
		updateModelFromEvent(&m, msg.event)

		return m, waitForActivity(m.sub) // wait for next event
	// Is it a key press?
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Play):
			playCmdClosure := cmdPlayMpv(m)
			m.playing = !m.playing

			return m, playCmdClosure
		case key.Matches(msg, m.keys.Fullscreen):
			m.fullScreen = !m.fullScreen

			return m, cmdFullscreenMpv(m)
		case key.Matches(msg, m.keys.Up):
			return m, cmdForwardFiveSecondsMpv(m)
		case key.Matches(msg, m.keys.Down):
			return m, cmdBackwardFiveSecondsMpv(m)
		case key.Matches(msg, m.keys.Left):
			return m, cmdBackOneFrameMpv(m)
		case key.Matches(msg, m.keys.Right):
			return m, cmdForwardOneFrameMpv(m)
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true

			return m, cmdQuitMpv(m)
		case key.Matches(msg, m.keys.Debug):
			m.showDebug = !m.showDebug
		}
	}
	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func (m model) View() string {
	// The header
	h := titleStyle().Render("VBS player")
	//h = lipgloss.PlaceHorizontal(m.terminalWidth, lipgloss.Right, h)

	// Render the row
	s := fmt.Sprintf("\nCurrent item: %v\n", m.currentItem)
	s += fmt.Sprintf("Remaining time: %v\n", m.remainingTime)
	s += fmt.Sprintf("Output screen: %v\n", m.outputScreen)
	s += fmt.Sprintf("Fullscreen: %v\n", m.fullScreen)
	s += fmt.Sprintf("Playing: %v\n", m.playing)

	infoStyle := lipgloss.NewStyle().
		//BorderStyle(lipgloss.HiddenBorder()).
		//BorderForeground(lipgloss.Color("63")).
		Width(m.progress.Width - lipgloss.Width(h))

	s = lipgloss.JoinHorizontal(lipgloss.Top, infoStyle.Render(s), h)
	s += fmt.Sprintf("\n%s Events received: %d\n\n", m.spinner.View(), m.responses)
	s += "\n" + m.progress.ViewAs(m.percent) + "\n\n"
	// render debug messages
	if m.showDebug {
		debugHeaderStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("63")).
			//Padding(1, 1, 1, 1).
			Align(lipgloss.Right)
		s += debugHeaderStyle.Render("Debug messages") + "\n"

		debugWidth := m.progress.Width // cheat and use the same width as progress bar
		debugStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("63")).
			Width(debugWidth)

		debugMsg := ""

		for d := m.debug.Front(); d != nil; d = d.Next() {
			entry, ok := d.Value.(string)
			if ok {
				debugMsg += entry + "\n"
			}
		}

		s += "\n"
		s += debugStyle.Render(debugMsg)
		s += "\n"
	}

	helpView := m.help.View(m.keys)
	s += fmt.Sprintf("%s\n", helpView)

	// The footer
	//s += statusMessageStyle("\nPress q to quit. Press ? for help on commands\n")

	// Send the UI for rendering
	return appStyle.Copy().
		//Width(m.terminalWidth).
		//Height(m.terminalHeight).
		Render(s)
}

func play(cmd *coral.Command, args []string) {
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

	m.ipcName = GetIPCName()

	defer os.Remove(m.ipcName)

	pushDebugList(&m, m.ipcName)

	go runMpvPlayer(m.outputScreen, m.ipcName, m.currentItem)

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

			responseEvent := string(response[:count])
			for _, eventLine := range strings.Split(responseEvent, "\n") {
				sub <- responseMsg{
					event: eventLine,
				}
			}
		}
	}
}

type responseMsg struct {
	event string
}

// A command that waits for mpv responses.
func waitForActivity(sub chan responseMsg) tea.Cmd {
	// TODO: This extra indirection of using a channel is probably unneeded.
	// refactor this to use the socket directly.
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
		"--autofit=30%",
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

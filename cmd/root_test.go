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
	"testing"

	"github.com/rs/zerolog"
)

func TestRootCommand_Structure(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd should not be nil")
	}

	if rootCmd.Use != "vbs" {
		t.Errorf("Expected rootCmd.Use to be 'vbs', got '%s'", rootCmd.Use)
	}

	if rootCmd.Short == "" {
		t.Error("rootCmd.Short should not be empty")
	}
}

func TestRootCommand_HasSubcommands(t *testing.T) {
	// Core commands that should always be present
	requiredCommands := []string{
		"lighting-bridge",
		"ivs-bridge",
		"ivs-put",
		"chapterlist",
		"chaptersplit",
		"serve",
	}

	availableCommands := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		availableCommands[cmd.Name()] = true
	}

	for _, cmdName := range requiredCommands {
		if !availableCommands[cmdName] {
			t.Errorf("Expected subcommand '%s' not found in rootCmd", cmdName)
		}
	}

	// Verify we have at least the required number of commands
	if len(rootCmd.Commands()) < len(requiredCommands) {
		t.Errorf("Expected at least %d commands, got %d", len(requiredCommands), len(rootCmd.Commands()))
	}
}

func TestDebugFlag_EnablesDebugLogging(t *testing.T) {
	// Save original state
	originalDebug := Debug
	originalLevel := zerolog.GlobalLevel()

	// Test with debug enabled
	Debug = true
	initConfig()

	currentLevel := zerolog.GlobalLevel()
	if currentLevel != zerolog.DebugLevel {
		t.Errorf("Expected debug level when Debug=true, got %s", currentLevel)
	}

	// Test with debug disabled
	Debug = false
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	initConfig()

	currentLevel = zerolog.GlobalLevel()
	if currentLevel == zerolog.DebugLevel {
		t.Error("Expected non-debug level when Debug=false")
	}

	// Restore original state
	Debug = originalDebug
	zerolog.SetGlobalLevel(originalLevel)
}

func TestRootCommand_HasDebugFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("debug")
	if flag == nil {
		t.Fatal("Expected --debug flag to exist")
	}

	if flag.Shorthand != "d" {
		t.Errorf("Expected debug flag shorthand to be 'd', got '%s'", flag.Shorthand)
	}

	if flag.DefValue != "false" {
		t.Errorf("Expected debug flag default to be 'false', got '%s'", flag.DefValue)
	}
}

func TestRootCommand_HasConfigFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("config")
	if flag == nil {
		t.Fatal("Expected --config flag to exist")
	}

	if flag.Usage == "" {
		t.Error("Expected config flag to have usage description")
	}
}

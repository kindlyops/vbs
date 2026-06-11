// Copyright © 2026 Kindly Ops, LLC <support@kindlyops.com>
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
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveInputPath(t *testing.T) {
	abs := filepath.Join(t.TempDir(), "meeting.playlist")

	t.Run("absolute path is unchanged", func(t *testing.T) {
		t.Setenv("BUILD_WORKING_DIRECTORY", "/some/where")
		if got := resolveInputPath(abs); got != abs {
			t.Errorf("resolveInputPath(%q) = %q, want unchanged", abs, got)
		}
	})

	t.Run("relative path joins BUILD_WORKING_DIRECTORY", func(t *testing.T) {
		t.Setenv("BUILD_WORKING_DIRECTORY", "/launch/dir")
		got := resolveInputPath("../export/meeting.playlist")
		want := filepath.Join("/launch/dir", "../export/meeting.playlist")
		if got != want {
			t.Errorf("resolveInputPath = %q, want %q", got, want)
		}
	})

	t.Run("relative path unchanged when env unset", func(t *testing.T) {
		t.Setenv("BUILD_WORKING_DIRECTORY", "")
		if got := resolveInputPath("rel/meeting.playlist"); got != "rel/meeting.playlist" {
			t.Errorf("resolveInputPath = %q, want unchanged relative path", got)
		}
	})
}

func TestCheckPlaylistFile(t *testing.T) {
	dir := t.TempDir()

	present := filepath.Join(dir, "present.playlist")
	if err := os.WriteFile(present, []byte("data"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := checkPlaylistFile(present); err != nil {
		t.Errorf("existing file should pass, got: %v", err)
	}

	missing := filepath.Join(dir, "missing.playlist")
	err := checkPlaylistFile(missing)
	if err == nil {
		t.Fatal("missing file should error")
	}
	if !strings.Contains(err.Error(), "no such file") {
		t.Errorf("error %q should mention 'no such file'", err.Error())
	}
}

func TestDescribeSource(t *testing.T) {
	cases := []struct {
		name string
		item Item
		want string
	}{
		{
			"pub track",
			Item{Location: &Location{KeySymbol: "sjj", Track: 135, Type: 0}},
			"pub sjj track 135",
		},
		{
			"book/chapter",
			Item{Location: &Location{KeySymbol: "nwt", BookNumber: 23, ChapterNumber: 5, Type: 0}},
			"book 23:5",
		},
		{
			"docid",
			Item{Location: &Location{DocumentID: 1112024040, Track: 1, Type: 3}},
			"docid 1112024040",
		},
		{
			"image",
			Item{Image: &EmbeddedImage{OriginalFilename: "picture.jpg"}},
			"embedded image",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := describeSource(tc.item); got != tc.want {
				t.Errorf("describeSource() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBuildPrintView(t *testing.T) {
	pl := parseFixture(t)
	view := buildPrintView(pl)

	if view.Name != "synthetic event" {
		t.Errorf("view.Name = %q", view.Name)
	}
	if len(view.Items) != 4 {
		t.Fatalf("len(view.Items) = %d, want 4", len(view.Items))
	}

	song := view.Items[0]
	if song.Source != "pub sjj track 135" {
		t.Errorf("song.Source = %q", song.Source)
	}
	if math.Abs(song.DurationSec-139.006) > 1e-6 {
		t.Errorf("song.DurationSec = %v, want 139.006", song.DurationSec)
	}

	chapter := view.Items[1]
	if len(chapter.Markers) != 3 {
		t.Errorf("chapter markers = %d, want 3", len(chapter.Markers))
	}

	img := view.Items[2]
	if img.Source != "embedded image" {
		t.Errorf("img.Source = %q", img.Source)
	}
	if math.Abs(img.DurationSec-4.0) > 1e-6 {
		t.Errorf("img.DurationSec = %v, want 4.0", img.DurationSec)
	}
}

func TestRenderJSON_RoundTrips(t *testing.T) {
	view := buildPrintView(parseFixture(t))

	var buf strings.Builder
	if err := renderJSON(&buf, view); err != nil {
		t.Fatalf("renderJSON: %v", err)
	}

	var back printView
	if err := json.Unmarshal([]byte(buf.String()), &back); err != nil {
		t.Fatalf("json did not round-trip: %v", err)
	}
	if back.Name != view.Name || len(back.Items) != len(view.Items) {
		t.Errorf("round-trip mismatch: %+v", back)
	}
}

func TestRenderText_ContainsKeyData(t *testing.T) {
	view := buildPrintView(parseFixture(t))

	var buf strings.Builder
	if err := renderText(&buf, view); err != nil {
		t.Fatalf("renderText: %v", err)
	}
	out := buf.String()

	wants := []string{
		"synthetic event", "pub sjj track 135", "book 23:5",
		"embedded image", "docid 1112024040",
	}
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("text output missing %q\n%s", want, out)
		}
	}
}

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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sampleManifest() buildManifest {
	return buildManifest{
		Name:       "event Dec 2nd",
		Slug:       "event-dec-2nd",
		Language:   langInfo{ID: 420, Code: "ASL"},
		Resolution: "720p",
		BuiltAt:    "2026-06-11T12:00:00Z",
		Cues: []cue{
			{
				Index: 1, Label: "First Clip", Kind: "video",
				Clip: "clips/01-opening-song.mp4", SourceMedia: "media/sjj_ASL_135_r720P.mp4",
				EndActionRaw: 2, DurationSec: 139.006, Thumbnail: "thumbs/01.jpeg",
			},
			{
				Index: 2, Label: "Part 1 Section 5:1, 2", Kind: "video",
				Clip:        "clips/02-part-1-section-5-1-2.mp4",
				SourceMedia: "media/nwt_23_Isa_ASL_05_r720P.mp4",
				Markers: []cueMarker{
					{Label: "Marker one", Start: 2.068, Duration: 18.651},
					{Label: "Marker two", Start: 20.720, Duration: 35.602},
				},
				Cut:          &cutInfo{RequestedStart: 2.068, SnappedStart: 2.002, LeadIn: 0.066, End: 56.322, Duration: 54.320},
				EndActionRaw: 2, DurationSec: 54.320, Thumbnail: "thumbs/02.jpeg",
			},
			{
				Index: 3, Label: "picture.jpg", Kind: "image",
				Clip: "clips/03-picture.jpg", EndActionRaw: 2, DurationSec: 4.0, Thumbnail: "thumbs/03.jpeg",
			},
		},
	}
}

func TestFormatTimecode(t *testing.T) {
	cases := map[float64]string{
		0:       "0:00.0",
		4.0:     "0:04.0",
		54.32:   "0:54.3",
		139.006: "2:19.0",
		793.5:   "13:13.5",
	}
	for in, want := range cases {
		if got := formatTimecode(in); got != want {
			t.Errorf("formatTimecode(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestWritePlaylistJSON_RoundTrips(t *testing.T) {
	dir := t.TempDir()
	manifest := sampleManifest()

	if err := writePlaylistJSON(dir, manifest); err != nil {
		t.Fatalf("writePlaylistJSON: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "playlist.json"))
	if err != nil {
		t.Fatalf("read playlist.json: %v", err)
	}

	var back buildManifest
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("playlist.json did not parse: %v", err)
	}
	if back.Slug != "event-dec-2nd" || len(back.Cues) != 3 {
		t.Errorf("round-trip mismatch: %+v", back)
	}
	if back.Cues[1].Cut == nil || back.Cues[1].Cut.SnappedStart != 2.002 {
		t.Errorf("cut info lost in round-trip: %+v", back.Cues[1].Cut)
	}
	if back.Cues[2].Cut != nil {
		t.Error("image cue should have no cut")
	}
}

func TestRenderCueSheet(t *testing.T) {
	out := renderCueSheet(sampleManifest())

	for _, want := range []string{
		"us-letter",
		"event Dec 2nd",
		"ASL (420)",
		"720p",
		"clips/02-part-1-section-5-1-2.mp4",
		"thumbs/02.jpeg",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("cue sheet missing %q", want)
		}
	}

	// One table row per cue (count the leading "  [%d]," cells via the index).
	if n := strings.Count(out, "image(\"thumbs/"); n != 3 {
		t.Errorf("expected 3 thumbnail cells, got %d", n)
	}
}

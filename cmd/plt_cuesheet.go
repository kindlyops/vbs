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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// buildManifest is the playlist.json contract: a self-describing, ordered list
// of play-ready cues plus the context needed to regenerate or hand off the
// working directory (consumed by the Phase 3 .mitti writer).
type buildManifest struct {
	Name       string   `json:"name"`
	Slug       string   `json:"slug"`
	Language   langInfo `json:"language"`
	Resolution string   `json:"resolution"`
	BuiltAt    string   `json:"builtAt"`
	Cues       []cue    `json:"cues"`
}

type langInfo struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
}

type cue struct {
	Index        int         `json:"index"`
	Label        string      `json:"label"`
	Kind         string      `json:"kind"`
	Clip         string      `json:"clip"`
	SourceMedia  string      `json:"sourceMedia,omitempty"`
	Markers      []cueMarker `json:"markers,omitempty"`
	Cut          *cutInfo    `json:"cut,omitempty"`
	EndActionRaw int         `json:"endActionRaw"`
	DurationSec  float64     `json:"durationSec"`
	Thumbnail    string      `json:"thumbnail"`
}

type cueMarker struct {
	Label    string  `json:"label"`
	Start    float64 `json:"start"`
	Duration float64 `json:"duration"`
}

type cutInfo struct {
	RequestedStart float64 `json:"requestedStart"`
	SnappedStart   float64 `json:"snappedStart"`
	LeadIn         float64 `json:"leadIn"`
	End            float64 `json:"end"`
	Duration       float64 `json:"duration"`
}

// writePlaylistJSON writes the manifest to playlist.json in dir.
func writePlaylistJSON(dir string, manifest buildManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode playlist.json: %w", err)
	}
	path := filepath.Join(dir, "playlist.json")
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("could not write playlist.json: %w", err)
	}
	return nil
}

// formatTimecode renders seconds as m:ss.t (tenths).
func formatTimecode(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	minutes := int(seconds) / 60
	rem := seconds - float64(minutes*60)
	return fmt.Sprintf("%d:%04.1f", minutes, rem)
}

// renderCueSheet builds the Typst source for the technical-director cue sheet.
func renderCueSheet(manifest buildManifest) string {
	var b strings.Builder

	total := 0.0
	for _, c := range manifest.Cues {
		total += c.DurationSec
	}

	b.WriteString("#set page(paper: \"us-letter\", margin: 1.5cm)\n")
	b.WriteString("#set text(size: 9pt)\n\n")
	fmt.Fprintf(&b, "= %s\n\n", manifest.Name)
	fmt.Fprintf(&b, "Language: %s (%d) · Resolution: %s · Cues: %d · Runtime: %s · Built: %s\n\n",
		manifest.Language.Code, manifest.Language.ID, manifest.Resolution,
		len(manifest.Cues), formatTimecode(total), manifest.BuiltAt)

	b.WriteString("#table(\n")
	b.WriteString("  columns: (auto, auto, 1fr, auto, auto, auto, auto),\n")
	b.WriteString("  table.header[\\#][Thumb][Cue][Dur][Lead][After][Elapsed],\n")

	elapsed := 0.0
	for _, c := range manifest.Cues {
		elapsed += c.DurationSec
		thumb := "[]"
		if c.Thumbnail != "" {
			thumb = fmt.Sprintf("image(%q, width: 2cm)", c.Thumbnail)
		}
		lead := ""
		if c.Cut != nil {
			lead = fmt.Sprintf("%.3f", c.Cut.LeadIn)
		}
		fmt.Fprintf(&b, "  [%d], [#%s], [%s \\ #raw(%q)], [%s], [%s], [%d], [%s],\n",
			c.Index, thumb, escapeTypst(c.Label), c.Clip,
			formatTimecode(c.DurationSec), lead, c.EndActionRaw, formatTimecode(elapsed))
	}

	b.WriteString(")\n")
	return b.String()
}

// escapeTypst escapes characters that would otherwise be Typst markup.
func escapeTypst(s string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\", "#", "\\#", "*", "\\*", "_", "\\_",
		"$", "\\$", "[", "\\[", "]", "\\]", "@", "\\@",
	)
	return replacer.Replace(s)
}

// writeCueSheet writes cuesheet.typ and, when typst is on PATH, compiles
// cuesheet.pdf. It returns whether a PDF was produced.
func writeCueSheet(dir string, manifest buildManifest) (bool, error) {
	typPath := filepath.Join(dir, "cuesheet.typ")
	if err := os.WriteFile(typPath, []byte(renderCueSheet(manifest)), 0o600); err != nil {
		return false, fmt.Errorf("could not write cuesheet.typ: %w", err)
	}

	if _, err := exec.LookPath("typst"); err != nil {
		return false, nil
	}

	pdfPath := filepath.Join(dir, "cuesheet.pdf")
	cmd := exec.Command("typst", "compile", "--root", dir, typPath, pdfPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("typst compile failed: %s: %w", out, err)
	}
	return true, nil
}

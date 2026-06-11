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
	"io"
	"os"
	"text/tabwriter"

	"github.com/muesli/coral"
	"github.com/rs/zerolog/log"
)

var pltPrintJSON bool

var pltCmd = &coral.Command{
	Use:   "plt",
	Short: "Work with purple playlists.",
	Long: `Parse, build, and prepare media for purple playlist exports from the
source app for use in live meetings.`,
}

var pltPrintCmd = &coral.Command{
	Use:   "print <playlist>",
	Short: "Parse and pretty-print a purple playlist.",
	Long: `Parse a purple playlist export and print its cues. Works entirely
offline; no media is downloaded.`,
	Run:  runPltPrint,
	Args: coral.ExactArgs(1),
}

func runPltPrint(cmd *coral.Command, args []string) {
	path := args[0]

	arc, err := sniffPlaylist(path)
	if err != nil {
		log.Fatal().Err(err).Msgf("Not a valid purple playlist: %s", path)
	}
	defer func() { _ = arc.Close() }()

	if arc.schemaVersion != verifiedSchemaVersion {
		log.Warn().Msgf("schema version %d differs from verified version %d; proceeding because required tables are present",
			arc.schemaVersion, verifiedSchemaVersion)
	}

	pl, err := parsePlaylist(arc)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not parse playlist")
	}

	view := buildPrintView(pl)

	if pltPrintJSON {
		err = renderJSON(os.Stdout, view)
	} else {
		err = renderText(os.Stdout, view)
	}
	if err != nil {
		log.Fatal().Err(err).Msg("Could not render playlist")
	}
}

// printView is the offline summary rendered by plt print, shared by the text
// and JSON outputs so both show the same data.
type printView struct {
	Name  string      `json:"name"`
	Items []printItem `json:"items"`
}

type printItem struct {
	Position    int           `json:"position"`
	Label       string        `json:"label"`
	Source      string        `json:"source"`
	Kind        string        `json:"kind"`
	DurationSec float64       `json:"durationSec"`
	EndAction   int           `json:"endAction"`
	Markers     []printMarker `json:"markers,omitempty"`
}

type printMarker struct {
	Label       string  `json:"label"`
	StartSec    float64 `json:"startSec"`
	DurationSec float64 `json:"durationSec"`
}

// buildPrintView projects the parsed playlist into the print summary.
func buildPrintView(pl *Playlist) printView {
	view := printView{Name: pl.Name, Items: make([]printItem, 0, len(pl.Items))}

	for _, it := range pl.Items {
		pi := printItem{
			Position:    it.Position,
			Label:       it.Label,
			Source:      describeSource(it),
			Kind:        itemKind(it),
			DurationSec: itemDurationSec(it),
			EndAction:   it.EndAction,
		}
		for _, m := range it.Markers {
			pi.Markers = append(pi.Markers, printMarker{
				Label:       m.Label,
				StartSec:    ticksToSeconds(m.StartTimeTicks),
				DurationSec: ticksToSeconds(m.DurationTicks),
			})
		}
		view.Items = append(view.Items, pi)
	}
	return view
}

// describeSource renders a one-line description of where an item's media comes
// from, following the catalog resolution rule (KeySymbol+Track, book/chapter, docid).
func describeSource(it Item) string {
	if it.IsImage() {
		return "embedded image"
	}

	loc := it.Location
	if loc == nil {
		return "unknown source"
	}

	switch {
	case loc.KeySymbol == "nwt" && loc.BookNumber > 0:
		return fmt.Sprintf("book %d:%d", loc.BookNumber, loc.ChapterNumber)
	case loc.KeySymbol != "" && loc.Track > 0:
		return fmt.Sprintf("pub %s track %d", loc.KeySymbol, loc.Track)
	case loc.Type == 3 && loc.DocumentID > 0:
		return fmt.Sprintf("docid %d", loc.DocumentID)
	default:
		return fmt.Sprintf("unsupported location (type %d)", loc.Type)
	}
}

// itemKind reports "image" or "video" for an item.
func itemKind(it Item) string {
	if it.IsImage() {
		return "image"
	}
	return "video"
}

// itemDurationSec returns the cue's nominal duration in seconds.
func itemDurationSec(it Item) float64 {
	if it.IsImage() {
		return ticksToSeconds(it.Image.DurationTicks)
	}
	if it.Location != nil {
		return ticksToSeconds(it.Location.BaseDurationTicks)
	}
	return 0
}

// renderJSON writes the print view as indented JSON.
func renderJSON(w io.Writer, view printView) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(view); err != nil {
		return fmt.Errorf("could not encode JSON: %w", err)
	}
	return nil
}

// renderText writes the print view as an aligned table.
func renderText(w io.Writer, view printView) error {
	if _, err := fmt.Fprintf(w, "Playlist: %s\n\n", view.Name); err != nil {
		return fmt.Errorf("could not write header: %w", err)
	}

	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "#\tLABEL\tSOURCE\tDURATION\tMARKERS\tEND"); err != nil {
		return fmt.Errorf("could not write table header: %w", err)
	}

	for _, it := range view.Items {
		markers := ""
		if len(it.Markers) > 0 {
			markers = fmt.Sprintf("%d", len(it.Markers))
		}
		if _, err := fmt.Fprintf(tw, "%d\t%s\t%s\t%.1fs\t%s\t%d\n",
			it.Position, it.Label, it.Source, it.DurationSec, markers, it.EndAction); err != nil {
			return fmt.Errorf("could not write table row: %w", err)
		}
	}

	if err := tw.Flush(); err != nil {
		return fmt.Errorf("could not flush table: %w", err)
	}
	return nil
}

func init() {
	pltPrintCmd.Flags().BoolVar(&pltPrintJSON, "json", false, "emit the playlist as JSON instead of a table")

	pltCmd.AddCommand(pltPrintCmd)
	rootCmd.AddCommand(pltCmd)
}

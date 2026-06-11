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

import "testing"

func parseFixture(t *testing.T) *Playlist {
	t.Helper()

	path := writePlaylistFixture(t, fixtureOptions{})
	arc, err := sniffPlaylist(path)
	if err != nil {
		t.Fatalf("sniff fixture: %v", err)
	}
	t.Cleanup(func() { _ = arc.Close() })

	pl, err := parsePlaylist(arc)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return pl
}

func TestParsePlaylist_NameAndOrder(t *testing.T) {
	pl := parseFixture(t)

	if pl.Name != "synthetic event" {
		t.Errorf("Name = %q, want %q", pl.Name, "synthetic event")
	}
	if len(pl.Items) != 4 {
		t.Fatalf("len(Items) = %d, want 4", len(pl.Items))
	}
	for i, it := range pl.Items {
		if it.Position != i {
			t.Errorf("Items[%d].Position = %d, want %d", i, it.Position, i)
		}
	}
}

func TestParsePlaylist_PubTrackItem(t *testing.T) {
	song := parseFixture(t).Items[0]

	if song.IsImage() {
		t.Error("song should not be an image cue")
	}
	if song.Location == nil {
		t.Fatal("song should have a Location")
	}
	if song.Location.KeySymbol != "sjj" || song.Location.Track != 135 {
		t.Errorf("song location = %+v, want KeySymbol sjj track 135", song.Location)
	}
	if song.Location.BaseDurationTicks != 1390060000 {
		t.Errorf("song BaseDurationTicks = %d, want 1390060000", song.Location.BaseDurationTicks)
	}
	if len(song.Markers) != 0 {
		t.Errorf("song should have no markers, got %d", len(song.Markers))
	}
}

func TestParsePlaylist_BookChapterMarkers(t *testing.T) {
	chapter := parseFixture(t).Items[1]

	if chapter.Location == nil || chapter.Location.BookNumber != 23 || chapter.Location.ChapterNumber != 5 {
		t.Fatalf("book/chapter location = %+v, want book 23 chapter 5", chapter.Location)
	}
	if len(chapter.Markers) != 3 {
		t.Fatalf("chapter markers = %d, want 3", len(chapter.Markers))
	}
	wantLabels := []string{"Marker one", "Marker two", "Marker three"}
	for i, want := range wantLabels {
		if chapter.Markers[i].Label != want {
			t.Errorf("marker[%d].Label = %q, want %q", i, chapter.Markers[i].Label, want)
		}
	}
	if chapter.Markers[0].StartTimeTicks != 20680000 {
		t.Errorf("marker[0].StartTimeTicks = %d, want 20680000", chapter.Markers[0].StartTimeTicks)
	}
}

func TestParsePlaylist_ImageCue(t *testing.T) {
	img := parseFixture(t).Items[2]

	if !img.IsImage() {
		t.Fatal("third item should be an image cue")
	}
	if img.Location != nil {
		t.Error("image cue should not have a Location")
	}
	if img.Image.DurationTicks != 40000000 {
		t.Errorf("image DurationTicks = %d, want 40000000", img.Image.DurationTicks)
	}
	if img.Image.OriginalFilename != "picture.jpg" {
		t.Errorf("image OriginalFilename = %q, want picture.jpg", img.Image.OriginalFilename)
	}
}

func TestParsePlaylist_DocidItem(t *testing.T) {
	doc := parseFixture(t).Items[3]

	if doc.Location == nil || doc.Location.Type != 3 || doc.Location.DocumentID != 1112024040 {
		t.Fatalf("docid location = %+v, want type 3 docid 1112024040", doc.Location)
	}
	if doc.Location.KeySymbol != "" {
		t.Errorf("docid KeySymbol = %q, want empty", doc.Location.KeySymbol)
	}
}

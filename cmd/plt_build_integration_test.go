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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

// buildMediaFixtureServer serves a single generated video plus media-API JSON
// (with example.invalid-style local URLs) describing one 720p rendition of it.
func buildMediaFixtureServer(t *testing.T, video []byte) (*httptest.Server, *int32) {
	t.Helper()

	var videoHits int32
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	mux.HandleFunc("/media/video.mp4", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&videoHits, 1)
		_, _ = w.Write(video)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		resp := mediaResponse{Files: map[string]mediaLang{
			"ASL": {MP4: []mediaItem{{
				Title: "synthetic", Label: "720p",
				Filesize: int64(len(video)), Duration: 165, FrameWidth: 1280, FrameHeight: 720,
				File: mediaFile{URL: srv.URL + "/media/video.mp4", Checksum: md5Hex(video)},
			}}},
		}}
		_ = json.NewEncoder(w).Encode(resp)
	})
	return srv, &videoHits
}

func TestBuildPlaylist_EndToEnd(t *testing.T) {
	requireFFmpeg(t)

	// A source long enough to reach the fixture's farthest segment marker (~126s).
	videoPath := filepath.Join(t.TempDir(), "source.mp4")
	makeTestVideo(t, videoPath, 170)
	video, err := os.ReadFile(videoPath)
	if err != nil {
		t.Fatalf("read generated video: %v", err)
	}

	srv, videoHits := buildMediaFixtureServer(t, video)

	// Redirect the shared cache and working dir into the test's temp space.
	t.Setenv("HOME", t.TempDir())
	outDir := t.TempDir()
	pltBuildOut = outDir
	pltBuildResolution = "720p"
	pltBuildLang = ""
	t.Cleanup(func() { pltBuildOut = "."; pltBuildResolution = "720p"; pltBuildLang = "" })

	arc, err := sniffPlaylist(writePlaylistFixture(t, fixtureOptions{}))
	if err != nil {
		t.Fatalf("sniff: %v", err)
	}
	t.Cleanup(func() { _ = arc.Close() })
	playlist, err := parsePlaylist(arc)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	manifest, err := buildPlaylist(arc, playlist, srv.URL)
	if err != nil {
		t.Fatalf("buildPlaylist: %v", err)
	}

	assertBuildOutputs(t, outDir, manifest)

	// The book/chapter item has two contiguous markers plus one far marker, so it
	// becomes two lettered sub-clips: 4 items -> 5 cues.
	if len(manifest.Cues) != 5 {
		t.Errorf("cues = %d, want 5 (image + pub + 2 book/chapter sub-clips + docid)", len(manifest.Cues))
	}

	firstHits := atomic.LoadInt32(videoHits)
	// Re-run must reuse the cache, not re-download.
	if _, err := buildPlaylist(arc, playlist, srv.URL); err != nil {
		t.Fatalf("second buildPlaylist: %v", err)
	}
	if got := atomic.LoadInt32(videoHits); got != firstHits {
		t.Errorf("re-run downloaded again: video hits %d -> %d", firstHits, got)
	}
}

func assertBuildOutputs(t *testing.T, outDir string, manifest buildManifest) {
	t.Helper()

	workDir := filepath.Join(outDir, manifest.Slug)
	if _, err := os.Stat(filepath.Join(workDir, "playlist.json")); err != nil {
		t.Errorf("playlist.json missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workDir, "cuesheet.typ")); err != nil {
		t.Errorf("cuesheet.typ missing: %v", err)
	}

	var segments, image int
	for _, c := range manifest.Cues {
		assertClipExists(t, workDir, c)
		if c.Kind == "image" {
			image++
		}
		if c.Cut != nil {
			segments++
		}
	}
	if image != 1 {
		t.Errorf("image cues = %d, want 1", image)
	}
	if segments != 2 {
		t.Errorf("cut (segment) cues = %d, want 2", segments)
	}
}

func assertClipExists(t *testing.T, workDir string, c cue) {
	t.Helper()
	info, err := os.Stat(filepath.Join(workDir, filepath.FromSlash(c.Clip)))
	if err != nil || info.Size() == 0 {
		t.Errorf("clip %q missing or empty: %v", c.Clip, err)
	}
}

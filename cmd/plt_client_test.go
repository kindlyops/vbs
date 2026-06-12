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
	"net/http"
	"net/http/httptest"
	"testing"
)

// sanitizedMediaJSON mirrors a real media-API response shape with every
// hostname rewritten to example.invalid. It offers four renditions for ASL.
const sanitizedMediaJSON = `{
  "pubName": "Synthetic Songs",
  "pub": "sjj",
  "track": 135,
  "fileformat": "mp4",
  "files": {
    "ASL": {
      "MP4": [
        {"title": "song", "label": "240p", "filesize": 4000000, "duration": 139.006,
         "frameWidth": 426, "frameHeight": 240,
         "file": {"url": "https://example.invalid/a/x/o/sjj_ASL_135_r240P.mp4",
                  "checksum": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}},
        {"title": "song", "label": "360p", "filesize": 7000000, "duration": 139.006,
         "frameWidth": 640, "frameHeight": 360,
         "file": {"url": "https://example.invalid/a/x/o/sjj_ASL_135_r360P.mp4",
                  "checksum": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},
        {"title": "song", "label": "480p", "filesize": 11000000, "duration": 139.006,
         "frameWidth": 854, "frameHeight": 480,
         "file": {"url": "https://example.invalid/a/x/o/sjj_ASL_135_r480P.mp4",
                  "checksum": "cccccccccccccccccccccccccccccccc"}},
        {"title": "song", "label": "720p", "filesize": 16112944, "duration": 139.006,
         "frameWidth": 1280, "frameHeight": 720,
         "file": {"url": "https://example.invalid/a/x/o/sjj_ASL_135_r720P.mp4",
                  "checksum": "1915564d166a4b264eda0f607ccda127"}}
      ]
    }
  }
}`

func mediaTestServer(t *testing.T, body string) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchMedia_ParsesRenditions(t *testing.T) {
	srv := mediaTestServer(t, sanitizedMediaJSON)

	resp, err := fetchMedia(srv.Client(), srv.URL, "ASL", &Location{KeySymbol: "sjj", Track: 135})
	if err != nil {
		t.Fatalf("fetchMedia: %v", err)
	}
	mp4 := resp.Files["ASL"].MP4
	if len(mp4) != 4 {
		t.Fatalf("renditions = %d, want 4", len(mp4))
	}
}

func TestSelectRendition(t *testing.T) {
	srv := mediaTestServer(t, sanitizedMediaJSON)
	resp, err := fetchMedia(srv.Client(), srv.URL, "ASL", &Location{KeySymbol: "sjj", Track: 135})
	if err != nil {
		t.Fatalf("fetchMedia: %v", err)
	}

	t.Run("exact 720p", func(t *testing.T) {
		item, fellBack, err := selectRendition(resp, "ASL", "720p")
		if err != nil {
			t.Fatalf("selectRendition: %v", err)
		}
		if fellBack {
			t.Error("720p is present; should not fall back")
		}
		if item.File.Checksum != "1915564d166a4b264eda0f607ccda127" {
			t.Errorf("checksum = %q", item.File.Checksum)
		}
	})

	t.Run("absent resolution falls back to highest", func(t *testing.T) {
		item, fellBack, err := selectRendition(resp, "ASL", "1080p")
		if err != nil {
			t.Fatalf("selectRendition: %v", err)
		}
		if !fellBack {
			t.Error("1080p is absent; should fall back")
		}
		if item.Label != "720p" || item.FrameHeight != 720 {
			t.Errorf("fallback picked %q (%dp), want highest 720p", item.Label, item.FrameHeight)
		}
	})

	t.Run("missing language is an error", func(t *testing.T) {
		if _, _, err := selectRendition(resp, "ZZZ", "720p"); err == nil {
			t.Error("expected error for missing language")
		}
	})
}

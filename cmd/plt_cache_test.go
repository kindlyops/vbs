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
	"crypto/md5" //nolint:gosec // mirrors the production MD5 verification
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func md5Hex(b []byte) string {
	sum := md5.Sum(b) //nolint:gosec // mirrors the production MD5 verification
	return hex.EncodeToString(sum[:])
}

// mediaFileServer serves fixed bytes at any path and counts requests.
func mediaFileServer(t *testing.T, body []byte) (*httptest.Server, *int32) {
	t.Helper()

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func TestDownloadAndVerify(t *testing.T) {
	body := []byte("hello media bytes")
	srv, _ := mediaFileServer(t, body)
	dest := filepath.Join(t.TempDir(), "file.mp4")

	t.Run("good size and checksum", func(t *testing.T) {
		err := downloadAndVerify(srv.Client(), srv.URL+"/file.mp4", dest, int64(len(body)), md5Hex(body))
		if err != nil {
			t.Fatalf("downloadAndVerify: %v", err)
		}
	})

	t.Run("checksum mismatch is an error", func(t *testing.T) {
		err := downloadAndVerify(srv.Client(), srv.URL+"/file.mp4", dest, int64(len(body)), "deadbeef")
		if err == nil {
			t.Fatal("expected checksum mismatch error")
		}
	})

	t.Run("size mismatch is an error", func(t *testing.T) {
		err := downloadAndVerify(srv.Client(), srv.URL+"/file.mp4", dest, 99999, md5Hex(body))
		if err == nil {
			t.Fatal("expected size mismatch error")
		}
	})
}

func TestFetchToCache_ReusesCache(t *testing.T) {
	body := []byte("song bytes here")
	srv, hits := mediaFileServer(t, body)
	cacheDir := t.TempDir()

	item := mediaItem{
		Label:    "720p",
		Filesize: int64(len(body)),
		Duration: 139.006,
		File:     mediaFile{URL: srv.URL + "/o/sjj_ASL_135_r720P.mp4", Checksum: md5Hex(body)},
	}

	first, err := fetchToCache(srv.Client(), cacheDir, item)
	if err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	if filepath.Base(first) != "sjj_ASL_135_r720P.mp4" {
		t.Errorf("cached name = %q, want basename of URL", filepath.Base(first))
	}

	second, err := fetchToCache(srv.Client(), cacheDir, item)
	if err != nil {
		t.Fatalf("second fetch: %v", err)
	}
	if second != first {
		t.Errorf("cache path changed: %q vs %q", first, second)
	}
	if got := atomic.LoadInt32(hits); got != 1 {
		t.Errorf("server hit %d times, want 1 (second call should be a cache hit)", got)
	}
}

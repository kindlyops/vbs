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
	"crypto/md5" //nolint:gosec // the media API publishes MD5 checksums; this verifies downloads, not security
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
)

// mediaResponse is the subset of the publisher's media-API response we read.
type mediaResponse struct {
	Files map[string]mediaLang `json:"files"`
}

type mediaLang struct {
	MP4 []mediaItem `json:"MP4"`
}

type mediaItem struct {
	Title       string    `json:"title"`
	Label       string    `json:"label"`
	Filesize    int64     `json:"filesize"`
	Duration    float64   `json:"duration"`
	FrameWidth  int       `json:"frameWidth"`
	FrameHeight int       `json:"frameHeight"`
	File        mediaFile `json:"file"`
}

type mediaFile struct {
	URL      string `json:"url"`
	Checksum string `json:"checksum"`
}

// fetchMedia queries the media API for a location and decodes the response.
func fetchMedia(client *http.Client, base, langCode string, loc *Location) (mediaResponse, error) {
	var out mediaResponse

	endpoint, err := buildMediaURL(base, langCode, loc)
	if err != nil {
		return out, err
	}

	resp, err := client.Get(endpoint)
	if err != nil {
		return out, fmt.Errorf("media API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return out, fmt.Errorf("media API returned status %d for %+v", resp.StatusCode, *loc)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return out, fmt.Errorf("could not read media API response: %w", err)
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, fmt.Errorf("could not parse media API response: %w", err)
	}
	return out, nil
}

// selectRendition picks the requested resolution for a language, or the
// highest available when the requested one is absent. The bool reports whether
// a fallback happened so the caller can warn naming the item.
func selectRendition(resp mediaResponse, langCode, resolution string) (mediaItem, bool, error) {
	lang, ok := resp.Files[langCode]
	if !ok || len(lang.MP4) == 0 {
		return mediaItem{}, false, fmt.Errorf("no MP4 renditions for language %q", langCode)
	}

	for _, item := range lang.MP4 {
		if item.Label == resolution {
			return item, false, nil
		}
	}

	highest := lang.MP4[0]
	for _, item := range lang.MP4[1:] {
		if item.FrameHeight > highest.FrameHeight {
			highest = item
		}
	}
	return highest, true, nil
}

// mediaSidecar records what was cached alongside a downloaded media file.
type mediaSidecar struct {
	URL      string  `json:"url"`
	Size     int64   `json:"size"`
	Checksum string  `json:"checksum"`
	Duration float64 `json:"duration"`
}

// fetchToCache ensures the rendition is present in cacheDir, downloading and
// verifying it when absent. It is idempotent: a present, size-matching file
// with a matching sidecar is reused. Returns the cached file path.
func fetchToCache(client *http.Client, cacheDir string, item mediaItem) (string, error) {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("could not create cache dir: %w", err)
	}

	dest := filepath.Join(cacheDir, path.Base(item.File.URL))
	if cacheHit(dest, item) {
		return dest, nil
	}

	err := downloadAndVerify(client, item.File.URL, dest, item.Filesize, item.File.Checksum)
	if err != nil {
		// One re-download on mismatch, then hard error.
		err = downloadAndVerify(client, item.File.URL, dest, item.Filesize, item.File.Checksum)
		if err != nil {
			return "", err
		}
	}

	if err := writeSidecar(dest, item); err != nil {
		return "", err
	}
	return dest, nil
}

// cacheHit reports whether dest already satisfies the rendition: the file and
// its sidecar exist and the recorded size and checksum match.
func cacheHit(dest string, item mediaItem) bool {
	info, err := os.Stat(dest)
	if err != nil {
		return false
	}

	data, err := os.ReadFile(dest + ".json")
	if err != nil {
		return false
	}
	var side mediaSidecar
	if err := json.Unmarshal(data, &side); err != nil {
		return false
	}
	return info.Size() == side.Size && side.Checksum == item.File.Checksum
}

// downloadAndVerify downloads url to dest and checks size and MD5 (when known).
func downloadAndVerify(client *http.Client, url, dest string, wantSize int64, wantChecksum string) error {
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("download failed for %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download for %s returned status %d", url, resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("could not create %s: %w", dest, err)
	}

	hash := md5.New() //nolint:gosec // matching the API's published MD5
	size, err := io.Copy(io.MultiWriter(out, hash), resp.Body)
	closeErr := out.Close()
	if err != nil {
		return fmt.Errorf("could not write %s: %w", dest, err)
	}
	if closeErr != nil {
		return fmt.Errorf("could not close %s: %w", dest, closeErr)
	}

	if wantSize > 0 && size != wantSize {
		return fmt.Errorf("size mismatch for %s: got %d, want %d", url, size, wantSize)
	}
	if wantChecksum != "" {
		got := hex.EncodeToString(hash.Sum(nil))
		if got != wantChecksum {
			return fmt.Errorf("checksum mismatch for %s: got %s, want %s", url, got, wantChecksum)
		}
	}
	return nil
}

// writeSidecar records the cached file's provenance next to it.
func writeSidecar(dest string, item mediaItem) error {
	side := mediaSidecar{
		URL:      item.File.URL,
		Size:     item.Filesize,
		Checksum: item.File.Checksum,
		Duration: item.Duration,
	}
	data, err := json.MarshalIndent(side, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode sidecar: %w", err)
	}
	if err := os.WriteFile(dest+".json", data, 0o600); err != nil {
		return fmt.Errorf("could not write sidecar: %w", err)
	}
	return nil
}

// shapeKind identifies how a Location resolves to a catalog media query.
type shapeKind int

const (
	shapePub         shapeKind = iota // KeySymbol + Track (e.g. a song or story)
	shapeBookChapter                  // KeySymbol "nwt" + BookNumber + ChapterNumber
	shapeDocid                        // Type 3, resolved by DocumentId
)

// classifyLocation determines a Location's catalog shape, following the
// resolution rule: a book/chapter reference (nwt + book), then a tracked publication,
// then a document id. Anything else is an error naming the row.
func classifyLocation(loc *Location) (shapeKind, error) {
	switch {
	case loc.KeySymbol == "nwt" && loc.BookNumber > 0:
		return shapeBookChapter, nil
	case loc.KeySymbol != "" && loc.Track > 0:
		return shapePub, nil
	case loc.Type == 3 && loc.DocumentID > 0:
		return shapeDocid, nil
	default:
		return shapePub, fmt.Errorf("unsupported location shape: %+v", *loc)
	}
}

// buildMediaURL builds the publisher's media-API query for a location. Every
// query carries output=json, fileformat=mp4, and the written-language code,
// plus the shape-specific parameters.
func buildMediaURL(base, langCode string, loc *Location) (string, error) {
	shape, err := classifyLocation(loc)
	if err != nil {
		return "", err
	}

	params := url.Values{}
	params.Set("output", "json")
	params.Set("fileformat", "mp4")
	params.Set("langwritten", langCode)

	switch shape {
	case shapePub:
		params.Set("pub", loc.KeySymbol)
		params.Set("track", strconv.FormatInt(loc.Track, 10))
	case shapeBookChapter:
		params.Set("pub", "nwt")
		params.Set("booknum", strconv.FormatInt(loc.BookNumber, 10))
		params.Set("track", strconv.FormatInt(loc.ChapterNumber, 10))
	case shapeDocid:
		params.Set("docid", strconv.FormatInt(loc.DocumentID, 10))
	}

	return base + "?" + params.Encode(), nil
}

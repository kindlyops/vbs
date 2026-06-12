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
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/json"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// fixtureOptions controls how a synthetic playlist export is built so that
// individual sniffer/parser cases can request a deliberately broken file.
// The zero value produces a valid schema-14 export covering every source shape.
type fixtureOptions struct {
	schemaVersion int
	omitTables    []string // tables to drop from the created database
	omitManifest  bool     // build a zip with no manifest.json
	notZip        bool     // write raw bytes instead of a zip archive
	corruptDB     bool     // store non-SQLite bytes where the database belongs
	databaseName  string   // manifest databaseName (defaults to userData.db)
}

const fixtureThumbA = "11111111-1111-1111-1111-111111111111.jpg"
const fixtureThumbBook = "22222222-2222-2222-2222-222222222222.jpg"
const fixtureThumbImage = "33333333-3333-3333-3333-333333333333.jpg"
const fixtureThumbDocid = "44444444-4444-4444-4444-444444444444.jpg"
const fixtureImageFile = "55555555-5555-5555-5555-555555555555.jpg"

// writePlaylistFixture creates a synthetic export on disk and returns its path.
// It never copies any real export; everything is generated.
func writePlaylistFixture(t *testing.T, opts fixtureOptions) string {
	t.Helper()

	dir := t.TempDir()
	out := filepath.Join(dir, "test-export")

	if opts.notZip {
		if err := os.WriteFile(out, []byte("this is not a zip archive"), 0o600); err != nil {
			t.Fatalf("write non-zip fixture: %v", err)
		}
		return out
	}

	dbName := opts.databaseName
	if dbName == "" {
		dbName = "userData.db"
	}

	var dbBytes []byte
	if opts.corruptDB {
		dbBytes = []byte("not a real database")
	} else {
		dbBytes = buildFixtureDB(t, dir, opts.omitTables)
	}

	files := map[string][]byte{
		dbName:            dbBytes,
		fixtureThumbA:     onePixelJPEG(t),
		fixtureThumbBook: onePixelJPEG(t),
		fixtureThumbImage: onePixelJPEG(t),
		fixtureThumbDocid: onePixelJPEG(t),
		fixtureImageFile:  onePixelJPEG(t),
	}

	if !opts.omitManifest {
		files["manifest.json"] = fixtureManifest(t, dbName, opts.schemaVersion)
	}

	writeZip(t, out, files)
	return out
}

// fixtureManifest renders a manifest.json mirroring the observed shape.
func fixtureManifest(t *testing.T, dbName string, schemaVersion int) []byte {
	t.Helper()

	if schemaVersion == 0 {
		schemaVersion = verifiedSchemaVersion
	}

	manifest := map[string]any{
		"version": 1,
		"userDataBackup": map[string]any{
			"schemaVersion": schemaVersion,
			"databaseName":  dbName,
			"deviceName":    "synthetic",
		},
	}

	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	return data
}

// writeZip assembles a zip archive from the given entries.
func writeZip(t *testing.T, path string, files map[string][]byte) {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip create %s: %v", name, err)
		}
		if _, err := w.Write(body); err != nil {
			t.Fatalf("zip write %s: %v", name, err)
		}
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("write zip: %v", err)
	}
}

// onePixelJPEG returns the bytes of a 1x1 JPEG image.
func onePixelJPEG(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return buf.Bytes()
}

// buildFixtureDB creates the SQLite database file and returns its bytes.
func buildFixtureDB(t *testing.T, dir string, omit []string) []byte {
	t.Helper()

	dbPath := filepath.Join(dir, "fixture.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	omitted := make(map[string]bool, len(omit))
	for _, name := range omit {
		omitted[name] = true
	}

	for _, stmt := range fixtureSchema(omitted) {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("exec schema: %v\n%s", err, stmt)
		}
	}
	// Negative fixtures that drop a table only exercise the table check, which
	// runs before parsing, so they need no row data.
	if len(omit) == 0 {
		for _, stmt := range fixtureData() {
			if _, err := db.Exec(stmt); err != nil {
				t.Fatalf("exec data: %v\n%s", err, stmt)
			}
		}
	}

	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	body, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("read db: %v", err)
	}
	return body
}

// fixtureSchema returns CREATE statements for every table except those omitted.
func fixtureSchema(omit map[string]bool) []string {
	tables := map[string]string{
		"Tag":                     `CREATE TABLE Tag (TagId INTEGER PRIMARY KEY, Name TEXT, Type INTEGER)`,
		"TagMap":                  `CREATE TABLE TagMap (TagMapId INTEGER PRIMARY KEY, TagId INTEGER, PlaylistItemId INTEGER, Position INTEGER)`,
		"PlaylistItem":            `CREATE TABLE PlaylistItem (PlaylistItemId INTEGER PRIMARY KEY, Label TEXT, StartTrimOffsetTicks INTEGER, EndTrimOffsetTicks INTEGER, EndAction INTEGER, ThumbnailFilePath TEXT)`,
		"PlaylistItemLocationMap": `CREATE TABLE PlaylistItemLocationMap (PlaylistItemId INTEGER, LocationId INTEGER, MajorMultimediaType INTEGER, BaseDurationTicks INTEGER)`,
		"Location":                `CREATE TABLE Location (LocationId INTEGER PRIMARY KEY, BookNumber INTEGER, ChapterNumber INTEGER, DocumentId INTEGER, Track INTEGER, KeySymbol TEXT, MepsLanguage INTEGER, Type INTEGER)`,
		"IndependentMedia":        `CREATE TABLE IndependentMedia (IndependentMediaId INTEGER PRIMARY KEY, OriginalFilename TEXT, FilePath TEXT, MimeType TEXT, Hash TEXT)`,
		"PlaylistItemIndependentMediaMap": `CREATE TABLE PlaylistItemIndependentMediaMap ` +
			`(PlaylistItemId INTEGER, IndependentMediaId INTEGER, DurationTicks INTEGER)`,
		"PlaylistItemMarker": `CREATE TABLE PlaylistItemMarker (PlaylistItemMarkerId INTEGER PRIMARY KEY, ` +
			`PlaylistItemId INTEGER, Label TEXT, StartTimeTicks INTEGER, DurationTicks INTEGER, ` +
			`EndTransitionDurationTicks INTEGER)`,
	}

	order := []string{
		"Tag", "TagMap", "PlaylistItem", "PlaylistItemLocationMap", "Location",
		"IndependentMedia", "PlaylistItemIndependentMediaMap", "PlaylistItemMarker",
	}

	var stmts []string
	for _, name := range order {
		if omit[name] {
			continue
		}
		stmts = append(stmts, tables[name])
	}
	return stmts
}

// fixtureData returns INSERT statements describing the four-cue playlist:
// a pub/track song, a book/chapter item with two contiguous markers plus one
// separate marker, an image cue, and a docid item.
func fixtureData() []string {
	return []string{
		`INSERT INTO Tag (TagId, Name, Type) VALUES (1, 'synthetic event', 2)`,

		`INSERT INTO PlaylistItem VALUES (1, 'First Clip', 0, 0, 2, '` + fixtureThumbA + `')`,
		`INSERT INTO PlaylistItem VALUES (2, 'Marked Clip', 0, 0, 2, '` + fixtureThumbBook + `')`,
		`INSERT INTO PlaylistItem VALUES (3, 'picture.jpg', 0, 0, 0, '` + fixtureThumbImage + `')`,
		`INSERT INTO PlaylistItem VALUES (4, 'Downloaded Video Clip', 0, 0, 2, '` + fixtureThumbDocid + `')`,

		`INSERT INTO TagMap VALUES (1, 1, 1, 0)`,
		`INSERT INTO TagMap VALUES (2, 1, 2, 1)`,
		`INSERT INTO TagMap VALUES (3, 1, 3, 2)`,
		`INSERT INTO TagMap VALUES (4, 1, 4, 3)`,

		`INSERT INTO Location VALUES (1, NULL, NULL, 1102016935, 135, 'sjj', 420, 0)`,
		`INSERT INTO Location VALUES (2, 23, 5, NULL, NULL, 'nwt', 420, 0)`,
		`INSERT INTO Location VALUES (4, NULL, NULL, 1112024040, 1, NULL, 420, 3)`,

		`INSERT INTO PlaylistItemLocationMap VALUES (1, 1, 2, 1390060000)`,
		`INSERT INTO PlaylistItemLocationMap VALUES (2, 2, 2, 542530000)`,
		`INSERT INTO PlaylistItemLocationMap VALUES (4, 4, 2, 7935420000)`,

		`INSERT INTO IndependentMedia VALUES (1, 'picture.jpg', '` + fixtureImageFile + `', 'image/jpeg', 'abc123')`,
		`INSERT INTO PlaylistItemIndependentMediaMap VALUES (3, 1, 40000000)`,

		`INSERT INTO PlaylistItemMarker VALUES (1, 2, 'Marker one', 20680000, 186510000, 0)`,
		`INSERT INTO PlaylistItemMarker VALUES (2, 2, 'Marker two', 207200000, 356020000, 0)`,
		`INSERT INTO PlaylistItemMarker VALUES (3, 2, 'Marker three', 1257580000, 372370000, 12340000)`,
	}
}

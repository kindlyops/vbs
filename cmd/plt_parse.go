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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite" // registers the pure-Go "sqlite" database/sql driver
)

const sqliteMagic = "SQLite format 3\x00"

var zipMagic = []byte("PK\x03\x04")

// minVerifiedSchemaVersion and maxVerifiedSchemaVersion bound the inclusive
// range of source-app backup schema versions this parser has been validated
// against. Versions within the range are known to be compatible: the newer
// ones only add tables and columns the parser ignores, leaving every column it
// reads unchanged. Versions outside the range warn but still proceed when the
// required tables are present — the tables are the real contract.
const (
	minVerifiedSchemaVersion = 14
	maxVerifiedSchemaVersion = 16
)

// schemaVersionVerified reports whether v falls within the inclusive range of
// schema versions this parser has been validated against.
func schemaVersionVerified(v int) bool {
	return v >= minVerifiedSchemaVersion && v <= maxVerifiedSchemaVersion
}

// requiredTables are the database tables the parser depends on. Their presence
// is the contract that lets us read a playlist regardless of schema version.
var requiredTables = []string{
	"Tag",
	"TagMap",
	"PlaylistItem",
	"PlaylistItemLocationMap",
	"Location",
	"IndependentMedia",
	"PlaylistItemMarker",
}

// Playlist is the ordered, parsed contents of a purple playlist export.
type Playlist struct {
	Name          string
	SchemaVersion int
	DatabaseName  string
	Items         []Item
}

// Item is a single cue in playback order.
type Item struct {
	Position       int
	PlaylistItemID int64
	Label          string
	StartTrimTicks int64
	EndTrimTicks   int64
	EndAction      int
	ThumbnailPath  string
	Location       *Location
	Image          *EmbeddedImage
	Markers        []Marker
}

// IsImage reports whether the item is an image cue (embedded media) rather
// than a reference to published video.
func (i Item) IsImage() bool {
	return i.Image != nil
}

// Location identifies published media referenced from the publisher's catalog.
// Absent numeric columns are represented as zero; KeySymbol is empty when null.
type Location struct {
	MajorMultimediaType int
	BaseDurationTicks   int64
	BookNumber          int64
	ChapterNumber       int64
	DocumentID          int64
	Track               int64
	KeySymbol           string
	MepsLanguage        int
	Type                int
}

// EmbeddedImage is media shipped inside the export zip (an image cue).
type EmbeddedImage struct {
	DurationTicks    int64
	OriginalFilename string
	FilePath         string
	MimeType         string
	Hash             string
}

// Marker is a sub-clip range within a referenced video.
type Marker struct {
	Label                      string
	StartTimeTicks             int64
	DurationTicks              int64
	EndTransitionDurationTicks int64
}

// archive is a sniffed, validated export ready to be parsed. The database has
// been extracted to a temp directory; Close removes it. The original zip path
// is retained so later phases can extract images without re-validating.
type archive struct {
	path          string
	dbName        string
	dbPath        string
	tmpDir        string
	schemaVersion int
}

// Close removes the temp directory holding the extracted database.
func (a *archive) Close() error {
	if a.tmpDir == "" {
		return nil
	}
	return os.RemoveAll(a.tmpDir)
}

// extractEntry writes the named zip entry to destPath. destPath is chosen by
// the caller (derived from item order and slug), never from the entry name, so
// there is no zip-slip exposure; the entry name only locates the source bytes.
func (a *archive) extractEntry(entryName, destPath string) error {
	zr, err := zip.OpenReader(a.path)
	if err != nil {
		return fmt.Errorf("could not reopen archive: %w", err)
	}
	defer func() { _ = zr.Close() }()

	entry := findZipEntry(&zr.Reader, entryName)
	if entry == nil {
		return fmt.Errorf("zip entry %q not found in archive", entryName)
	}

	rc, err := entry.Open()
	if err != nil {
		return fmt.Errorf("could not open zip entry %q: %w", entryName, err)
	}
	defer func() { _ = rc.Close() }()

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("could not create %s: %w", destPath, err)
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, rc); err != nil {
		return fmt.Errorf("could not extract %q to %s: %w", entryName, destPath, err)
	}
	return nil
}

// manifestDoc is the subset of manifest.json the sniffer validates.
type manifestDoc struct {
	UserDataBackup struct {
		SchemaVersion json.Number `json:"schemaVersion"`
		DatabaseName  string      `json:"databaseName"`
	} `json:"userDataBackup"`
}

// sniffPlaylist validates that path is a purple playlist export and extracts
// its database. Each validation failure names what was wrong (see the format
// sniffer spec): zip magic, manifest, SQLite database, required tables.
func sniffPlaylist(path string) (*archive, error) {
	if err := checkZipMagic(path); err != nil {
		return nil, err
	}

	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("not a zip archive: %w", err)
	}
	defer func() { _ = zr.Close() }()

	man, err := readManifest(&zr.Reader)
	if err != nil {
		return nil, err
	}

	dbEntry := findZipEntry(&zr.Reader, man.UserDataBackup.DatabaseName)
	if dbEntry == nil {
		return nil, fmt.Errorf("missing or non-SQLite database: manifest names %q but it is not in the archive",
			man.UserDataBackup.DatabaseName)
	}

	tmpDir, err := os.MkdirTemp("", "vbs-plt-")
	if err != nil {
		return nil, fmt.Errorf("could not create temp dir: %w", err)
	}

	dbPath, err := extractDatabase(dbEntry, tmpDir)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, err
	}

	if err := verifyTables(dbPath); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, err
	}

	schemaVersion, _ := man.UserDataBackup.SchemaVersion.Int64()
	return &archive{
		path:          path,
		dbName:        man.UserDataBackup.DatabaseName,
		dbPath:        dbPath,
		tmpDir:        tmpDir,
		schemaVersion: int(schemaVersion),
	}, nil
}

// parsePlaylist reads the validated archive's database into an ordered model.
func parsePlaylist(a *archive) (*Playlist, error) {
	db, err := sql.Open("sqlite", a.dbPath)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	name, err := queryPlaylistName(db)
	if err != nil {
		return nil, err
	}

	items, index, err := queryItems(db)
	if err != nil {
		return nil, err
	}
	if err := attachLocations(db, items, index); err != nil {
		return nil, err
	}
	if err := attachImages(db, items, index); err != nil {
		return nil, err
	}
	if err := attachMarkers(db, items, index); err != nil {
		return nil, err
	}

	return &Playlist{
		Name:          name,
		SchemaVersion: a.schemaVersion,
		DatabaseName:  a.dbName,
		Items:         items,
	}, nil
}

// queryPlaylistName returns the name of the playlist tag (Type 2).
func queryPlaylistName(db *sql.DB) (string, error) {
	var name string
	err := db.QueryRow(`SELECT Name FROM Tag WHERE Type = 2`).Scan(&name)
	if err != nil {
		return "", fmt.Errorf("could not read playlist name: %w", err)
	}
	return name, nil
}

// queryItems returns the playlist items in playback order plus an index from
// PlaylistItemId to the item's slice position, used to attach related rows.
func queryItems(db *sql.DB) ([]Item, map[int64]int, error) {
	rows, err := db.Query(`
		SELECT tm.Position, pi.PlaylistItemId, pi.Label,
		       pi.StartTrimOffsetTicks, pi.EndTrimOffsetTicks,
		       pi.EndAction, pi.ThumbnailFilePath
		FROM TagMap tm
		JOIN Tag t ON t.TagId = tm.TagId AND t.Type = 2
		JOIN PlaylistItem pi ON pi.PlaylistItemId = tm.PlaylistItemId
		ORDER BY tm.Position`)
	if err != nil {
		return nil, nil, fmt.Errorf("could not read playlist items: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []Item
	index := make(map[int64]int)

	for rows.Next() {
		var (
			it        Item
			startTrim sql.NullInt64
			endTrim   sql.NullInt64
			thumb     sql.NullString
		)
		if err := rows.Scan(&it.Position, &it.PlaylistItemID, &it.Label,
			&startTrim, &endTrim, &it.EndAction, &thumb); err != nil {
			return nil, nil, fmt.Errorf("could not scan playlist item: %w", err)
		}
		it.StartTrimTicks = startTrim.Int64
		it.EndTrimTicks = endTrim.Int64
		it.ThumbnailPath = thumb.String

		index[it.PlaylistItemID] = len(items)
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error reading playlist items: %w", err)
	}
	return items, index, nil
}

// attachLocations attaches published-media references to their items.
func attachLocations(db *sql.DB, items []Item, index map[int64]int) error {
	rows, err := db.Query(`
		SELECT plm.PlaylistItemId, plm.MajorMultimediaType, plm.BaseDurationTicks,
		       l.BookNumber, l.ChapterNumber, l.DocumentId, l.Track,
		       l.KeySymbol, l.MepsLanguage, l.Type
		FROM PlaylistItemLocationMap plm
		JOIN Location l ON l.LocationId = plm.LocationId`)
	if err != nil {
		return fmt.Errorf("could not read locations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			itemID                      int64
			loc                         Location
			book, chapter, docID, track sql.NullInt64
			key                         sql.NullString
		)
		if err := rows.Scan(&itemID, &loc.MajorMultimediaType, &loc.BaseDurationTicks,
			&book, &chapter, &docID, &track, &key, &loc.MepsLanguage, &loc.Type); err != nil {
			return fmt.Errorf("could not scan location: %w", err)
		}
		loc.BookNumber = book.Int64
		loc.ChapterNumber = chapter.Int64
		loc.DocumentID = docID.Int64
		loc.Track = track.Int64
		loc.KeySymbol = key.String

		if pos, ok := index[itemID]; ok {
			locCopy := loc
			items[pos].Location = &locCopy
		}
	}
	return rows.Err()
}

// attachImages attaches embedded image media, marking those items image cues.
func attachImages(db *sql.DB, items []Item, index map[int64]int) error {
	rows, err := db.Query(`
		SELECT pim.PlaylistItemId, pim.DurationTicks,
		       im.OriginalFilename, im.FilePath, im.MimeType, im.Hash
		FROM PlaylistItemIndependentMediaMap pim
		JOIN IndependentMedia im ON im.IndependentMediaId = pim.IndependentMediaId`)
	if err != nil {
		return fmt.Errorf("could not read embedded media: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			itemID int64
			img    EmbeddedImage
		)
		if err := rows.Scan(&itemID, &img.DurationTicks, &img.OriginalFilename,
			&img.FilePath, &img.MimeType, &img.Hash); err != nil {
			return fmt.Errorf("could not scan embedded media: %w", err)
		}
		if pos, ok := index[itemID]; ok {
			imgCopy := img
			items[pos].Image = &imgCopy
		}
	}
	return rows.Err()
}

// attachMarkers attaches segment markers to their items, ordered by start time.
func attachMarkers(db *sql.DB, items []Item, index map[int64]int) error {
	rows, err := db.Query(`
		SELECT PlaylistItemId, Label, StartTimeTicks, DurationTicks,
		       EndTransitionDurationTicks
		FROM PlaylistItemMarker
		ORDER BY PlaylistItemId, StartTimeTicks`)
	if err != nil {
		return fmt.Errorf("could not read markers: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			itemID int64
			m      Marker
		)
		if err := rows.Scan(&itemID, &m.Label, &m.StartTimeTicks,
			&m.DurationTicks, &m.EndTransitionDurationTicks); err != nil {
			return fmt.Errorf("could not scan marker: %w", err)
		}
		if pos, ok := index[itemID]; ok {
			items[pos].Markers = append(items[pos].Markers, m)
		}
	}
	return rows.Err()
}

// checkZipMagic confirms the file begins with the local-file-header signature.
func checkZipMagic(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("could not open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	magic := make([]byte, len(zipMagic))
	if _, err := io.ReadFull(f, magic); err != nil {
		return fmt.Errorf("not a zip archive: %s is too short", path)
	}
	if !bytes.Equal(magic, zipMagic) {
		return fmt.Errorf("not a zip archive: %s does not start with the zip signature", path)
	}
	return nil
}

// readManifest parses manifest.json and validates the fields the parser needs.
func readManifest(zr *zip.Reader) (manifestDoc, error) {
	var doc manifestDoc

	entry := findZipEntry(zr, "manifest.json")
	if entry == nil {
		return doc, fmt.Errorf("missing manifest.json in export")
	}

	rc, err := entry.Open()
	if err != nil {
		return doc, fmt.Errorf("could not open manifest.json: %w", err)
	}
	defer func() { _ = rc.Close() }()

	data, err := io.ReadAll(rc)
	if err != nil {
		return doc, fmt.Errorf("could not read manifest.json: %w", err)
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return doc, fmt.Errorf("manifest.json did not parse: %w", err)
	}
	if doc.UserDataBackup.DatabaseName == "" {
		return doc, fmt.Errorf("manifest missing userDataBackup.databaseName")
	}
	if _, err := doc.UserDataBackup.SchemaVersion.Int64(); err != nil {
		return doc, fmt.Errorf("manifest userDataBackup.schemaVersion is not an integer: %w", err)
	}
	return doc, nil
}

// findZipEntry returns the named entry, or nil when absent.
func findZipEntry(zr *zip.Reader, name string) *zip.File {
	for _, f := range zr.File {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// extractDatabase writes the database entry to tmpDir after confirming the
// SQLite file signature.
func extractDatabase(entry *zip.File, tmpDir string) (string, error) {
	rc, err := entry.Open()
	if err != nil {
		return "", fmt.Errorf("could not open database entry %s: %w", entry.Name, err)
	}
	defer func() { _ = rc.Close() }()

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("could not read database entry %s: %w", entry.Name, err)
	}
	if !bytes.HasPrefix(data, []byte(sqliteMagic)) {
		return "", fmt.Errorf("missing or non-SQLite database: %s does not start with the SQLite signature", entry.Name)
	}

	dbPath := filepath.Join(tmpDir, "userData.db")
	if err := os.WriteFile(dbPath, data, 0o600); err != nil {
		return "", fmt.Errorf("could not write database to temp dir: %w", err)
	}
	return dbPath, nil
}

// verifyTables confirms every required table is present in the database.
func verifyTables(dbPath string) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("could not open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	present := make(map[string]bool)
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table'`)
	if err != nil {
		return fmt.Errorf("could not list database tables: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("could not scan table name: %w", err)
		}
		present[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error reading table list: %w", err)
	}

	var missing []string
	for _, table := range requiredTables {
		if !present[table] {
			missing = append(missing, table)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("database is missing required tables: %s", strings.Join(missing, ", "))
	}
	return nil
}

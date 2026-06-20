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
	"strings"
	"testing"
)

func TestSniffPlaylist_AcceptsValidRenamedFile(t *testing.T) {
	path := writePlaylistFixture(t, fixtureOptions{})

	arc, err := sniffPlaylist(path)
	if err != nil {
		t.Fatalf("expected valid fixture to sniff cleanly, got: %v", err)
	}
	t.Cleanup(func() { _ = arc.Close() })

	if arc.schemaVersion != minVerifiedSchemaVersion {
		t.Errorf("schemaVersion = %d, want %d", arc.schemaVersion, minVerifiedSchemaVersion)
	}
	if arc.dbPath == "" {
		t.Error("expected dbPath to be populated")
	}
}

func TestSniffPlaylist_Rejections(t *testing.T) {
	cases := []struct {
		name        string
		opts        fixtureOptions
		wantInError string
	}{
		{"not a zip", fixtureOptions{notZip: true}, "not a zip archive"},
		{"missing manifest", fixtureOptions{omitManifest: true}, "manifest"},
		{"non-sqlite database", fixtureOptions{corruptDB: true}, "SQLite"},
		{"missing required table", fixtureOptions{omitTables: []string{"PlaylistItemMarker"}}, "PlaylistItemMarker"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writePlaylistFixture(t, tc.opts)

			arc, err := sniffPlaylist(path)
			if arc != nil {
				_ = arc.Close()
			}
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantInError) {
				t.Errorf("error %q does not mention %q", err.Error(), tc.wantInError)
			}
		})
	}
}

func TestSniffPlaylist_UnverifiedSchemaProceeds(t *testing.T) {
	path := writePlaylistFixture(t, fixtureOptions{schemaVersion: 99})

	arc, err := sniffPlaylist(path)
	if err != nil {
		t.Fatalf("unverified schema version should still parse, got: %v", err)
	}
	t.Cleanup(func() { _ = arc.Close() })

	if arc.schemaVersion != 99 {
		t.Errorf("schemaVersion = %d, want 99", arc.schemaVersion)
	}
}

// TestSniffPlaylist_AcceptsSchemaV16 covers the newer verified schema version.
// v16 only adds tables and columns the parser ignores, so the cues it reads are
// unchanged and the export sniffs and parses like any other verified version.
func TestSniffPlaylist_AcceptsSchemaV16(t *testing.T) {
	path := writePlaylistFixture(t, fixtureOptions{schemaVersion: 16})

	arc, err := sniffPlaylist(path)
	if err != nil {
		t.Fatalf("schema v16 should sniff cleanly, got: %v", err)
	}
	t.Cleanup(func() { _ = arc.Close() })

	if arc.schemaVersion != 16 {
		t.Errorf("schemaVersion = %d, want 16", arc.schemaVersion)
	}
	if !schemaVersionVerified(arc.schemaVersion) {
		t.Errorf("schema v16 should be within the verified range %d-%d",
			minVerifiedSchemaVersion, maxVerifiedSchemaVersion)
	}

	pl, err := parsePlaylist(arc)
	if err != nil {
		t.Fatalf("parse schema v16: %v", err)
	}
	if len(pl.Items) != 4 {
		t.Errorf("len(Items) = %d, want 4", len(pl.Items))
	}
}

// TestSchemaVersionVerified pins the inclusive verified range so the warning
// gate trips only for versions outside it.
func TestSchemaVersionVerified(t *testing.T) {
	cases := []struct {
		version int
		want    bool
	}{
		{13, false},
		{14, true},
		{15, true},
		{16, true},
		{17, false},
		{99, false},
	}
	for _, tc := range cases {
		if got := schemaVersionVerified(tc.version); got != tc.want {
			t.Errorf("schemaVersionVerified(%d) = %v, want %v", tc.version, got, tc.want)
		}
	}
}

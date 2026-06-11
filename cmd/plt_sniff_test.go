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

	if arc.schemaVersion != verifiedSchemaVersion {
		t.Errorf("schemaVersion = %d, want %d", arc.schemaVersion, verifiedSchemaVersion)
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

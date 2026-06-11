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
	"net/url"
	"testing"
)

func TestBuildMediaURL(t *testing.T) {
	const base = "https://example.invalid/api"

	cases := []struct {
		name       string
		loc        Location
		wantParams map[string]string
		forbidden  []string
	}{
		{
			name:       "pub track",
			loc:        Location{KeySymbol: "sjj", Track: 135, Type: 0},
			wantParams: map[string]string{"output": "json", "fileformat": "mp4", "langwritten": "ASL", "pub": "sjj", "track": "135"},
			forbidden:  []string{"booknum", "docid"},
		},
		{
			name: "book/chapter",
			loc:  Location{KeySymbol: "nwt", BookNumber: 23, ChapterNumber: 5, Type: 0},
			wantParams: map[string]string{
				"output": "json", "fileformat": "mp4", "langwritten": "ASL",
				"pub": "nwt", "booknum": "23", "track": "5",
			},
			forbidden: []string{"docid"},
		},
		{
			name:       "docid",
			loc:        Location{DocumentID: 1112024040, Track: 1, Type: 3},
			wantParams: map[string]string{"output": "json", "fileformat": "mp4", "langwritten": "ASL", "docid": "1112024040"},
			forbidden:  []string{"pub", "track", "booknum"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildMediaURL(base, "ASL", &tc.loc)
			if err != nil {
				t.Fatalf("buildMediaURL: %v", err)
			}
			parsed, err := url.Parse(got)
			if err != nil {
				t.Fatalf("result is not a URL: %v", err)
			}
			q := parsed.Query()
			for k, want := range tc.wantParams {
				if q.Get(k) != want {
					t.Errorf("param %q = %q, want %q (full: %s)", k, q.Get(k), want, got)
				}
			}
			for _, k := range tc.forbidden {
				if q.Has(k) {
					t.Errorf("query should not have %q: %s", k, got)
				}
			}
		})
	}
}

func TestBuildMediaURL_UnsupportedShape(t *testing.T) {
	_, err := buildMediaURL("https://example.invalid/api", "ASL", &Location{Type: 9})
	if err == nil {
		t.Fatal("expected error for unsupported location shape")
	}
}

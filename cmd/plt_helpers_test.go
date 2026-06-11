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
	"math"
	"testing"
)

func TestTicksToSeconds(t *testing.T) {
	cases := []struct {
		name  string
		ticks int64
		want  float64
	}{
		{"zero", 0, 0},
		{"verified song duration", 1390060000, 139.006},
		{"marker start", 20680000, 2.068},
		{"marker two start", 207200000, 20.72},
		{"image cue 4s", 40000000, 4.0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ticksToSeconds(tc.ticks)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("ticksToSeconds(%d) = %v, want %v", tc.ticks, got, tc.want)
			}
		})
	}
}

func TestResolveLanguage(t *testing.T) {
	cases := []struct {
		name     string
		id       int
		override string
		wantCode string
		wantErr  bool
	}{
		{"known id ASL", 420, "", "ASL", false},
		{"override wins over known", 420, "ESL", "ESL", false},
		{"override fills unknown", 999, "FOO", "FOO", false},
		{"unknown id is fatal", 999, "", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, err := resolveLanguage(tc.id, tc.override)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("resolveLanguage(%d, %q) expected error, got nil", tc.id, tc.override)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveLanguage(%d, %q) unexpected error: %v", tc.id, tc.override, err)
			}
			if code != tc.wantCode {
				t.Errorf("resolveLanguage(%d, %q) = %q, want %q", tc.id, tc.override, code, tc.wantCode)
			}
		})
	}
}

func TestSlugify(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "event Dec 2nd", "event-dec-2nd"},
		{"em dash", "40—Two Part Name", "40-two-part-name"},
		{"curly quotes", "4. “Quoted Heading”", "4-quoted-heading"},
		{"curly apostrophe dropped", "12. It’s a Heading:", "12-its-a-heading"},
		{"colon comma run", "Part 1 Section 5:1, 2", "part-1-section-5-1-2"},
		{"straight apostrophe dropped", "Don't Stop", "dont-stop"},
		{"leading trailing junk", "  --Hello--  ", "hello"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := slugify(tc.in)
			if got != tc.want {
				t.Errorf("slugify(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestUniqueSlug(t *testing.T) {
	seen := map[string]int{}
	got := []string{
		uniqueSlug("talk", seen),
		uniqueSlug("talk", seen),
		uniqueSlug("talk", seen),
		uniqueSlug("other", seen),
	}
	want := []string{"talk", "talk-2", "talk-3", "other"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("uniqueSlug call %d = %q, want %q", i, got[i], want[i])
		}
	}
}

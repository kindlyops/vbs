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

import "testing"

const oneSecondTicks = 10_000_000

func mk(label string, startSec, durSec, transSec float64) Marker {
	return Marker{
		Label:                      label,
		StartTimeTicks:             int64(startSec * oneSecondTicks),
		DurationTicks:              int64(durSec * oneSecondTicks),
		EndTransitionDurationTicks: int64(transSec * oneSecondTicks),
	}
}

func TestMergeMarkers(t *testing.T) {
	cases := []struct {
		name       string
		markers    []Marker
		wantRanges int
		check      func(t *testing.T, ranges []clipRange)
	}{
		{
			name:       "single marker",
			markers:    []Marker{mk("v1", 2.0, 18.0, 0)},
			wantRanges: 1,
			check: func(t *testing.T, ranges []clipRange) {
				if ranges[0].endTicks != int64(20.0*oneSecondTicks) {
					t.Errorf("end = %d, want %d", ranges[0].endTicks, int64(20.0*oneSecondTicks))
				}
			},
		},
		{
			name:       "1ms gap merges adjacent segments",
			markers:    []Marker{mk("v1", 2.068, 18.651, 0), mk("v2", 20.720, 35.602, 0)},
			wantRanges: 1,
			check: func(t *testing.T, ranges []clipRange) {
				if len(ranges[0].markers) != 2 {
					t.Errorf("markers = %v, want both", ranges[0].markers)
				}
			},
		},
		{
			name:       "3s gap splits",
			markers:    []Marker{mk("v1", 2.0, 10.0, 0), mk("v2", 15.0, 10.0, 0)},
			wantRanges: 2,
		},
		{
			name:       "transition extends end to bridge the gap",
			markers:    []Marker{mk("v1", 0.0, 10.0, 0.6), mk("v2", 10.4, 5.0, 0)},
			wantRanges: 1,
		},
		{
			name:       "transition is included in range end",
			markers:    []Marker{mk("v1", 0.0, 10.0, 2.0)},
			wantRanges: 1,
			check: func(t *testing.T, ranges []clipRange) {
				if ranges[0].endTicks != int64(12.0*oneSecondTicks) {
					t.Errorf("end = %d, want %d (incl transition)", ranges[0].endTicks, int64(12.0*oneSecondTicks))
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ranges := mergeMarkers(tc.markers)
			if len(ranges) != tc.wantRanges {
				t.Fatalf("ranges = %d, want %d", len(ranges), tc.wantRanges)
			}
			if tc.check != nil {
				tc.check(t, ranges)
			}
		})
	}
}

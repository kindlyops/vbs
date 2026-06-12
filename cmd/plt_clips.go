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
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// markerMergeGapTicks is the maximum gap between two marker ranges that still
// merges them into one contiguous clip (0.5 s, in 100 ns ticks).
const markerMergeGapTicks = 5_000_000

// clipRange is a contiguous span to cut from a source video, covering one or
// more segment markers. Times are in 100 ns ticks relative to the source.
type clipRange struct {
	startTicks int64
	endTicks   int64
	markers    []Marker
}

// mergeMarkers groups segment markers into contiguous clip ranges. A marker's
// range ends at start + duration + end-transition; two ranges merge when the
// gap between them is at most markerMergeGapTicks. Non-contiguous markers
// become separate ranges (later emitted as lettered sub-clips).
func mergeMarkers(markers []Marker) []clipRange {
	if len(markers) == 0 {
		return nil
	}

	sorted := make([]Marker, len(markers))
	copy(sorted, markers)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].StartTimeTicks < sorted[j].StartTimeTicks
	})

	var ranges []clipRange
	for _, m := range sorted {
		end := m.StartTimeTicks + m.DurationTicks + m.EndTransitionDurationTicks

		if len(ranges) > 0 && m.StartTimeTicks-ranges[len(ranges)-1].endTicks <= markerMergeGapTicks {
			cur := &ranges[len(ranges)-1]
			if end > cur.endTicks {
				cur.endTicks = end
			}
			cur.markers = append(cur.markers, m)
			continue
		}

		ranges = append(ranges, clipRange{
			startTicks: m.StartTimeTicks,
			endTicks:   end,
			markers:    []Marker{m},
		})
	}
	return ranges
}

// cutResult records how a segment was cut, with all times in seconds. The
// clip snaps back to the nearest keyframe at or before the requested start so
// a stream copy stays valid; leadIn is the extra footage before the segment.
type cutResult struct {
	requestedStart float64
	snappedStart   float64
	leadIn         float64
	end            float64
	duration       float64
}

// probeKeyframeBefore returns the presentation time of the last keyframe at or
// before startSec. Published files use a uniform GOP, but this handles any.
func probeKeyframeBefore(file string, startSec float64) (float64, error) {
	from := startSec - 5
	if from < 0 {
		from = 0
	}
	interval := fmt.Sprintf("%.3f%%%.3f", from, startSec+0.1)

	out, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0",
		"-skip_frame", "nokey", "-show_entries", "frame=pts_time", "-of", "csv",
		"-read_intervals", interval, file).Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe keyframes failed for %s: %w", file, err)
	}

	best := 0.0
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Split(line, ",")
		if len(fields) < 2 {
			continue
		}
		pts, err := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64)
		if err != nil {
			continue
		}
		if pts <= startSec && pts > best {
			best = pts
		}
	}
	return best, nil
}

// cutSegment cuts [startSec, endSec] from src into out using a stream copy,
// snapping the start back to the nearest keyframe so no re-encode is needed.
func cutSegment(src, out string, startSec, endSec float64) (cutResult, error) {
	keyframe, err := probeKeyframeBefore(src, startSec)
	if err != nil {
		return cutResult{}, err
	}

	duration := endSec - keyframe
	cmd := exec.Command("ffmpeg", "-loglevel", "error",
		"-ss", fmt.Sprintf("%.3f", keyframe), "-i", src,
		"-t", fmt.Sprintf("%.3f", duration), "-c", "copy",
		"-avoid_negative_ts", "make_zero", "-y", out)
	if combined, err := cmd.CombinedOutput(); err != nil {
		return cutResult{}, fmt.Errorf("ffmpeg cut failed for %s: %s: %w", out, combined, err)
	}

	return cutResult{
		requestedStart: startSec,
		snappedStart:   keyframe,
		leadIn:         startSec - keyframe,
		end:            endSec,
		duration:       duration,
	}, nil
}

// copyFile copies src to dst, creating or truncating dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("could not open %s: %w", src, err)
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("could not create %s: %w", dst, err)
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("could not copy %s to %s: %w", src, dst, err)
	}
	return nil
}

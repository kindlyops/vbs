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
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// requireFFmpeg skips the test when ffmpeg/ffprobe are not installed.
func requireFFmpeg(t *testing.T) {
	t.Helper()
	for _, tool := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not installed; skipping integration test", tool)
		}
	}
}

// makeTestVideo renders a test pattern with a one-second GOP (keyframes at
// every whole second) so keyframe snapping is predictable.
func makeTestVideo(t *testing.T, path string, durationSec int) {
	t.Helper()

	cmd := exec.Command("ffmpeg", "-loglevel", "error", "-f", "lavfi",
		"-i", "testsrc=duration="+strconv.Itoa(durationSec)+":size=320x240:rate=30",
		"-g", "30", "-pix_fmt", "yuv420p", "-y", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("could not render test video: %s: %v", out, err)
	}
}

func ffprobeDuration(t *testing.T, path string) float64 {
	t.Helper()

	out, err := exec.Command("ffprobe", "-v", "error", "-show_entries",
		"format=duration", "-of", "csv=p=0", path).Output()
	if err != nil {
		t.Fatalf("ffprobe duration failed: %v", err)
	}
	d, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		t.Fatalf("could not parse duration %q: %v", out, err)
	}
	return d
}

func TestProbeKeyframeBefore_Integration(t *testing.T) {
	requireFFmpeg(t)

	dir := t.TempDir()
	src := filepath.Join(dir, "src.mp4")
	makeTestVideo(t, src, 10)

	kf, err := probeKeyframeBefore(src, 3.5)
	if err != nil {
		t.Fatalf("probeKeyframeBefore: %v", err)
	}
	if math.Abs(kf-3.0) > 0.05 {
		t.Errorf("keyframe before 3.5s = %v, want ~3.0", kf)
	}
}

func TestCutSegment_Integration(t *testing.T) {
	requireFFmpeg(t)

	dir := t.TempDir()
	src := filepath.Join(dir, "src.mp4")
	out := filepath.Join(dir, "clip.mp4")
	makeTestVideo(t, src, 10)

	// Request 3.5s..6.5s. Snaps back to keyframe 3.0, so the clip runs 3.5s.
	res, err := cutSegment(src, out, 3.5, 6.5)
	if err != nil {
		t.Fatalf("cutSegment: %v", err)
	}
	if math.Abs(res.snappedStart-3.0) > 0.05 {
		t.Errorf("snappedStart = %v, want ~3.0", res.snappedStart)
	}
	if math.Abs(res.leadIn-0.5) > 0.05 {
		t.Errorf("leadIn = %v, want ~0.5", res.leadIn)
	}

	if _, err := os.Stat(out); err != nil {
		t.Fatalf("clip not written: %v", err)
	}
	if got := ffprobeDuration(t, out); math.Abs(got-3.5) > 0.3 {
		t.Errorf("clip duration = %v, want ~3.5", got)
	}
}

func TestArchiveExtractEntry(t *testing.T) {
	path := writePlaylistFixture(t, fixtureOptions{})
	arc, err := sniffPlaylist(path)
	if err != nil {
		t.Fatalf("sniff: %v", err)
	}
	t.Cleanup(func() { _ = arc.Close() })

	dest := filepath.Join(t.TempDir(), "out.jpg")
	if err := arc.extractEntry(fixtureImageFile, dest); err != nil {
		t.Fatalf("extractEntry: %v", err)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read extracted: %v", err)
	}
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		t.Errorf("extracted file is not a JPEG (len %d)", len(data))
	}
}

# plt build image stills Implementation Plan

<!-- Go code blocks keep gofmt tabs and Go line lengths. -->
<!-- markdownlint-configure-file {
  "MD010": { "code_blocks": false },
  "MD013": { "code_blocks": false }
} -->

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** `vbs plt build` encodes every image cue as a 4-second H.264 MP4 in
`clips/`, keeping the original image in `media/`.

**Architecture:** Two small functions in `cmd/plt_clips.go` (`frameSize` maps
`--resolution` to a 16:9 frame; `encodeStill` runs ffmpeg). `imageCue` in
`cmd/plt_build.go` calls them, and the offline `plt cuesheet` command reports
the same clip name and duration. Spec:
`docs/superpowers/specs/2026-10-16-plt-image-stills-design.md`.

**Tech Stack:** Go 1.22, ffmpeg/ffprobe (already required by `plt build`),
libx264.

**Rules for this repo:** no publisher names, domains, or real playlist content in
code, tests, comments, or commit messages (`scripts/check-vendor-neutral.sh`
enforces part of this). Commit subjects are imperative mood, no
conventional-commit prefix. All commands run from the repo root.

---

## File map

- `cmd/plt_clips.go`: add `stillDurationSec`, `frameSize`, `encodeStill`.
- `cmd/plt_clips_test.go`: unit tests for `frameSize`.
- `cmd/plt_clips_integration_test.go`: ffmpeg tests for `encodeStill`.
- `cmd/plt_build.go`: frame size on `buildContext`; new `imageCue`.
- `cmd/plt_build_integration_test.go`: assert the image cue is an MP4 still.
- `cmd/plt_cuesheet.go`: image cue clip name and duration in `cuesheetCues`.
- `cmd/plt_cuesheet_test.go`: assert offline image cues; update
  `sampleManifest`.

---

### Task 1: frameSize

**Files:**

- Modify: `cmd/plt_clips.go`
- Test: `cmd/plt_clips_test.go`

- [ ] **Step 1: Write the failing test**

Append to `cmd/plt_clips_test.go`:

```go
func TestFrameSize(t *testing.T) {
	cases := []struct {
		label         string
		width, height int
	}{
		{"240p", 426, 240},
		{"360p", 640, 360},
		{"480p", 854, 480},
		{"720p", 1280, 720},
		{"1080p", 1920, 1080},
	}
	for _, tc := range cases {
		w, h, err := frameSize(tc.label)
		if err != nil {
			t.Errorf("frameSize(%q): %v", tc.label, err)
			continue
		}
		if w != tc.width || h != tc.height {
			t.Errorf("frameSize(%q) = %dx%d, want %dx%d", tc.label, w, h, tc.width, tc.height)
		}
	}
}

func TestFrameSize_Invalid(t *testing.T) {
	for _, label := range []string{"", "720", "hd", "p", "0p", "-720p", "721p"} {
		if _, _, err := frameSize(label); err == nil {
			t.Errorf("frameSize(%q) should fail", label)
		}
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./cmd -run TestFrameSize` Expected: build failure, `undefined:
frameSize`.

- [ ] **Step 3: Implement**

In `cmd/plt_clips.go`, after `markerMergeGapTicks`, add:

```go
// frameSize maps a resolution label such as "720p" to a 16:9 frame size. The
// width rounds to the nearest even number (480p -> 854x480) because H.264 with
// 4:2:0 chroma needs even dimensions; for the same reason the height must be even.
func frameSize(resolution string) (int, int, error) {
	digits, ok := strings.CutSuffix(resolution, "p")
	height, err := strconv.Atoi(digits)
	if !ok || err != nil || height <= 0 || height%2 != 0 {
		return 0, 0, fmt.Errorf("unsupported resolution %q: want an even height such as 720p", resolution)
	}
	width := 2 * ((height*8 + 4) / 9)
	return width, height, nil
}
```

(`fmt`, `strconv`, `strings` are already imported.)

- [ ] **Step 4: Run it to verify it passes**

Run: `go test ./cmd -run TestFrameSize` Expected: `ok`.

- [ ] **Step 5: Commit**

```bash
git add cmd/plt_clips.go cmd/plt_clips_test.go
git commit -m "Add frameSize to map resolution labels to 16:9 frames"
```

---

### Task 2: encodeStill

**Files:**

- Modify: `cmd/plt_clips.go`
- Test: `cmd/plt_clips_integration_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `cmd/plt_clips_integration_test.go`:

```go
// ffprobeStream returns the first video stream's fields as key=value lines.
func ffprobeStream(t *testing.T, path string) string {
	t.Helper()

	out, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=codec_name,profile,width,height,pix_fmt,r_frame_rate,"+
			"color_range,color_space",
		"-of", "default=noprint_wrappers=1", path).Output()
	if err != nil {
		t.Fatalf("ffprobe stream failed: %v", err)
	}
	return string(out)
}

func TestEncodeStill_Integration(t *testing.T) {
	requireFFmpeg(t)

	dir := t.TempDir()
	src := filepath.Join(dir, "still.jpg")
	out := filepath.Join(dir, "still.mp4")

	// A 4:3 source, so the 16:9 frame needs padding on the sides.
	render := exec.Command("ffmpeg", "-loglevel", "error", "-f", "lavfi",
		"-i", "testsrc=size=640x480", "-frames:v", "1", "-y", src)
	if combined, err := render.CombinedOutput(); err != nil {
		t.Fatalf("could not render test image: %s: %v", combined, err)
	}

	if err := encodeStill(src, out, 1280, 720); err != nil {
		t.Fatalf("encodeStill: %v", err)
	}

	stream := ffprobeStream(t, out)
	for _, want := range []string{"codec_name=h264", "width=1280", "height=720", "pix_fmt=yuv420p"} {
		if !strings.Contains(stream, want) {
			t.Errorf("stream missing %q:\n%s", want, stream)
		}
	}
	if got := ffprobeDuration(t, out); math.Abs(got-stillDurationSec) > 0.05 {
		t.Errorf("still duration = %v, want ~%v", got, stillDurationSec)
	}
}

func TestEncodeStill_MissingSource(t *testing.T) {
	requireFFmpeg(t)

	dir := t.TempDir()
	err := encodeStill(filepath.Join(dir, "missing.jpg"), filepath.Join(dir, "out.mp4"), 1280, 720)
	if err == nil {
		t.Fatal("expected an error for a missing source image")
	}
}
```

(`math`, `os/exec`, `path/filepath`, `strings`, `testing` are already imported.)

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./cmd -run TestEncodeStill` Expected: build failure, `undefined:
encodeStill` and `undefined: stillDurationSec`.

- [ ] **Step 3: Implement**

In `cmd/plt_clips.go`, after `frameSize`, add:

```go
// stillDurationSec is the fixed length of a clip encoded from an image cue.
const stillDurationSec = 4.0

// encodeStill encodes an image as a stillDurationSec H.264 clip of width x
// height. The image is scaled to fit and composited onto black, so transparent
// areas become black. The output is BT.709 TV-range yuv420p at 29.97 fps to
// match the published videos.
func encodeStill(src, out string, width, height int) error {
	filter := fmt.Sprintf(
		"color=c=black:s=%dx%d:r=30000/1001,format=rgb24[bg];", width, height) +
		fmt.Sprintf("[0:v]scale=%d:%d:force_original_aspect_ratio=decrease:", width, height) +
		"force_divisible_by=2,format=rgba[fg];" +
		"[bg][fg]overlay=(W-w)/2:(H-h)/2:shortest=1:format=gbrp," +
		"setsar=1,scale=out_color_matrix=bt709:out_range=tv,format=yuv420p[v]"

	cmd := exec.Command("ffmpeg", "-loglevel", "error",
		"-loop", "1", "-framerate", "30000/1001", "-i", src,
		"-filter_complex", filter, "-map", "[v]",
		"-t", strconv.FormatFloat(stillDurationSec, 'f', -1, 64),
		"-color_range", "tv", "-colorspace", "bt709",
		"-color_primaries", "bt709", "-color_trc", "bt709",
		"-c:v", "libx264", "-profile:v", "main", "-an", "-movflags", "+faststart",
		"-y", out)
	if combined, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg still encode failed for %s -> %s: %s: %w", src, out, combined, err)
	}

	return nil
}
```

JPEG sources are full range; the explicit TV-range conversion and BT.709 tags make
every still match the published videos.

- [ ] **Step 4: Run them to verify they pass**

Run: `go test ./cmd -run TestEncodeStill -v` Expected: both PASS (not SKIP;
ffmpeg is installed).

- [ ] **Step 5: Commit**

```bash
git add cmd/plt_clips.go cmd/plt_clips_integration_test.go
git commit -m "Add encodeStill to render an image as a 4-second clip"
```

---

### Task 3: Build image cues as stills

**Files:**

- Modify: `cmd/plt_build.go` (`buildContext`, `newBuildContext`, `imageCue`)
- Test: `cmd/plt_build_integration_test.go`

- [ ] **Step 1: Write the failing test**

In `cmd/plt_build_integration_test.go`, add `"math"` and `"strings"` to the
imports. In `assertBuildOutputs`, replace

```go
		if c.Kind == "image" {
			image++
		}
```

with

```go
		if c.Kind == "image" {
			image++
			assertStillCue(t, workDir, c)
		}
```

and add after the loop, before `if image != 1 {`:

```go
	assertClipsAreMP4(t, workDir)
```

Then append:

```go
// assertStillCue checks an image cue was encoded to a still clip and that the
// original image was kept in media/.
func assertStillCue(t *testing.T, workDir string, c cue) {
	t.Helper()

	if !strings.HasSuffix(c.Clip, ".mp4") {
		t.Errorf("image cue clip = %q, want an .mp4", c.Clip)
	}
	if c.DurationSec != stillDurationSec {
		t.Errorf("image cue duration = %v, want %v", c.DurationSec, stillDurationSec)
	}
	if !strings.HasPrefix(c.SourceMedia, "media/") {
		t.Errorf("image cue sourceMedia = %q, want a path in media/", c.SourceMedia)
	} else if _, err := os.Stat(filepath.Join(workDir, filepath.FromSlash(c.SourceMedia))); err != nil {
		t.Errorf("original image missing: %v", err)
	}
	if strings.HasSuffix(c.Clip, ".mp4") {
		got := ffprobeDuration(t, filepath.Join(workDir, filepath.FromSlash(c.Clip)))
		if math.Abs(got-stillDurationSec) > 0.05 {
			t.Errorf("still clip duration = %v, want ~%v", got, stillDurationSec)
		}
	}
}

// assertClipsAreMP4 checks clips/ holds only MP4 files.
func assertClipsAreMP4(t *testing.T, workDir string) {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join(workDir, "clips"))
	if err != nil {
		t.Fatalf("read clips/: %v", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".mp4" {
			t.Errorf("clips/ should hold only MP4 files, found %s", e.Name())
		}
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./cmd -run TestBuildPlaylist_EndToEnd` Expected: FAIL. The image
cue clip is `.jpg`, its duration is the playlist's, `sourceMedia` is empty, and
`clips/` holds a non-MP4 file.

- [ ] **Step 3: Implement**

In `cmd/plt_build.go`, add two fields to `buildContext` after `resolution
string`:

```go
	frameWidth  int
	frameHeight int
```

In `newBuildContext`, right after the `resolveLanguage` error check (before the
cache dir and any directories are created), add:

```go
	frameWidth, frameHeight, err := frameSize(pltBuildResolution)
	if err != nil {
		return nil, err
	}
```

and set them in the returned struct after `resolution: pltBuildResolution,`:

```go
		frameWidth:  frameWidth,
		frameHeight: frameHeight,
```

Replace `imageCue` with:

```go
// imageCue extracts an embedded image into media/ and encodes it as a
// fixed-length still clip in clips/.
func (ctx *buildContext) imageCue(item Item, index int, slug, thumb string) ([]cue, error) {
	sourceRel := filepath.Join("media", fmt.Sprintf("%02d-%s%s", index, slug, imageExt(item.Image)))
	if err := ctx.arc.extractEntry(item.Image.FilePath, filepath.Join(ctx.outDir, sourceRel)); err != nil {
		return nil, err
	}

	clipRel := filepath.Join("clips", fmt.Sprintf("%02d-%s.mp4", index, slug))
	err := encodeStill(filepath.Join(ctx.outDir, sourceRel), filepath.Join(ctx.outDir, clipRel),
		ctx.frameWidth, ctx.frameHeight)
	if err != nil {
		return nil, err
	}

	return []cue{{
		Index:        index,
		Label:        item.Label,
		Kind:         "image",
		Clip:         filepath.ToSlash(clipRel),
		SourceMedia:  filepath.ToSlash(sourceRel),
		EndActionRaw: item.EndAction,
		DurationSec:  stillDurationSec,
		Thumbnail:    thumb,
	}}, nil
}
```

An encode failure already gets the item's index and label from the
`fmt.Errorf("item %d (%q): %w", ...)` wrapper in `buildPlaylist`.

- [ ] **Step 4: Run it to verify it passes**

Run: `go test ./cmd -run TestBuildPlaylist_EndToEnd -v` Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/plt_build.go cmd/plt_build_integration_test.go
git commit -m "Encode plt build image cues as 4-second MP4 clips"
```

---

### Task 4: Offline cue sheet matches build

**Files:**

- Modify: `cmd/plt_cuesheet.go` (`cuesheetCues`, image branch)
- Test: `cmd/plt_cuesheet_test.go`

- [ ] **Step 1: Write the failing test**

In `cmd/plt_cuesheet_test.go`, in `TestBuildCueSheetOnly_NoMedia`, after
`assertCuesHaveNoCut(t, manifest.Cues)` add:

```go
	assertImageCuesAreStills(t, manifest.Cues)
```

Append:

```go
// assertImageCuesAreStills checks image cues name the MP4 build would encode.
func assertImageCuesAreStills(t *testing.T, cues []cue) {
	t.Helper()

	found := false
	for _, c := range cues {
		if c.Kind != "image" {
			continue
		}
		found = true
		if !strings.HasSuffix(c.Clip, ".mp4") {
			t.Errorf("image cue clip = %q, want an .mp4", c.Clip)
		}
		if c.DurationSec != stillDurationSec {
			t.Errorf("image cue duration = %v, want %v", c.DurationSec, stillDurationSec)
		}
	}
	if !found {
		t.Error("fixture should contain an image cue")
	}
}
```

In `sampleManifest`, change the image cue to:

```go
			{
				Index: 3, Label: "picture.jpg", Kind: "image",
				Clip: "clips/03-picture.mp4", SourceMedia: "media/03-picture.jpg",
				EndActionRaw: 2, DurationSec: 4.0, Thumbnail: "thumbs/03.jpeg",
			},
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./cmd -run TestBuildCueSheetOnly_NoMedia` Expected: FAIL with
`image cue clip = "clips/NN-....jpg", want an .mp4`.

- [ ] **Step 3: Implement**

In `cmd/plt_cuesheet.go`, in `cuesheetCues`, change the image branch's two
fields to:

```go
			Clip:         fmt.Sprintf("clips/%02d-%s.mp4", index, slug),
```

```go
			DurationSec:  stillDurationSec,
```

- [ ] **Step 4: Run it to verify it passes**

Run: `go test ./cmd` Expected: `ok` (this also covers the cue sheet rendering
tests that use `sampleManifest`).

- [ ] **Step 5: Commit**

```bash
git add cmd/plt_cuesheet.go cmd/plt_cuesheet_test.go
git commit -m "Report image cues as 4-second MP4 clips in plt cuesheet"
```

---

### Task 5: Verify end to end

- [ ] **Step 1: Full package tests, vet, format**

Run: `go test ./cmd && go vet ./cmd && gofmt -l cmd/` Expected: `ok`, no vet
output, no files listed by gofmt.

- [ ] **Step 2: Lint the changed lines**

Run: `golangci-lint run --new-from-rev=origin/main ./cmd/...` Expected: only
findings of kinds the package already has everywhere (`err113`, `exhaustruct`,
`paralleltest`, `goconst`, `varnamelen`). Fix anything else, such as `lll` (120
chars) or `wsl` whitespace.

- [ ] **Step 3: Vendor-neutral guard**

Run: `bash scripts/check-vendor-neutral.sh` Expected: `Vendor-neutrality guard
passed`.

- [ ] **Step 4: Real playlist**

Run (from the repo root, where `test/` holds an untracked playlist with four
images):

```bash
go run . plt build --out test test/*playlist
ls test/*/clips test/*/media
for f in test/*/clips/*.mp4; do ffprobe -v error -show_entries format=duration -of csv=p=0 "$f"; done
```

Expected: `Built 13 cues`. `clips/` holds only `.mp4` files; the four image
clips are ~4.004 s and the original images sit in `media/`. Do not commit
anything under `test/`.

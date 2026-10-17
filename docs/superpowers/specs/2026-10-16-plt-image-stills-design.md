# plt build: encode image cues as 4-second MP4 clips

## Goal

`vbs plt build` writes every cue in `clips/` as an MP4, so a show can be run
from video files alone. Each embedded image in the playlist becomes a
4-second MP4 clip.

## Behavior

For each image item, build:

1. Extracts the original image to `media/NN-slug.<ext>`.
2. Encodes it to `clips/NN-slug.mp4`. `clips/` holds no image files.

The encoded clip:

- **Duration:** exactly 4.0 s, regardless of the playlist duration.
- **Video codec:** H.264, Main profile, `yuv420p`.
- **Frame rate:** 30000/1001.
- **Frame size:** 16:9 at the `--resolution` height (720p → 1280×720).
- **Audio:** none.
- **Aspect:** scaled to fit and padded with black; never cropped or stretched.

The frame size comes from `--resolution`. The height is the number in an
`NNNp` label, so any such label works, including ones with no matching video
rendition (e.g. `1080p`). Width is the 16:9 width rounded to an even number
(e.g. 480p → 854×480). A label that is not of the form `NNNp` is an error
that names the label.

## playlist.json

An image cue keeps `kind: "image"` so stills stay distinguishable from
videos. Its fields:

- `clip`: `clips/NN-slug.mp4`
- `sourceMedia`: `media/NN-slug.<ext>` (the original image)
- `durationSec`: `4`

The cue sheet shows 4.0 s for image cues.

## plt cuesheet (offline)

`plt cuesheet` produces the same `clip` name (`.mp4`) and `durationSec` (4)
for image cues as build, so the two commands stay consistent. It does not
encode anything.

## Components

- `encodeStill(src, out string, width, height int) error` in
  `cmd/plt_clips.go`: runs ffmpeg with `-loop 1 -t 4`, a scale+pad filter,
  and the codec settings above.
- `frameSize(resolution string) (width, height int, err error)` in
  `cmd/plt_clips.go`: maps a resolution label to a 16:9 frame size.
- `imageCue` in `cmd/plt_build.go`: extracts to `media/`, calls
  `encodeStill`, and builds the cue.
- `cuesheetCues` in `cmd/plt_cuesheet.go`: uses the same clip name and
  duration.
- A shared constant for the 4-second still duration, used by both commands.

## Errors

- An ffmpeg failure stops the build. The error includes ffmpeg's output and
  the item's index and label, matching how a failed cut is reported.
- A `--resolution` label that is not of the form `NNNp` fails before any
  encoding.

## Tests

- Unit: `frameSize` for `240p`, `480p` (odd width rounds to even), `720p`,
  `1080p`, and malformed labels (`""`, `"720"`, `"hd"`).
- Unit: `plt cuesheet` gives image cues a `.mp4` clip and 4.0 s duration.
- Integration (ffmpeg, existing `*_integration_test.go` pattern): encode a
  small 4:3 test image at 720p, then check with ffprobe that the result is
  1280×720, 4.0 s, H.264, and `yuv420p`.
- Integration: an existing build integration test with an image item checks
  that `clips/` holds the MP4 and `media/` holds the original image.
- Manual: run build on a real playlist with images and check the output.

## Out of scope

- A flag to change the still duration.
- Audio tracks on still clips.

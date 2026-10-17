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
- **Color:** BT.709 matrix, primaries, and transfer; TV (limited) range.
- **Transparency:** composited onto black, so transparent areas are black.
- **Audio:** none.
- **Aspect:** scaled to fit and centered on a black frame; never cropped or
  stretched.

The frame size comes from `--resolution`. The height is the number in an
`NNNp` label, so any such label works, including ones with no matching video
rendition (e.g. `1080p`). Width is the 16:9 width rounded to an even number
(e.g. 480p → 854×480). Heights must be even (H.264 4:2:0 needs even
dimensions) and at most 4320, so typos fail clearly and the width math cannot
overflow. A malformed label, or a zero, odd, or over-4320 height, is an error
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
  `cmd/plt_clips.go`: runs ffmpeg with `-loop 1 -t 4`. It scales the image
  to fit, overlays it onto a black color source, converts it to BT.709
  TV-range `yuv420p`, and tags it, with the codec settings above.
- `frameSize(resolution string) (width, height int, err error)` in
  `cmd/plt_clips.go`: maps a resolution label to a 16:9 frame size. It also
  enforces `maxFrameHeight` (4320).
- `imageCue` in `cmd/plt_build.go`: extracts to `media/`, calls
  `encodeStill`, and builds the cue.
- `cuesheetCues` in `cmd/plt_cuesheet.go`: uses the same clip name and
  duration.
- A shared constant for the 4-second still duration, used by both commands.

## Errors

- A bad `--resolution` fails before any output directory is created.
- An ffmpeg failure stops the build. The error names the source and output
  and is wrapped with the item's index and label, matching how a failed cut
  is reported.

## Tests

- Unit: `frameSize` accepts `240p`, `360p`, `480p`, `720p`, `1080p`, and
  `4320p`, and rejects `""`, `"720"`, `"hd"`, `"p"`, `"0p"`, `"-720p"`,
  `"721p"`, `"4322p"`, and `"99999p"`.
- Integration (ffmpeg): `encodeStill` on a 4:3 JPEG at 720p. ffprobe checks
  the codec, Main profile, size, `yuv420p`, 30000/1001, TV range, BT.709
  tags, and 4 s duration.
- Integration: a transparent PNG comes out black.
- Integration: a missing source returns an error that names the file.
- Integration: build end to end checks that the image cue is an MP4 still
  with its original in `media/`, and that `clips/` holds only MP4s.
- Integration: a bad resolution leaves the output directory empty.
- Unit: `plt cuesheet` image cues use the exact clip name build produces and
  4 s.
- Manual: run build on a real playlist with images and check the output.

## Out of scope

- A flag to change the still duration.
- Audio tracks on still clips.

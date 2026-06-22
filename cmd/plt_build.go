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
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	"github.com/muesli/coral"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	pltBuildOut        string
	pltBuildLang       string
	pltBuildResolution string
)

var pltBuildCmd = &coral.Command{
	Use:   "build <playlist-file>",
	Short: "Download media and build a play-ready working directory.",
	Long: `Parse a purple playlist, download every referenced video at the chosen
resolution, pre-cut segment clips, and write a self-contained working directory
with ordered clips, a JSON cue sheet, and a Typst cue sheet (compiled to PDF
when typst is installed).`,
	Example: `  vbs plt build meeting.playlist
  vbs plt build --resolution 480p --out ./shows meeting.playlist`,
	Run:  runPltBuild,
	Args: coral.ExactArgs(1),
}

// resolvedMedia is a downloaded, cached rendition for one catalog location.
type resolvedMedia struct {
	cachePath string
	basename  string
	duration  float64
}

// buildContext carries the configuration and paths for one build.
type buildContext struct {
	arc        *archive
	client     *http.Client
	base       string
	langID     int
	langCode   string
	resolution string
	cacheDir   string
	outDir     string
	mediaByURL map[string]resolvedMedia
	copied     map[string]bool
}

func runPltBuild(_ *coral.Command, args []string) {
	requireMediaTools()

	base := viper.GetString("plt.mediaapi")
	if base == "" {
		log.Fatal().Msg("media API endpoint is not configured; set the config key plt.mediaapi (or pass --media-api)")
	}

	arc := openPlaylist(args[0])
	defer func() { _ = arc.Close() }()

	playlist, err := parsePlaylist(arc)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not parse playlist")
	}

	manifest, err := buildPlaylist(arc, playlist, base)
	if err != nil {
		log.Fatal().Err(err).Msg("Build failed")
	}

	log.Info().Msgf("Built %d cues into %s", len(manifest.Cues), filepath.Join(pltBuildOut, manifest.Slug))
}

// requireMediaTools fails fast when ffmpeg or ffprobe are missing.
func requireMediaTools() {
	for _, tool := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(tool); err != nil {
			log.Fatal().Err(err).Msgf("Could not find %s. Please install ffmpeg and ffprobe.", tool)
		}
	}
}

// buildPlaylist runs the whole pipeline and returns the written manifest.
func buildPlaylist(arc *archive, playlist *Playlist, base string) (buildManifest, error) {
	ctx, err := newBuildContext(arc, playlist, base)
	if err != nil {
		return buildManifest{}, err
	}

	var cues []cue
	seen := map[string]int{}
	for i, item := range playlist.Items {
		itemCues, err := ctx.buildItemCues(item, i+1, seen)
		if err != nil {
			return buildManifest{}, fmt.Errorf("item %d (%q): %w", i+1, item.Label, err)
		}
		cues = append(cues, itemCues...)
	}

	manifest := buildManifest{
		Name:       playlist.Name,
		Slug:       slugify(playlist.Name),
		Language:   langInfo{ID: ctx.langID, Code: ctx.langCode},
		Resolution: ctx.resolution,
		BuiltAt:    time.Now().UTC().Format(time.RFC3339),
		Cues:       cues,
	}

	if err := writePlaylistJSON(ctx.outDir, manifest); err != nil {
		return manifest, err
	}
	if pdf, err := writeCueSheet(ctx.outDir, manifest); err != nil {
		return manifest, err
	} else if !pdf {
		log.Info().Msg("typst not found on PATH; wrote cuesheet.typ only (install typst to render cuesheet.pdf)")
	}
	return manifest, nil
}

// newBuildContext resolves the language, creates the working directory layout,
// and locates the shared media cache.
func newBuildContext(arc *archive, playlist *Playlist, base string) (*buildContext, error) {
	langID := playlistLanguageID(playlist)
	langCode, err := resolveLanguage(langID, pltBuildLang)
	if err != nil {
		return nil, err
	}

	userCache, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("could not locate user cache dir: %w", err)
	}
	cacheDir := filepath.Join(userCache, "vbs", "media")

	outDir := filepath.Join(resolveInputPath(pltBuildOut), slugify(playlist.Name))
	for _, sub := range []string{"clips", "media", "thumbs"} {
		if err := os.MkdirAll(filepath.Join(outDir, sub), 0o755); err != nil {
			return nil, fmt.Errorf("could not create %s: %w", sub, err)
		}
	}
	if err := markWorkingDir(outDir); err != nil {
		return nil, err
	}

	return &buildContext{
		arc:        arc,
		client:     http.DefaultClient,
		base:       base,
		langID:     langID,
		langCode:   langCode,
		resolution: pltBuildResolution,
		cacheDir:   cacheDir,
		outDir:     outDir,
		mediaByURL: map[string]resolvedMedia{},
		copied:     map[string]bool{},
	}, nil
}

// markWorkingDir drops a .gitignore that ignores the whole generated working
// directory, so plt output is never accidentally committed when the command is
// run inside a git repository.
func markWorkingDir(outDir string) error {
	path := filepath.Join(outDir, ".gitignore")
	if err := os.WriteFile(path, []byte("# Generated by vbs plt; do not commit.\n*\n"), 0o644); err != nil {
		return fmt.Errorf("could not write %s: %w", path, err)
	}
	return nil
}

// playlistLanguageID returns the MepsLanguage of the first located item, or 0.
func playlistLanguageID(playlist *Playlist) int {
	for _, item := range playlist.Items {
		if item.Location != nil {
			return item.Location.MepsLanguage
		}
	}
	return 0
}

// buildItemCues produces the cues for one playlist item, performing the
// downloads, cuts, copies, and extractions they require.
func (ctx *buildContext) buildItemCues(item Item, index int, seen map[string]int) ([]cue, error) {
	thumb := ctx.extractThumb(item, index)
	slug := uniqueSlug(slugify(item.Label), seen)

	if item.IsImage() {
		return ctx.imageCue(item, index, slug, thumb)
	}
	if item.Location == nil {
		return nil, fmt.Errorf("video item has no catalog location")
	}

	rm, err := ctx.resolveMedia(item.Location)
	if err != nil {
		return nil, err
	}
	sourceRel, err := ctx.ensureMediaCopy(rm)
	if err != nil {
		return nil, err
	}

	ranges := mergeMarkers(item.Markers)
	if len(ranges) == 0 {
		if trimmed, ok := trimRange(item); ok {
			ranges = []clipRange{trimmed}
		} else {
			return ctx.wholeVideoCue(item, index, slug, rm, sourceRel, thumb)
		}
	}
	return ctx.cutCues(item, index, slug, ranges, sourceRel, thumb)
}

// imageCue extracts an embedded image cue from the archive into clips/.
func (ctx *buildContext) imageCue(item Item, index int, slug, thumb string) ([]cue, error) {
	clipRel := filepath.Join("clips", fmt.Sprintf("%02d-%s%s", index, slug, imageExt(item.Image)))
	if err := ctx.arc.extractEntry(item.Image.FilePath, filepath.Join(ctx.outDir, clipRel)); err != nil {
		return nil, err
	}
	return []cue{{
		Index:        index,
		Label:        item.Label,
		Kind:         "image",
		Clip:         filepath.ToSlash(clipRel),
		EndActionRaw: item.EndAction,
		DurationSec:  ticksToSeconds(item.Image.DurationTicks),
		Thumbnail:    thumb,
	}}, nil
}

// wholeVideoCue copies an untrimmed, marker-free video to its ordered clip.
func (ctx *buildContext) wholeVideoCue(
	item Item, index int, slug string, rm resolvedMedia, sourceRel, thumb string,
) ([]cue, error) {
	clipRel := filepath.Join("clips", fmt.Sprintf("%02d-%s.mp4", index, slug))
	if err := copyFile(filepath.Join(ctx.outDir, sourceRel), filepath.Join(ctx.outDir, clipRel)); err != nil {
		return nil, err
	}

	duration := rm.duration
	if duration == 0 {
		duration = ticksToSeconds(item.Location.BaseDurationTicks)
	}
	return []cue{{
		Index:        index,
		Label:        item.Label,
		Kind:         "video",
		Clip:         filepath.ToSlash(clipRel),
		SourceMedia:  filepath.ToSlash(sourceRel),
		EndActionRaw: item.EndAction,
		DurationSec:  duration,
		Thumbnail:    thumb,
	}}, nil
}

// cutCues cuts one clip per range; multiple ranges become lettered sub-clips.
func (ctx *buildContext) cutCues(
	item Item, index int, slug string, ranges []clipRange, sourceRel, thumb string,
) ([]cue, error) {
	srcPath := filepath.Join(ctx.outDir, sourceRel)

	cues := make([]cue, 0, len(ranges))
	for i, r := range ranges {
		suffix := ""
		if len(ranges) > 1 {
			suffix = string(rune('a' + i))
		}
		clipRel := filepath.Join("clips", fmt.Sprintf("%02d%s-%s.mp4", index, suffix, slug))

		res, err := cutSegment(srcPath, filepath.Join(ctx.outDir, clipRel),
			ticksToSeconds(r.startTicks), ticksToSeconds(r.endTicks))
		if err != nil {
			return nil, err
		}

		cues = append(cues, cue{
			Index:        index,
			Label:        item.Label,
			Kind:         "video",
			Clip:         filepath.ToSlash(clipRel),
			SourceMedia:  filepath.ToSlash(sourceRel),
			Markers:      toCueMarkers(r.markers),
			Cut:          &cutInfo{res.requestedStart, res.snappedStart, res.leadIn, res.end, res.duration},
			EndActionRaw: item.EndAction,
			DurationSec:  res.duration,
			Thumbnail:    thumb,
		})
	}
	return cues, nil
}

// resolveMedia downloads and caches the rendition for a location, memoizing by
// query URL so a location referenced by several items is fetched once.
func (ctx *buildContext) resolveMedia(loc *Location) (resolvedMedia, error) {
	if ctx.langCode == "" {
		return resolvedMedia{}, fmt.Errorf(
			"unknown language id %d; set it explicitly with --lang", ctx.langID)
	}

	endpoint, err := buildMediaURL(ctx.base, ctx.langCode, loc)
	if err != nil {
		return resolvedMedia{}, err
	}
	if rm, ok := ctx.mediaByURL[endpoint]; ok {
		return rm, nil
	}

	resp, err := fetchMedia(ctx.client, ctx.base, ctx.langCode, loc)
	if err != nil {
		return resolvedMedia{}, err
	}
	item, fellBack, err := selectRendition(resp, ctx.langCode, ctx.resolution)
	if err != nil {
		return resolvedMedia{}, err
	}
	if fellBack {
		log.Warn().Msgf("rendition %s unavailable for %q; using %s instead", ctx.resolution, item.Title, item.Label)
	}

	cachePath, err := fetchToCache(ctx.client, ctx.cacheDir, item)
	if err != nil {
		return resolvedMedia{}, err
	}

	rm := resolvedMedia{cachePath: cachePath, basename: path.Base(item.File.URL), duration: item.Duration}
	ctx.mediaByURL[endpoint] = rm
	return rm, nil
}

// ensureMediaCopy copies a cached source into the working dir's media/ once,
// returning its path relative to the working directory.
func (ctx *buildContext) ensureMediaCopy(rm resolvedMedia) (string, error) {
	rel := filepath.Join("media", rm.basename)
	if ctx.copied[rm.basename] {
		return rel, nil
	}
	if err := copyFile(rm.cachePath, filepath.Join(ctx.outDir, rel)); err != nil {
		return "", err
	}
	ctx.copied[rm.basename] = true
	return rel, nil
}

// extractThumb extracts an item's thumbnail to thumbs/NN.ext, best effort.
func (ctx *buildContext) extractThumb(item Item, index int) string {
	return extractThumbnail(ctx.arc, item, index, ctx.outDir)
}

// extractThumbnail extracts an item's thumbnail from the archive to
// outDir/thumbs/NN.ext and returns its path relative to outDir (best effort).
func extractThumbnail(arc *archive, item Item, index int, outDir string) string {
	if item.ThumbnailPath == "" {
		return ""
	}
	ext := filepath.Ext(item.ThumbnailPath)
	if ext == "" {
		ext = ".jpg"
	}
	rel := filepath.Join("thumbs", fmt.Sprintf("%02d%s", index, ext))
	if err := arc.extractEntry(item.ThumbnailPath, filepath.Join(outDir, rel)); err != nil {
		log.Warn().Err(err).Msgf("could not extract thumbnail for item %d", index)
		return ""
	}
	return filepath.ToSlash(rel)
}

// toCueMarkers projects domain markers into the manifest's marker shape.
func toCueMarkers(markers []Marker) []cueMarker {
	out := make([]cueMarker, 0, len(markers))
	for _, m := range markers {
		out = append(out, cueMarker{
			Label:    m.Label,
			Start:    ticksToSeconds(m.StartTimeTicks),
			Duration: ticksToSeconds(m.DurationTicks),
		})
	}
	return out
}

// trimRange turns an item's trim offsets into a single clip range, when set.
func trimRange(item Item) (clipRange, bool) {
	if item.StartTrimTicks == 0 && item.EndTrimTicks == 0 {
		return clipRange{}, false
	}
	if item.Location == nil {
		return clipRange{}, false
	}
	return clipRange{
		startTicks: item.StartTrimTicks,
		endTicks:   item.Location.BaseDurationTicks - item.EndTrimTicks,
	}, true
}

// imageExt returns the file extension to use for an embedded image cue.
func imageExt(img *EmbeddedImage) string {
	if ext := filepath.Ext(img.OriginalFilename); ext != "" {
		return ext
	}
	return ".jpg"
}

func init() {
	pltBuildCmd.Flags().StringVar(&pltBuildOut, "out", ".", "directory to create the working directory in")
	pltBuildCmd.Flags().StringVar(&pltBuildLang, "lang", "", "override the written-language code (e.g. ASL)")
	pltBuildCmd.Flags().StringVar(&pltBuildResolution, "resolution", "720p", "preferred rendition")

	var mediaAPI string
	pltBuildCmd.Flags().StringVar(&mediaAPI, "media-api", "", "media API base URL (overrides config key plt.mediaapi)")
	_ = viper.BindPFlag("plt.mediaapi", pltBuildCmd.Flags().Lookup("media-api"))

	pltCmd.AddCommand(pltBuildCmd)
}

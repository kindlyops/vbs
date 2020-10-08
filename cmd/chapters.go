// Copyright © 2018 Kindly Ops, LLC <support@kindlyops.com>
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
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kennygrant/sanitize"
	"github.com/spf13/cobra"
)

var chapterListCmd = &cobra.Command{
	Use:   "chapterlist <videofile.mp4>",
	Short: "List chapters in a video container.",
	Long:  `Use ffprobe to discover all chapter metadata in a video file container.`,
	Run:   chapterList,
	Args:  cobra.ExactArgs(1),
}

var chapterSplitCmd = &cobra.Command{
	Use:   "chaptersplit <videofile.mp4>",
	Short: "Split video file into separate files per chapter.",
	Long:  `Use ffmpeg to copy each chapter from a video file into it's own file.`,
	Run:   chapterSplit,
	Args:  cobra.ExactArgs(1),
}

func chapterSplit(cmd *cobra.Command, args []string) {

	_, err := exec.LookPath("ffprobe")

	if err != nil {
		log.Fatal("Could not find ffprobe. Please install ffmpeg and ffprobe.")
	}

	_, err = exec.LookPath("ffmpeg")

	if err != nil {
		log.Fatal("Could not find ffmpeg. Please install ffmpeg.")
	}

	target := args[0]
	_, err = os.Stat(target)

	if err != nil {
		log.Fatal("Could not access video container ", target)
	}

	data, err := getChapters(target)
	base := strings.Trim(path.Base(target), path.Ext(target))
	targetdir := fmt.Sprintf("split_%s", base)
	err = os.MkdirAll(targetdir, 0777)
	if err != nil {
		log.Fatal(err)
	}
	var wg sync.WaitGroup
	for _, c := range data.Chapters {
		wg.Add(1)
		go copyChapter(&wg, c, target, targetdir)
	}
	wg.Wait()
}

func copyChapter(wg *sync.WaitGroup, c chapter, sourcefile string, targetdir string) error {
	defer wg.Done()
	title := strings.Trim(c.Tags.Title, " \n\r")
	safetitle := sanitize.Name(title)
	prefix := fmt.Sprintf("%03d_", c.Id)
	outfile := filepath.Join(targetdir, prefix+safetitle+path.Ext(sourcefile))
	cmd := exec.Command("ffmpeg",
		"-loglevel", "error",
		"-i", sourcefile,
		"-c", "copy",
		"-map", "0",
		"-ss", c.StartTime,
		"-to", c.EndTime,
		outfile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s: %s\n", outfile, output)
	}
	return err
}

// sample json output
// {
//   "chapters": [
//       {
//           "id": 0,
//           "time_base": "1/1000",
//           "start": 0,
//           "start_time": "0.000000",
//           "end": 6006,
//           "end_time": "6.006000",
//           "tags": {
//               "title": "Title Page\r"
//           }
// 			}
// 	]
// }

type tags struct {
	Title string `json:"title"`
}

type chapter struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Id        int    `json:"id"`
	Tags      tags   `json:"tags"`
}

type ffmprobeResponse struct {
	Chapters []chapter `json:"chapters"`
}

func getChapters(target string) (ffmprobeResponse, error) {
	command := exec.Command("ffprobe",
		"-print_format", "json",
		"-loglevel", "error",
		"-show_chapters",
		"-i", target)

	output, err := command.Output()
	response := ffmprobeResponse{}

	if err != nil {
		return response, err
	}

	err = json.Unmarshal(output, &response)
	return response, err
}

func chapterList(cmd *cobra.Command, args []string) {

	_, err := exec.LookPath("ffprobe")

	if err != nil {
		log.Fatal("Could not find ffprobe. Please install ffmpeg and ffprobe.")
	}

	_, err = exec.LookPath("ffmpeg")

	if err != nil {
		log.Fatal("Could not find ffmpeg. Please install ffmpeg.")
	}

	target := args[0]
	_, err = os.Stat(target)

	if err != nil {
		log.Fatal("Could not access video container ", target)
	}

	data, err := getChapters(target)

	if err != nil {
		log.Fatal("Problem getting chapter data ", err)
	}

	json, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	_, _ = os.Stdout.Write(json)
}

func init() {
	rootCmd.AddCommand(chapterListCmd)
	rootCmd.AddCommand(chapterSplitCmd)
}

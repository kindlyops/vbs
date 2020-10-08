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
	"fmt"

	"github.com/spf13/cobra"
)

// lastUsedCmd represents the lastUsed command
var chapterListCmd = &cobra.Command{
	Use:   "chapterlist",
	Short: "List chapters in a video container.",
	Long:  `Use ffprobe to discover all chapter metadata in a video file container.`,
	Run:   chapterList,
	Args:  cobra.ExactArgs(1),
}

func chapterList(cmd *cobra.Command, args []string) {

	fmt.Printf("This is chapterlist")
}

func init() {
	rootCmd.AddCommand(chapterListCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// dryrunCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:

}

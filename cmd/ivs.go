// Copyright © 2021 Kindly Ops, LLC <support@kindlyops.com>
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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ivs"
	"github.com/spf13/cobra"
)

var ivsOscBridgeCmd = &cobra.Command{
	Use:   "ivs-bridge <ivs-stream-arn>",
	Short: "Connect OSC commands to IVS PutMetadata.",
	Long:  `Use OSC to send messages to IVS using PutMetadata API.`,
	Run:   ivsOscBridge,
	Args:  cobra.ExactArgs(1),
}

var ivsPutMetadataCmd = &cobra.Command{
	Use:   "ivs-put <ivs-stream-arn> <data payload>",
	Short: "Send payload to IVS PutMetadata.",
	Long:  `Send messages to IVS using PutMetadata API.`,
	Run:   ivsPutMetadata,
	Args:  cobra.ExactArgs(2),
}

func ivsOscBridge(cmd *cobra.Command, args []string) {
	arn := args[0]
	fmt.Printf("Got stream arn: '%s'\n", arn)

	// s := session.Must(session.NewSession())
	// svc := ivs.New(mySession)

}

func ivsPutMetadata(cmd *cobra.Command, args []string) {
	arn := args[0]
	data := args[1]
	fmt.Printf("Got stream arn: '%s'\n", arn)

	s := session.Must(session.NewSession())
	svc := ivs.New(s)
	input := &ivs.PutMetadataInput{
		ChannelArn: aws.String(arn),
		Metadata:   aws.String(data),
	}

	_, err := svc.PutMetadata(input)
	if err != nil {
		err = fmt.Errorf("Error from ivs.PutMetadata: %s", err.Error())
	}
}

func init() {
	rootCmd.AddCommand(ivsOscBridgeCmd)
	rootCmd.AddCommand(ivsPutMetadataCmd)
}

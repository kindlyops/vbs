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
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ivs"
	"github.com/spf13/cobra"
	"github.com/hypebeast/go-osc/osc"
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
	addr := "127.0.0.1:"+Port
	if Debug {
		log.Printf("Got stream arn: '%s'\n", arn)
		log.Printf("Listening on port: '%s'\n", addr)
	}
	
	// s := session.Must(session.NewSession())
	// svc := ivs.New(mySession)
	d := osc.NewStandardDispatcher()
	d.AddMsgHandler("/vbs/ivsbridge", func(msg *osc.Message) {
		if Debug {
			osc.PrintMessage(msg)
		}
	})
	
	server := &osc.Server{
		Addr: addr,
		Dispatcher:d,
	}
	server.ListenAndServe()
}

func ivsPutMetadata(cmd *cobra.Command, args []string) {
	arn := args[0]
	data := args[1]
	if Debug {
		log.Printf("Got stream arn: '%s'\n", arn)
		log.Printf("Got data: '%s'\n", data)
	}

	s := session.Must(session.NewSession())
	svc := ivs.New(s)
	input := &ivs.PutMetadataInput{
		ChannelArn: aws.String(arn),
		Metadata:   aws.String(data),
	}

	_, err := svc.PutMetadata(input)
	if err != nil {
		log.Fatalf("Error from ivs.PutMetadata: %s", err.Error())
	}
}

// Port to listen for OSC messages
var Port string

func init() {
	ivsOscBridgeCmd.Flags().StringVarP(&Port, "port", "p", "4427", "Port to listen for OSC")
	rootCmd.AddCommand(ivsOscBridgeCmd)
	rootCmd.AddCommand(ivsPutMetadataCmd)
}

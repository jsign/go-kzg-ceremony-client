package main

import (
	"fmt"
	"log"

	"github.com/jsign/go-kzg-ceremony-client/sequencerclient"
	"github.com/spf13/cobra"
)

var verifyTranscriptCmd = &cobra.Command{
	Use:   "verify-transcript",
	Short: "Pulls and verifies the current sequencer transcript",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := sequencerclient.New()
		if err != nil {
			log.Fatalf("creating sequencer client: %s", err)
		}

		fmt.Printf("Pulling current transcript from sequencer... ")
		transcript, err := client.GetCurrentTranscript(cmd.Context())
		if err != nil {
			log.Fatalf("get sequencer status: %s", err)
		}
		fmt.Printf("OK\n")

		fmt.Printf("Verifying transcript... ")
		if err := transcript.Verify(); err != nil {
			log.Fatalf("verifying transcript: %s", err)
		}
		fmt.Printf("Valid!\n")
	},
}

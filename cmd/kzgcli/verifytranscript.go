package main

import (
	"fmt"
	"log"
	"time"

	"github.com/jsign/go-kzg-ceremony-client/sequencerclient"
	"github.com/spf13/cobra"
)

var verifyTranscriptCmd = &cobra.Command{
	Use:   "verify-transcript",
	Short: "Pulls and verifies the current sequencer transcript",
	Run: func(cmd *cobra.Command, args []string) {
		sequencerURL, err := cmd.Flags().GetString("sequencer-url")
		if err != nil {
			log.Fatalf("get --sequencer-url flag value: %s", err)
		}
		client, err := sequencerclient.New(sequencerURL)
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
		now := time.Now()
		if err := transcript.Verify(); err != nil {
			log.Fatalf("verifying transcript: %s", err)
		}
		fmt.Printf("Valid! (took %.02fs)\n", time.Since(now).Seconds())
	},
}

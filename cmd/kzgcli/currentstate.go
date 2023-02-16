package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jsign/go-kzg-ceremony-client/contribution"
	"github.com/jsign/go-kzg-ceremony-client/sequencerclient"
	"github.com/spf13/cobra"
)

var offlineDownloadStateCmd = &cobra.Command{
	Use:   "download-state <path>",
	Short: "Downloads the current state of the ceremony",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			log.Fatalf("one argument exected")
		}
		client, err := sequencerclient.New()
		if err != nil {
			log.Fatalf("creating sequencer client: %s", err)
		}

		fmt.Printf("Downloading current state... ")
		transcript, err := client.GetCurrentTranscript(cmd.Context())
		if err != nil {
			log.Fatalf("getting current transcript: %s", err)
		}
		fmt.Printf("OK\n")

		bc := contribution.BatchContribution{
			Contributions: make([]contribution.Contribution, len(transcript.Transcripts)),
		}
		for i, transcript := range transcript.Transcripts {
			bc.Contributions[i].NumG1Powers = transcript.NumG1Powers
			bc.Contributions[i].NumG2Powers = transcript.NumG2Powers
			bc.Contributions[i].PowersOfTau = transcript.PowersOfTau
		}

		fmt.Printf("Encoding and saving to %s... ", args[0])
		bytes, err := contribution.Encode(&bc, true)
		if err != nil {
			log.Fatalf("encoding current state to json: %s", err)
		}

		if err := os.WriteFile(args[0], bytes, os.ModePerm); err != nil {
			log.Fatalf("writing current state to file: %s", err)
		}

		fmt.Printf("OK\nSaved current state in %s\n", args[0])
	},
}

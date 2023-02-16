package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/jsign/go-kzg-ceremony-client/contribution"
	"github.com/jsign/go-kzg-ceremony-client/extrand"
	"github.com/spf13/cobra"
)

var offlineContributeCmd = &cobra.Command{
	Use:   "contribute <path-current-state-file> <path-contribution-file>",
	Short: "Opens a file with the current state of the ceremony, makes the contribution, and saves the new state to a file.",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			log.Fatalf("two arguments expected")
		}

		urlrand, err := cmd.Flags().GetString("urlrand")
		if err != nil {
			log.Fatalf("get --urlrand flag value: %s", err)
		}
		var extRandomness [][]byte
		if urlrand != "" {
			fmt.Printf("Pulling entropy from %s... ", urlrand)
			urlBytes, err := extrand.GetFromURL(cmd.Context(), urlrand)
			if err != nil {
				log.Fatalf("get bytes from url: %s", err)
			}
			fmt.Printf("Got it! (length: %d)\n", len(urlBytes))
			extRandomness = append(extRandomness, urlBytes)
		}

		fmt.Printf("Opening and parsing offline current state file...")
		f, err := os.Open(args[0])
		if err != nil {
			log.Fatalf("opening current state file at %s: %s", args[0], err)
		}
		defer f.Close()

		bytes, err := io.ReadAll(f)
		if err != nil {
			log.Fatalf("reading current state file: %s", err)
		}
		contributionBatch, err := contribution.DecodeBatchContribution(bytes)
		if err != nil {
			log.Fatalf("deserializing file content: %s", err)
		}
		fmt.Printf("OK\nCalculating contribution... ")

		if err := contributionBatch.Contribute(extRandomness...); err != nil {
			log.Fatalf("failed on calculating contribution: %s", err)
		}

		nbytes, err := contribution.Encode(contributionBatch, true)
		if err != nil {
			log.Fatalf("encoding contribution: %s", err)
		}

		if err := os.WriteFile(args[1], nbytes, os.ModePerm); err != nil {
			log.Fatalf("writing contribution file to %s: %s", args[1], err)
		}

		fmt.Printf("OK\nSuccess, saved contribution in %s\n", args[1])
	},
}

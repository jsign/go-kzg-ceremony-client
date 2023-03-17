package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jsign/go-kzg-ceremony-client/contribution"
	"github.com/jsign/go-kzg-ceremony-client/sequencerclient"
	"github.com/spf13/cobra"
)

var offlineSendContributionCmd = &cobra.Command{
	Use:   "send-contribution <path-contribution-file>",
	Short: "Sends a previously generated contribution to the sequencer",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			log.Fatalf("one argument expected")
		}

		sessionID, err := cmd.Flags().GetString("session-id")
		if err != nil {
			log.Fatalf("get --session-id flag value: %s", err)
		}
		if sessionID == "" {
			log.Fatalf("the session id can't be empty")
		}

		contributionBytes, err := os.ReadFile(args[0])
		if err != nil {
			log.Fatalf("reading contribution file: %s", err)
		}
		contributionBatch, err := contribution.DecodeBatchContribution(contributionBytes)
		if err != nil {
			log.Fatalf("decoding contribution file: %s", err)
		}

		sequencerURL, err := cmd.Flags().GetString("sequencer-url")
		if err != nil {
			log.Fatalf("get --sequencer-url flag value: %s", err)
		}
		client, err := sequencerclient.New(sequencerURL)
		if err != nil {
			log.Fatalf("creating sequencer client: %s", err)
		}

		for {
			_, ok, err := client.TryContribute(cmd.Context(), sessionID)
			if err != nil {
				fmt.Printf("%v Waiting for our turn failed (err: %s), retrying in %v...\n", time.Now().Format("2006-01-02 15:04:05"), err, tryContributeAttemptDelay)
				time.Sleep(tryContributeAttemptDelay)
				continue
			}
			if !ok {
				fmt.Printf("%v Can't enter the lobby, are you sure we're on your reserved slot? Waiting %v for retrying...\n", time.Now().Format("2006-01-02 15:04:05"), tryContributeAttemptDelay)
				time.Sleep(tryContributeAttemptDelay)
				continue
			}
			break
		}

		fmt.Printf("Sending our precomputed contribution %s to the sequencer...\n", args[0])
		var contributionReceipt *sequencerclient.ContributionReceipt
		for {
			var err error
			contributionReceipt, err = client.Contribute(cmd.Context(), sessionID, contributionBatch)
			if err != nil {
				fmt.Printf("Failed sending contribution!: %s\n", err)
				fmt.Printf("Retrying sending contribution in %v\n", sendContributionRetryDelay)
				time.Sleep(sendContributionRetryDelay)
				continue
			}
			break
		}

		// Persist the receipt and contribution.
		receiptJSON, _ := json.Marshal(contributionReceipt)
		if err := os.WriteFile(fmt.Sprintf("contribution_receipt_%s.json", sessionID), receiptJSON, os.ModePerm); err != nil {
			log.Fatalf("failed to save the contribution receipt (err: %s), printing to stdout as last resort: %s", err, receiptJSON)
		}
		ourContributionBatchJSON, _ := contribution.Encode(contributionBatch, true)
		if err := os.WriteFile(fmt.Sprintf("my_contribution_%s.json", sessionID), ourContributionBatchJSON, os.ModePerm); err != nil {
			log.Fatalf("failed to save the contribution (err: %s), printing to stdout as last resort: %s", err, ourContributionBatchJSON)
		}

		fmt.Printf("Success!\n")
	},
}

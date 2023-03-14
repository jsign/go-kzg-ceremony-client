package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jsign/go-kzg-ceremony-client/contribution"
	"github.com/jsign/go-kzg-ceremony-client/extrand"
	"github.com/jsign/go-kzg-ceremony-client/sequencerclient"
	"github.com/spf13/cobra"
)

const (
	tryContributeAttemptDelay  = time.Second * 30
	sendContributionRetryDelay = time.Second
)

var contributeCmd = &cobra.Command{
	Use:   "contribute",
	Short: "Joins the lobby, waits for a contribution turn, and contributes to the ceremony",
	Run: func(cmd *cobra.Command, args []string) {
		sessionID, err := cmd.Flags().GetString("session-id")
		if err != nil {
			log.Fatalf("get --session-id flag value: %s", err)
		}
		if sessionID == "" {
			log.Fatalf("the session id can't be empty")
		}

		var extRandomness [][]byte
		drand, err := cmd.Flags().GetBool("drand")
		if err != nil {
			log.Fatalf("get --drand flag value: %s", err)
		}
		if drand {
			fmt.Printf("Pulling entropy from drand... ")
			drandBytes, drandRound, err := extrand.GetFromDrand(cmd.Context())
			if err != nil {
				log.Fatalf("get drand bytes: %s", err)
			}
			fmt.Printf("Got it! (length: %d, round: %d)\n", len(drandBytes), drandRound)
			extRandomness = append(extRandomness, drandBytes)
		}

		urlrand, err := cmd.Flags().GetString("urlrand")
		if err != nil {
			log.Fatalf("get --session-id flag value: %s", err)
		}
		if urlrand != "" {
			fmt.Printf("Pulling entropy from %s... ", urlrand)
			urlBytes, err := extrand.GetFromURL(cmd.Context(), urlrand)
			if err != nil {
				log.Fatalf("get bytes from url: %s", err)
			}
			fmt.Printf("Got it! (length: %d)\n", len(urlBytes))
			extRandomness = append(extRandomness, urlBytes)
		}

		sequencerURL, err := cmd.Flags().GetString("sequencer-url")
		if err != nil {
			log.Fatalf("get --sequencer-url flag value: %s", err)
		}

		client, err := sequencerclient.New(sequencerURL)
		if err != nil {
			log.Fatalf("creating sequencer client: %s", err)
		}

		if err := contributeToCeremony(cmd.Context(), client, sessionID, extRandomness); err != nil {
			log.Fatalf("contributing to ceremony: %s", err)
		}
		fmt.Printf("Success!\n")
	},
}

func contributeToCeremony(ctx context.Context, client *sequencerclient.Client, sessionID string, extRandomness [][]byte) error {
	// Enter the lobby and wait for our turn.
	var contributionBatch *contribution.BatchContribution
	for {
		fmt.Printf("Waiting for our turn to contribute...\n")
		cb, ok, err := client.TryContribute(ctx, sessionID)
		if err != nil {
			fmt.Printf("%v Waiting for our turn failed (err: %s), retrying in %v...\n", time.Now().Format("2006-01-02 15:04:05"), err, tryContributeAttemptDelay)
			time.Sleep(tryContributeAttemptDelay)
			continue
		}
		if !ok {
			fmt.Printf("%v Still isn't our turn, waiting %v for retrying...\n", time.Now().Format("2006-01-02 15:04:05"), tryContributeAttemptDelay)
			time.Sleep(tryContributeAttemptDelay)
			continue
		}
		contributionBatch = cb
		break
	}

	// Contribute in our turn.
	fmt.Printf("It's our turn! Contributing...\n")
	now := time.Now()
	if err := contributionBatch.Contribute(extRandomness...); err != nil {
		log.Fatalf("failed on calculating contribution: %s", err)
	}
	fmt.Printf("Contribution ready, took %.02fs\n", time.Since(now).Seconds())

	// Send the contribution to the sequencer.
	var contributionReceipt *sequencerclient.ContributionReceipt
	for {
		var err error
		fmt.Printf("Sending contribution...\n")
		contributionReceipt, err = client.Contribute(ctx, sessionID, contributionBatch)
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

	return nil
}

package main

import (
	"fmt"
	"log"

	"github.com/jsign/go-kzg-ceremony-client/sequencerclient"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Returns the current status of the sequencer",
	Run: func(cmd *cobra.Command, args []string) {
		sequencerURL, err := cmd.Flags().GetString("sequencer-url")
		if err != nil {
			log.Fatalf("get --sequencer-url flag value: %s", err)
		}
		client, err := sequencerclient.New(sequencerURL)
		if err != nil {
			log.Fatalf("creating sequencer client: %s", err)
		}
		status, err := client.GetStatus(cmd.Context())
		if err != nil {
			log.Fatalf("get sequencer status: %s", err)
		}
		fmt.Printf("Lobby size: %d\n", status.LobbySize)
		fmt.Printf("Number of contributions: %d\n", status.NumContributions)
		fmt.Printf("Sequencer address: %s\n", status.SequencerAddress)
	},
}

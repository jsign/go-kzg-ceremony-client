package main

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Returns the current status of the sequencer",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getSequencerClient(cmd)
		if err != nil {
			log.Fatalf("creating sequencer client: %s", err)
		}
		status, err := client.GetStatus(cmd.Context())
		if err != nil {
			log.Fatalf("get sequencer status: %s", err)
		}
		fmt.Printf("LobbySize: %d\n", status.LobbySize)
		fmt.Printf("Number of contributions: %d\n", status.NumContributions)
		fmt.Printf("Sequencer address: %s\n", status.SequencerAddress)
	},
}

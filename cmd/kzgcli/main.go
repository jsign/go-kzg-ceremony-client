package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("root command failed: %s", err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "kzgcli",
	Short: "kzgcli is a Go client for the Powers-Of-Tau ceremony for Ethereum EIP-4844",
	Long: `kzgcli is a Go client for the Powers-Of-Tau ceremony for Ethereum EIP-4844.

You can check the following link to have detailed steps on how to contribute using this client:
https://github.com/jsign/go-kzg-ceremony-client#i-want-to-participate-in-the-ceremony-how-should-i-use-this-client
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Usage(); err != nil {
			log.Fatalf("cmd usage failed: %s", err)
		}
	},
}

var offlineCmd = &cobra.Command{
	Use:   "offline",
	Short: "Contains commands for offline contributions",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Usage(); err != nil {
			log.Fatalf("cmd usage failed: %s", err)
		}
	},
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().String("sequencer-url", "https://seq.ceremony.ethereum.org", "The URL of the ceremony sequencer")

	rootCmd.AddCommand(statusCmd)

	// Online contribution commands.
	contributeCmd.Flags().String("session-id", "", "The sesion id as generated in the 'session_id' field in the authentication process")
	contributeCmd.Flags().Bool("drand", false, "Pull entropy from the Drand network to be mixed with local CSRNG")
	contributeCmd.Flags().String("urlrand", "", "Pull entropy from an HTTP endpoint mixed with local CSRNG")
	rootCmd.AddCommand(contributeCmd)

	// Verification commands.
	rootCmd.AddCommand(verifyTranscriptCmd)

	// Offline commands.
	offlineContributeCmd.Flags().String("urlrand", "", "Pull entropy from an HTTP endpoint mixed with local CSRNG")
	offlineContributeCmd.Flags().String("hex-entropy", "", "Hex encoded entropy to be mixed with local CSRNG")
	offlineSendContributionCmd.Flags().String("session-id", "", "The sesion id as generated in the 'session_id' field in the authentication process")

	rootCmd.AddCommand(offlineCmd)
	offlineCmd.AddCommand(offlineDownloadStateCmd)
	offlineCmd.AddCommand(offlineContributeCmd)
	offlineCmd.AddCommand(offlineSendContributionCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

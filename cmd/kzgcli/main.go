package main

import (
	"fmt"
	"os"

	"github.com/jsign/go-kzg-ceremony-client/sequencerclient"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:   "kzgcli",
	Short: "spotcli is a Go client for the Powers-Of-Tau ceremony for Ethereum EIP-4844.",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

func init() {
	rootCmd.PersistentFlags().Bool("devnet", false, "Use the devnet sequencer")
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.AddCommand(statusCmd)

	contributeCmd.Flags().String("session-id", "", "The sesion id as generated in the 'session_id' field in the authentication process")
	contributeCmd.Flags().Bool("drand", false, "Pull randomness from the Drand network to be mixed with local CSRNG")
	contributeCmd.Flags().String("urlrand", "", "Pull randomness from an HTTP endpoint mixed with local CSRNG")
	rootCmd.AddCommand(contributeCmd)

	rootCmd.AddCommand(verifyTranscriptCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getSequencerClient(cmd *cobra.Command) (*sequencerclient.Client, error) {
	devnet, err := cmd.Flags().GetBool("devnet")
	if err != nil {
		return nil, fmt.Errorf("get --devnet flag: %s", err)
	}
	if devnet {
		return sequencerclient.NewDevnet()
	}
	return sequencerclient.New()
}

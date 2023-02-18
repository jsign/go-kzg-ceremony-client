package sequencerclient

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetStatus(t *testing.T) {
	t.Parallel()

	c := createClient(t)

	status, err := c.GetStatus(context.Background())
	require.NoError(t, err)
	require.GreaterOrEqual(t, status.LobbySize, 0)
	require.Greater(t, status.NumContributions, 0)
	require.NotEmpty(t, status.SequencerAddress)
}

func TestLiveVerification(t *testing.T) {
	t.Parallel()

	client := createClient(t)

	btr, err := client.GetCurrentTranscript(context.Background())
	require.NoError(t, err)

	err = btr.Verify()
	require.NoError(t, err)
}

func createClient(t *testing.T) *Client {
	c, err := New()
	require.NoError(t, err)
	return c
}

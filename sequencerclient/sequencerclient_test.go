package sequencerclient

import (
	"context"
	"os"
	"testing"

	"github.com/jsign/go-kzg-ceremony-client/transcript"
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

func TestVerifyTranscript(t *testing.T) {
	t.Parallel()

	f, err := os.Open("testdata/transcript_20230218_120700.json")
	require.NoError(t, err)
	defer f.Close()

	btr, err := transcript.Decode(f)
	require.NoError(t, err)

	err = btr.Verify()
	require.NoError(t, err)
}

func createClient(t *testing.T) *Client {
	c, err := New("https://seq.ceremony.ethereum.org")
	require.NoError(t, err)
	return c
}

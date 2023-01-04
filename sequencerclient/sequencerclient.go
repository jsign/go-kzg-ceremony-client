package sequencerclient

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/jsign/go-kzg-ceremony-client/contribution"
	"github.com/jsign/go-kzg-ceremony-client/transcript"
)

const sequencerURL = "https://sequencer.ceremony.ethereum.org"

type Client struct {
	sequencerURL string
}

func New() (*Client, error) {
	return &Client{
		sequencerURL: sequencerURL,
	}, nil
}

type CeremonyStatus struct {
	LobbySize        int    `json:"lobby_size"`
	NumContributions int    `json:"num_contributions"`
	SequencerAddress string `json:"sequencer_address"`
}

func (c *Client) GetStatus(ctx context.Context) (CeremonyStatus, error) {
	req, err := http.NewRequest("GET", c.sequencerURL+"/info/status", nil)
	if err != nil {
		return CeremonyStatus{}, fmt.Errorf("creating request: %s", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return CeremonyStatus{}, fmt.Errorf("making request: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return CeremonyStatus{}, fmt.Errorf("received status code %d", res.StatusCode)
	}

	var cs CeremonyStatus
	if err := json.NewDecoder(res.Body).Decode(&cs); err != nil {
		return CeremonyStatus{}, fmt.Errorf("decoding status: %s", err)
	}

	return cs, nil
}

func (c *Client) TryContribute(ctx context.Context, sessionID string) (*contribution.BatchContribution, bool, error) {
	req, err := http.NewRequest("POST", c.sequencerURL+"/lobby/try_contribute", nil)
	if err != nil {
		return nil, false, fmt.Errorf("creating request: %s", err)
	}
	req.Header.Add("Authorization", "Bearer "+sessionID)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("making request: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("received status code %d", res.StatusCode)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, false, fmt.Errorf("reading response: %s", err)
	}

	// Check if we're instructed to retry later.
	type mustWaitResponse struct {
		Error string `json:"error"`
	}
	var mwr mustWaitResponse
	if err := json.Unmarshal(resBody, &mwr); err != nil {
		return nil, false, fmt.Errorf("trying to unmarshal a wait response: %s", err)
	}
	// Should we keep waiting for our turn?
	if mwr.Error != "" {
		fmt.Println(mwr.Error)
		return nil, false, nil
	}

	currentBatch, err := contribution.DecodeBatchContribution(resBody)
	if err != nil {
		return nil, false, fmt.Errorf("validating current contribution file: %s", err)
	}

	return currentBatch, true, nil
}

type ContributionReceipt struct {
	Receipt   string `json:"receipt"`
	Signature string `json:"signature"`
}

func (c *Client) Contribute(ctx context.Context, sessionID string, batch *contribution.BatchContribution) (*ContributionReceipt, error) {
	batchJSON, err := contribution.Encode(batch, false)
	if err != nil {
		return nil, fmt.Errorf("marshaling batch contribution: %s", err)
	}
	req, err := http.NewRequest("POST", c.sequencerURL+"/contribute", bytes.NewReader(batchJSON))
	if err != nil {
		return nil, fmt.Errorf("creating request: %s", err)
	}
	req.Header.Add("Authorization", "Bearer "+sessionID)
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusBadRequest {
		var contributionError struct {
			Code  string `json:"code"`
			Error string `json:"error"`
		}
		_ = json.NewDecoder(res.Body).Decode(&contributionError)
		return nil, fmt.Errorf("contribution error (Code: %s, Error: %s)", contributionError.Code, contributionError.Error)
	}
	if res.StatusCode != http.StatusOK {
		io.Copy(io.Discard, res.Body)
		return nil, fmt.Errorf("unexpected contribution error (status code: %s)", res.Status)
	}

	var receipt ContributionReceipt
	if err := json.NewDecoder(res.Body).Decode(&receipt); err != nil {
		return nil, fmt.Errorf("decoding receipt: %s", err)
	}

	return &receipt, nil
}

func (c *Client) GetCurrentTranscript(ctx context.Context) (*transcript.BatchTranscript, error) {
	req, err := http.NewRequest("GET", c.sequencerURL+"/info/current_state", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received status code %d", res.StatusCode)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %s", err)
	}

	batchTranscript, err := transcript.Decode(resBody)
	if err != nil {
		return nil, fmt.Errorf("validating current contribution file: %s", err)
	}

	return batchTranscript, nil
}

package extrand

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"

	drandclient "github.com/drand/drand/client"
	drandhttp "github.com/drand/drand/client/http"
)

var urls = []string{
	"https://api.drand.sh",
	"https://drand.cloudflare.com",
}

var chainHash, _ = hex.DecodeString("8990e7a9aaed2ffed73dbd7092123d6f289930540d7651336225dc172e51b2ce")

func GetFromDrand(ctx context.Context) ([]byte, uint64, error) {
	drand, err := drandclient.New(drandclient.From(drandhttp.ForURLs(urls, chainHash)...), drandclient.WithChainHash(chainHash))
	if err != nil {
		return nil, 0, fmt.Errorf("creating drand client: %s", err)
	}
	drandres, err := drand.Get(context.Background(), 0)
	if err != nil {
		return nil, 0, fmt.Errorf("get randomness from drand: %s", err)
	}

	return drandres.Randomness(), drandres.Round(), nil
}

func GetFromURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %s", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("doing request: %s", err)
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %s", err)
	}

	return bodyBytes, nil
}

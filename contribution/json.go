package contribution

import (
	"bytes"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"

	bls12381 "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/xeipuuv/gojsonschema"
)

//go:embed contributionSchema.json
var contributionSchemaJSON []byte

type powersOfTauJSON struct {
	G1Powers []string `json:"G1Powers"`
	G2Powers []string `json:"G2Powers"`
}

type contributionJSON struct {
	NumG1Powers int             `json:"numG1Powers"`
	NumG2Powers int             `json:"numG2Powers"`
	PowersOfTau powersOfTauJSON `json:"powersOfTau"`
	PotPubKey   string          `json:"potPubkey"`
}

func DecodeContribution(cJSONBytes []byte) (*Contribution, error) {
	if err := schemaCheck(cJSONBytes, contributionSchemaJSON); err != nil {
		return nil, fmt.Errorf("validating contribution file schema: %s", err)
	}

	var cJSON contributionJSON
	if err := json.Unmarshal(cJSONBytes, &cJSON); err != nil {
		return nil, fmt.Errorf("unmarshaling contribution content: %s", err)
	}

	c, err := cJSON.decode()
	if err != nil {
		return nil, fmt.Errorf("subgroup checks: %s", err)
	}

	return c, nil
}

func Encode(c *Contribution) ([]byte, error) {
	potPubKeyBytes := c.PotPubKey.Bytes()
	potPubKeyHex := "0x" + hex.EncodeToString(potPubKeyBytes[:])
	cJSON := contributionJSON{
		NumG1Powers: c.NumG1Powers,
		NumG2Powers: c.NumG2Powers,
		PotPubKey:   potPubKeyHex,
		PowersOfTau: powersOfTauJSON{
			G1Powers: make([]string, len(c.PowersOfTau.G1Affines)),
			G2Powers: make([]string, len(c.PowersOfTau.G2Affines)),
		},
	}

	for j := range c.PowersOfTau.G1Affines {
		gBytes := c.PowersOfTau.G1Affines[j].Bytes()
		gHex := hex.EncodeToString(gBytes[:])
		cJSON.PowersOfTau.G1Powers[j] = "0x" + gHex
	}
	for j := range c.PowersOfTau.G2Affines {
		gBytes := c.PowersOfTau.G2Affines[j].Bytes()
		gHex := hex.EncodeToString(gBytes[:])
		cJSON.PowersOfTau.G2Powers[j] = "0x" + gHex
	}

	ret, err := json.Marshal(cJSON)
	if err != nil {
		return nil, fmt.Errorf("marshaling contribution: %s", err)
	}
	return ret, nil
}

func (c *contributionJSON) decode() (*Contribution, error) {
	ret := Contribution{}

	potPubKeyBytes, err := hex.DecodeString(c.PotPubKey[2:])
	if err != nil {
		return nil, fmt.Errorf("hex decoding pot pubkey: %s", err)
	}
	decoder := bls12381.NewDecoder(bytes.NewReader(potPubKeyBytes))
	var potPubKey bls12381.G2Affine
	if err := decoder.Decode(&potPubKey); err != nil {
		return nil, fmt.Errorf("decoding public key into G2: %s", err)
	}

	ret = Contribution{
		NumG1Powers: c.NumG1Powers,
		NumG2Powers: c.NumG2Powers,
		PowersOfTau: PowersOfTau{
			G1Affines: make([]bls12381.G1Affine, len(c.PowersOfTau.G1Powers)),
			G2Affines: make([]bls12381.G2Affine, len(c.PowersOfTau.G2Powers)),
		},
		PotPubKey: potPubKey,
	}

	for j, g1Power := range c.PowersOfTau.G1Powers {
		g1PowerBinary, err := hex.DecodeString(g1Power[2:])
		if err != nil {
			return nil, fmt.Errorf("hex decoding %d-th g1 power in contribution: %s", j, err)
		}

		// By default the Decoder *will do* subgroup checking.
		decoder := bls12381.NewDecoder(bytes.NewReader(g1PowerBinary))
		var g1Point bls12381.G1Affine
		if err := decoder.Decode(&g1Point); err != nil {
			return nil, fmt.Errorf("decoding g1 point: %s", err)
		}
		ret.PowersOfTau.G1Affines[j] = g1Point
	}

	for j, g2Power := range c.PowersOfTau.G2Powers {
		g2PowerBinary, err := hex.DecodeString(g2Power[2:])
		if err != nil {
			return nil, fmt.Errorf("hex decoding %d-th g2 power in contribution: %s", j, err)
		}

		// By default the Decoder *will do* subgroup checking.
		decoder := bls12381.NewDecoder(bytes.NewReader(g2PowerBinary))
		var g2Point bls12381.G2Affine
		if err := decoder.Decode(&g2Point); err != nil {
			return nil, fmt.Errorf("decoding g2 point: %s", err)
		}
		ret.PowersOfTau.G2Affines[j] = g2Point
	}

	return &ret, nil
}

func schemaCheck(contribution []byte, schema []byte) error {
	schemaLoader := gojsonschema.NewBytesLoader(contribution)
	documentLoader := gojsonschema.NewBytesLoader(schema)
	res, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("validating json schema: %s", err)
	}
	if !res.Valid() {
		return fmt.Errorf("schema validation failed: %v", res.Errors())
	}

	return nil
}

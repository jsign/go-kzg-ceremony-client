package contribution

import (
	"bytes"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"

	bls12381 "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/xeipuuv/gojsonschema"
	"golang.org/x/sync/errgroup"
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
type batchContributionJSON struct {
	Contributions []contributionJSON `json:"contributions"`
}

func DecodeBatchContribution(bcJSONBytes []byte) (*BatchContribution, error) {
	if err := schemaCheck(bcJSONBytes, contributionSchemaJSON); err != nil {
		return nil, fmt.Errorf("validating contribution file schema: %s", err)
	}

	var bcJSON batchContributionJSON
	if err := json.Unmarshal(bcJSONBytes, &bcJSON); err != nil {
		return nil, fmt.Errorf("unmarshaling contribution content: %s", err)
	}

	bc, err := bcJSON.decode()
	if err != nil {
		return nil, fmt.Errorf("subgroup checks: %s", err)
	}

	return bc, nil
}

func Encode(bc *BatchContribution, pretty bool) ([]byte, error) {
	bcJSON := batchContributionJSON{
		Contributions: make([]contributionJSON, len(bc.Contributions)),
	}
	for i := range bc.Contributions {
		potPubKeyBytes := bc.Contributions[i].PotPubKey.Bytes()
		potPubKeyHex := "0x" + hex.EncodeToString(potPubKeyBytes[:])
		bcJSON.Contributions[i] = contributionJSON{
			NumG1Powers: bc.Contributions[i].NumG1Powers,
			NumG2Powers: bc.Contributions[i].NumG2Powers,
			PotPubKey:   potPubKeyHex,
			PowersOfTau: powersOfTauJSON{
				G1Powers: make([]string, len(bc.Contributions[i].PowersOfTau.G1Affines)),
				G2Powers: make([]string, len(bc.Contributions[i].PowersOfTau.G2Affines)),
			},
		}

		for j := range bc.Contributions[i].PowersOfTau.G1Affines {
			gBytes := bc.Contributions[i].PowersOfTau.G1Affines[j].Bytes()
			gHex := hex.EncodeToString(gBytes[:])
			bcJSON.Contributions[i].PowersOfTau.G1Powers[j] = "0x" + gHex
		}
		for j := range bc.Contributions[i].PowersOfTau.G2Affines {
			gBytes := bc.Contributions[i].PowersOfTau.G2Affines[j].Bytes()
			gHex := hex.EncodeToString(gBytes[:])
			bcJSON.Contributions[i].PowersOfTau.G2Powers[j] = "0x" + gHex
		}
	}

	if pretty {
		ret, err := json.MarshalIndent(bcJSON, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshaling batch contribution: %s", err)
		}
		return ret, nil
	}

	ret, err := json.Marshal(bcJSON)
	if err != nil {
		return nil, fmt.Errorf("marshaling batch contribution: %s", err)
	}
	return ret, nil
}

func (bc *batchContributionJSON) decode() (*BatchContribution, error) {
	ret := BatchContribution{
		Contributions: make([]Contribution, len(bc.Contributions)),
	}

	var group errgroup.Group
	for i, contribution := range bc.Contributions {
		i, contribution := i, contribution
		group.Go(func() error {
			potPubKeyBytes, err := hex.DecodeString(contribution.PotPubKey[2:])
			if err != nil {
				return fmt.Errorf("hex decoding pot pubkey: %s", err)
			}
			decoder := bls12381.NewDecoder(bytes.NewReader(potPubKeyBytes))
			var potPubKey bls12381.G2Affine
			if err := decoder.Decode(&potPubKey); err != nil {
				return fmt.Errorf("decoding public key into G2: %s", err)
			}

			ret.Contributions[i] = Contribution{
				NumG1Powers: contribution.NumG1Powers,
				NumG2Powers: contribution.NumG2Powers,
				PowersOfTau: PowersOfTau{
					G1Affines: make([]bls12381.G1Affine, len(contribution.PowersOfTau.G1Powers)),
					G2Affines: make([]bls12381.G2Affine, len(contribution.PowersOfTau.G2Powers)),
				},
				PotPubKey: potPubKey,
			}

			for j, g1Power := range contribution.PowersOfTau.G1Powers {
				g1PowerBinary, err := hex.DecodeString(g1Power[2:])
				if err != nil {
					return fmt.Errorf("hex decoding %d-th g1 power in %d contribution: %s", j, i, err)
				}

				// By default the Decoder *will do* subgroup checking.
				decoder := bls12381.NewDecoder(bytes.NewReader(g1PowerBinary))
				var g1Point bls12381.G1Affine
				if err := decoder.Decode(&g1Point); err != nil {
					return fmt.Errorf("decoding g1 point: %s", err)
				}
				ret.Contributions[i].PowersOfTau.G1Affines[j] = g1Point
			}
			for j, g2Power := range contribution.PowersOfTau.G2Powers {
				g2PowerBinary, err := hex.DecodeString(g2Power[2:])
				if err != nil {
					return fmt.Errorf("hex decoding %d-th g2 power in %d contribution: %s", j, i, err)
				}

				// By default the Decoder *will do* subgroup checking.
				decoder := bls12381.NewDecoder(bytes.NewReader(g2PowerBinary))
				var g2Point bls12381.G2Affine
				if err := decoder.Decode(&g2Point); err != nil {
					return fmt.Errorf("decoding g2 point: %s", err)
				}
				ret.Contributions[i].PowersOfTau.G2Affines[j] = g2Point
			}
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("decoding contribution :%s", err)
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

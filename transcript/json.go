package transcript

import (
	"bytes"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"

	bls12381 "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/jsign/go-kzg-ceremony-client/contribution"
	"github.com/xeipuuv/gojsonschema"
	"golang.org/x/sync/errgroup"
)

//go:embed transcriptSchema.json
var transcriptSchemaJSON []byte

type powersOfTauJSON struct {
	G1Powers []string `json:"G1Powers"`
	G2Powers []string `json:"G2Powers"`
}

type transcriptJSON struct {
	NumG1Powers int `json:"numG1Powers"`
	NumG2Powers int `json:"numG2Powers"`
	PowersOfTau powersOfTauJSON
	Witness     witnessJSON
}

type witnessJSON struct {
	RunningProducts []string `json:"runningProducts"`
	PotPubKeys      []string `json:"potPubKeys"`
	BLSSignatures   []string `json:"blsSignatures"`
}

type batchTranscriptJSON struct {
	Transcripts                []transcriptJSON `json:"transcripts"`
	ParticipantIDs             []string         `json:"participantIds"`
	ParticipantECDSASignatures []string         `json:"participantEcdsaSignatures"`
}

func Decode(transcriptJSONBytes []byte) (*BatchTranscript, error) {
	if err := schemaCheck(transcriptJSONBytes, transcriptSchemaJSON); err != nil {
		return nil, fmt.Errorf("validating contribution file schema: %s", err)
	}

	var trJSON batchTranscriptJSON
	if err := json.Unmarshal(transcriptJSONBytes, &trJSON); err != nil {
		return nil, fmt.Errorf("unmarshaling contribution content: %s", err)
	}

	bc, err := trJSON.decode()
	if err != nil {
		return nil, fmt.Errorf("subgroup checks: %s", err)
	}

	return bc, nil
}

func (ts *batchTranscriptJSON) decode() (*BatchTranscript, error) {
	ret := BatchTranscript{
		Transcripts:                make([]Transcript, len(ts.Transcripts)),
		ParticipantIDs:             ts.ParticipantIDs,
		ParticipantECDSASignatures: ts.ParticipantECDSASignatures,
	}

	var group errgroup.Group
	for i, transcript := range ts.Transcripts {
		i, transcriptJSON := i, transcript
		group.Go(func() error {
			ret.Transcripts[i] = Transcript{
				NumG1Powers: transcriptJSON.NumG1Powers,
				NumG2Powers: transcriptJSON.NumG2Powers,
				PowersOfTau: contribution.PowersOfTau{
					G1Affines: make([]bls12381.G1Affine, len(transcriptJSON.PowersOfTau.G1Powers)),
					G2Affines: make([]bls12381.G2Affine, len(transcriptJSON.PowersOfTau.G2Powers)),
				},
				Witness: Witness{
					RunningProducts: make([]bls12381.G1Affine, len(transcriptJSON.Witness.RunningProducts)),
					PotPubKeys:      make([]bls12381.G2Affine, len(transcriptJSON.Witness.PotPubKeys)),
					BLSSignatures:   make([]*bls12381.G1Affine, len(transcriptJSON.Witness.BLSSignatures)),
				},
			}

			for j, g1Power := range transcriptJSON.PowersOfTau.G1Powers {
				g1PowerBytes, err := hex.DecodeString(g1Power[2:])
				if err != nil {
					return fmt.Errorf("hex decoding %d-th g1 power in %d contribution: %s", j, i, err)
				}

				// By default the Decoder *will do* subgroup checking.
				decoder := bls12381.NewDecoder(bytes.NewReader(g1PowerBytes))
				var g1Point bls12381.G1Affine
				if err := decoder.Decode(&g1Point); err != nil {
					return fmt.Errorf("decoding g1 point: %s", err)
				}
				ret.Transcripts[i].PowersOfTau.G1Affines[j] = g1Point
			}
			for j, g2Power := range transcriptJSON.PowersOfTau.G2Powers {
				g2PowerBytes, err := hex.DecodeString(g2Power[2:])
				if err != nil {
					return fmt.Errorf("hex decoding %d-th g2 power in %d contribution: %s", j, i, err)
				}

				// By default the Decoder *will do* subgroup checking.
				decoder := bls12381.NewDecoder(bytes.NewReader(g2PowerBytes))
				var g2Point bls12381.G2Affine
				if err := decoder.Decode(&g2Point); err != nil {
					return fmt.Errorf("decoding g2 point: %s", err)
				}
				ret.Transcripts[i].PowersOfTau.G2Affines[j] = g2Point
			}

			for j, runningProduct := range transcriptJSON.Witness.RunningProducts {
				runningProductBytes, err := hex.DecodeString(runningProduct[2:])
				if err != nil {
					return fmt.Errorf("hex decoding %d-th running product in %d transcript: %s", j, i, err)
				}
				decoder := bls12381.NewDecoder(bytes.NewReader(runningProductBytes))
				var g1Point bls12381.G1Affine
				if err := decoder.Decode(&g1Point); err != nil {
					return fmt.Errorf("decoding running product: %s", err)
				}
				ret.Transcripts[i].Witness.RunningProducts[j] = g1Point
			}
			for j, potPubKey := range transcriptJSON.Witness.PotPubKeys {
				potPubKeyBytes, err := hex.DecodeString(potPubKey[2:])
				if err != nil {
					return fmt.Errorf("hex decoding %d-th potpubkey in %d transcript: %s", j, i, err)
				}
				decoder := bls12381.NewDecoder(bytes.NewReader(potPubKeyBytes))
				var g2Point bls12381.G2Affine
				if err := decoder.Decode(&g2Point); err != nil {
					return fmt.Errorf("decoding potpubkey product: %s", err)
				}
				ret.Transcripts[i].Witness.PotPubKeys[j] = g2Point
			}
			for j, blsSignature := range transcriptJSON.Witness.BLSSignatures {
				if blsSignature == "" {
					continue
				}
				blsSignatureBytes, err := hex.DecodeString(blsSignature[2:])
				if err != nil {
					return fmt.Errorf("hex decoding %d-th bls signature in %d transcript: %s", j, i, err)
				}
				decoder := bls12381.NewDecoder(bytes.NewReader(blsSignatureBytes))
				var g1Point bls12381.G1Affine
				if err := decoder.Decode(&g1Point); err != nil {
					return fmt.Errorf("decoding bls signature product: %s", err)
				}
				ret.Transcripts[i].Witness.BLSSignatures[j] = &g1Point
			}

			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("decoding batch transcript :%s", err)
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

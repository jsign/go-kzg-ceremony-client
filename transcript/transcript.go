package transcript

import (
	"fmt"

	bls12381 "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/jsign/go-kzg-ceremony-client/contribution"
	"golang.org/x/sync/errgroup"
)

var (
	g2Generator bls12381.G2Affine
	g1Generator bls12381.G1Affine
)

func init() {
	_, _, g1Generator, g2Generator = bls12381.Generators()
}

type Transcript struct {
	NumG1Powers int
	NumG2Powers int
	PowersOfTau contribution.PowersOfTau
	Witness     Witness
}

type Witness struct {
	RunningProducts []bls12381.G1Affine
	PotPubKeys      []bls12381.G2Affine
	BLSSignatures   []*bls12381.G1Affine
}

type BatchTranscript struct {
	Transcripts                []Transcript
	ParticipantIDs             []string
	ParticipantECDSASignatures []string
}

func (bt *BatchTranscript) Verify() error {
	// 1. `schema_check` was validated when unmarshaling the received JSON.
	// 2.`parameter_check` was validated indirectly since the schema has the expected lengths.
	// 3. `subgroup_checks`` was checked when parsing the JSON, since gnark-crypto does the check when decoding G(1|2) bytes.

	var g errgroup.Group
	for i := range bt.Transcripts {
		i := i
		// 4. `tau_update_check`: check that `runningProducts` is valid by checking the pairings of each element with the
		//    previous one ziped with potPubKeys.
		for j := 0; j < len(bt.Transcripts[i].Witness.RunningProducts)-1; j++ {
			j := j
			g.Go(func() error {
				currRunningProduct := bt.Transcripts[i].Witness.RunningProducts[j]
				nextRunningProduct := bt.Transcripts[i].Witness.RunningProducts[j+1]
				potPubKey := bt.Transcripts[i].Witness.PotPubKeys[j+1]

				left, err := bls12381.Pair([]bls12381.G1Affine{currRunningProduct}, []bls12381.G2Affine{potPubKey})
				if err != nil {
					return fmt.Errorf("pairing current running product with potPubKey: %s", err)
				}
				right, err := bls12381.Pair([]bls12381.G1Affine{nextRunningProduct}, []bls12381.G2Affine{g2Generator})
				if err != nil {
					return fmt.Errorf("pairing next running product with G2: %s", err)
				}

				if !left.Equal(&right) {
					return fmt.Errorf("pairing check failed: %s", err)
				}
				return nil
			})

		}

		// 5. `g1PowersCheck`: checks that the G1 powers in the transcript are coherent powers.
		for j := 1; j < len(bt.Transcripts[i].PowersOfTau.G1Affines)-1; j++ {
			j := j
			g.Go(func() error {
				baseTauG2 := bt.Transcripts[i].PowersOfTau.G2Affines[1]
				currG1 := bt.Transcripts[i].PowersOfTau.G1Affines[j]
				nextG1 := bt.Transcripts[i].PowersOfTau.G1Affines[j+1]

				left, err := bls12381.Pair([]bls12381.G1Affine{currG1}, []bls12381.G2Affine{baseTauG2})
				if err != nil {
					return fmt.Errorf("pairing current G1 with baseTauG2: %s", err)
				}
				right, err := bls12381.Pair([]bls12381.G1Affine{nextG1}, []bls12381.G2Affine{g2Generator})
				if err != nil {
					return fmt.Errorf("pairing next G1 with G2: %s", err)
				}

				if !left.Equal(&right) {
					return fmt.Errorf("pairing check failed: %s", err)
				}
				return nil
			})
		}

		// 6. `g2PowersCheck`: checks that the G2 powers in the transcript are coherent powers.
		for j := 1; j < len(bt.Transcripts[i].PowersOfTau.G2Affines)-1; j++ {
			j := j
			g.Go(func() error {
				baseTauG1 := bt.Transcripts[i].PowersOfTau.G1Affines[1]
				currG2 := bt.Transcripts[i].PowersOfTau.G2Affines[j]
				nextG2 := bt.Transcripts[i].PowersOfTau.G2Affines[j+1]

				left, err := bls12381.Pair([]bls12381.G1Affine{baseTauG1}, []bls12381.G2Affine{currG2})
				if err != nil {
					return fmt.Errorf("pairing baseTauG1 with current G2: %s", err)
				}
				right, err := bls12381.Pair([]bls12381.G1Affine{g1Generator}, []bls12381.G2Affine{nextG2})
				if err != nil {
					return fmt.Errorf("pairing G1 with next G2: %s", err)
				}

				if !left.Equal(&right) {
					return fmt.Errorf("pairing check failed: %s", err)
				}
				return nil
			})
		}
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("verifying sequencer transcript: %s", err)
	}

	return nil
}

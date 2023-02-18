package contribution

import (
	"encoding/hex"
	"fmt"
	"math/big"

	bls12381Fr "github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	"golang.org/x/sync/errgroup"
)

type BatchContribution struct {
	Contributions []Contribution
}

func (bc *BatchContribution) Contribute(extRandomness ...[]byte) error {
	frs := make([]*bls12381Fr.Element, len(bc.Contributions))
	for i := range frs {
		frs[i] = &bls12381Fr.Element{}
		if _, err := frs[i].SetRandom(); err != nil {
			return fmt.Errorf("get random Fr: %s", err)
		}

		// For every externally provided randomness (if any), we multiply frs[i] (generated with CSRNG),
		// with a proper map from the []bytes->Fr.
		for _, externalRandomness := range extRandomness {
			extFr := &bls12381Fr.Element{}

			// SetBytes() is safe to call with an arbitrary sized []byte since:
			// - A big.Int `r` will be created from the bytes, which is an arbitrary precision integer.
			// - gnark-crypto will automatically make r.Mod(BLS_MODULUS) to have a proper Fr.
			extFr.SetBytes(externalRandomness)

			frs[i].Mul(frs[i], extFr)
		}
	}

	return bc.contributeWithFrs(frs)
}

func (bc *BatchContribution) contributeWithSecrets(secrets []string) error {
	frs := make([]*bls12381Fr.Element, len(secrets))
	for i := range secrets {
		secretBytes, err := hex.DecodeString(secrets[i][2:])
		if err != nil {
			return err
		}
		frs[i] = &bls12381Fr.Element{}
		frs[i].SetBytes(secretBytes)
		if err != nil {
			return fmt.Errorf("set Fr from bytes: %s", err)
		}
	}

	return bc.contributeWithFrs(frs)
}

func (bc *BatchContribution) contributeWithFrs(frs []*bls12381Fr.Element) error {
	var g errgroup.Group

	for i := range bc.Contributions {
		g.Go(func(i int, contribution *Contribution) func() error {
			return func() error {
				xBig := big.NewInt(0)
				frs[i].BigInt(xBig)

				contribution.updatePowersOfTau(xBig)
				contribution.updateWitness(xBig)

				// Cleanup in-memory secret.
				xBig.SetInt64(0)
				*frs[i] = bls12381Fr.Element{}

				return nil
			}
		}(i, &bc.Contributions[i]))
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("contributing to sub-ceremony: %s", err)
	}

	return nil
}

func (bc *BatchContribution) Verify(prevBatchContribution *BatchContribution) (bool, error) {
	for i, contribution := range prevBatchContribution.Contributions {
		ok, err := bc.Contributions[i].Verify(&contribution)
		if err != nil {
			return false, fmt.Errorf("verifying %d-th contribution: %s", i, err)
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

package contribution

import (
	"encoding/hex"
	"fmt"
	"math/big"

	bls12381 "github.com/consensys/gnark-crypto/ecc/bls12-381"
	bls12381Fr "github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
)

var g2Generator bls12381.G2Affine

func init() {
	_, _, _, g2Generator = bls12381.Generators()
}

type PowersOfTau struct {
	G1Affines []bls12381.G1Affine
	G2Affines []bls12381.G2Affine
}

type Contribution struct {
	NumG1Powers int
	NumG2Powers int
	PowersOfTau PowersOfTau
	PotPubKey   bls12381.G2Affine
}

func (c *Contribution) Contribute(extRandomness ...[]byte) error {
	frs := &bls12381Fr.Element{}
	if _, err := frs.SetRandom(); err != nil {
		return fmt.Errorf("get random Fr: %s", err)
	}

	// For every externally provided randomness (if any), we multiply frs (generated with CSRNG),
	// with a proper map from the []bytes->Fr.
	for _, externalRandomness := range extRandomness {
		extFr := &bls12381Fr.Element{}

		// SetBytes() is safe to call with an arbitrary sized []byte since:
		// - A big.Int `r` will be created from the bytes, which is an arbitrary precision integer.
		// - gnark-crypto will automatically make r.Mod(BLS_MODULUS) to have a proper Fr.
		extFr.SetBytes(externalRandomness)

		frs.Mul(frs, extFr)
	}

	return c.contributeWithFr(frs)
}

func (c *Contribution) contributeWithSecret(secret string) error {
	secretBytes, err := hex.DecodeString(secret[2:])
	if err != nil {
		return err
	}
	fr := &bls12381Fr.Element{}
	fr.SetBytes(secretBytes)
	if err != nil {
		return fmt.Errorf("set Fr from bytes: %s", err)
	}

	return c.contributeWithFr(fr)
}

func (c *Contribution) contributeWithFr(fr *bls12381Fr.Element) error {
	xBig := big.NewInt(0)

	fr.BigInt(xBig)

	c.updatePowersOfTau(xBig)
	c.updateWitness(xBig)

	// Cleanup in-memory secret.
	xBig.SetInt64(0)
	*fr = bls12381Fr.Element{}

	return nil
}

func (c *Contribution) Verify(previousContribution *Contribution) (bool, error) {
	//  Check that the updated G1Affine[1] complies with `PotPubKey`.
	//  That is,  newG1Affine[1] = x*oldG1Affine[1].
	//  We do this with the pairing check `e(oldG1Affine[1], PotPubKey) =?= e(newG1Affine[1], g2)`
	prevG1Power := previousContribution.PowersOfTau.G1Affines[1]
	left, err := bls12381.Pair([]bls12381.G1Affine{prevG1Power}, []bls12381.G2Affine{c.PotPubKey})
	if err != nil {
		return false, fmt.Errorf("pairing prevG1Power with PotPubKey: %s", err)
	}

	postG1Power := c.PowersOfTau.G1Affines[1]
	right, err := bls12381.Pair([]bls12381.G1Affine{postG1Power}, []bls12381.G2Affine{g2Generator})
	if err != nil {
		return false, fmt.Errorf("pairing postG1Power with G2: %s", err)
	}

	// If the pairing doesn't match, return `false`.
	if !left.Equal(&right) {
		return false, nil
	}

	// All validations are good, return `true`.
	return true, nil
}

func (c *Contribution) updatePowersOfTau(x *big.Int) {
	xi := big.NewInt(1)

	for i := 0; i < c.NumG1Powers; i++ {
		c.PowersOfTau.G1Affines[i].ScalarMultiplication(&c.PowersOfTau.G1Affines[i], xi)

		if i < c.NumG2Powers {
			c.PowersOfTau.G2Affines[i].ScalarMultiplication(&c.PowersOfTau.G2Affines[i], xi)
		}
		xi.Mul(xi, x).Mod(xi, bls12381Fr.Modulus())
	}
}

func (c *Contribution) updateWitness(x *big.Int) {
	c.PotPubKey.ScalarMultiplication(&g2Generator, x)
}

package contribution

import (
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

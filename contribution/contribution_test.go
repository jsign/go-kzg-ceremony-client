package contribution

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/savid/go-kzg-ceremony-client/extrand"
	"github.com/stretchr/testify/require"
)

func TestWithInitialContributionFile(t *testing.T) {
	t.Parallel()

	contributionFile, err := os.ReadFile("testdata/initialContribution4096.json")
	require.NoError(t, err)

	c, err := DecodeContribution(contributionFile)
	require.NoError(t, err)

	err = c.Contribute()
	require.NoError(t, err)

	originalc, err := DecodeContribution(contributionFile)
	require.NoError(t, err)
	ok, err := c.Verify(originalc)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestReferenceImplementationTestVector(t *testing.T) {
	t.Parallel()

	subTests := []struct {
		name        string
		secret      string
		initialFile string
		updatedFile string
		mustFail    bool
	}{
		{
			name: "success 4096",
			// Use the same secrets for sub-ceremonies as in the test vector.
			// See github.com/jsign/kzg-ceremony-test-vectors
			secret:      "0x111100",
			initialFile: "testdata/initialContribution4096.json",
			updatedFile: "testdata/updatedContribution4096.json",
		},
		{
			name: "success 8192",
			// Use the same secrets for sub-ceremonies as in the test vector.
			// See github.com/jsign/kzg-ceremony-test-vectors
			secret:      "0x221100",
			initialFile: "testdata/initialContribution8192.json",
			updatedFile: "testdata/updatedContribution8192.json",
		},
		{
			name: "success 16384",
			// Use the same secrets for sub-ceremonies as in the test vector.
			// See github.com/jsign/kzg-ceremony-test-vectors
			secret:      "0x331100",
			initialFile: "testdata/initialContribution16384.json",
			updatedFile: "testdata/updatedContribution16384.json",
		},
		{
			name: "success 32768",
			// Use the same secrets for sub-ceremonies as in the test vector.
			// See github.com/jsign/kzg-ceremony-test-vectors
			secret:      "0x441100",
			initialFile: "testdata/initialContribution32768.json",
			updatedFile: "testdata/updatedContribution32768.json",
		},
		{
			name: "must fail 4096",
			// Use slightly modified secrets. This should cause a failure!
			// PotPubkey and the powers of Tau will mismatch!
			secret:      "0x111101",
			initialFile: "testdata/initialContribution4096.json",
			updatedFile: "testdata/updatedContribution4096.json",
			mustFail:    true,
		},
		{
			name: "must fail 8192",
			// Use slightly modified secrets. This should cause a failure!
			// PotPubkey and the powers of Tau will mismatch!
			secret:      "0x221101",
			initialFile: "testdata/initialContribution8192.json",
			updatedFile: "testdata/updatedContribution8192.json",
			mustFail:    true,
		},
		{
			name: "must fail 16384",
			// Use slightly modified secrets. This should cause a failure!
			// PotPubkey and the powers of Tau will mismatch!
			secret:      "0x331101",
			initialFile: "testdata/initialContribution16384.json",
			updatedFile: "testdata/updatedContribution16384.json",
			mustFail:    true,
		},
		{
			name: "must fail 32768",
			// Use slightly modified secrets. This should cause a failure!
			// PotPubkey and the powers of Tau will mismatch!
			secret:      "0x441101",
			initialFile: "testdata/initialContribution32768.json",
			updatedFile: "testdata/updatedContribution32768.json",
			mustFail:    true,
		},
	}

	for _, subTest := range subTests {
		subTest := subTest
		t.Run(subTest.name, func(t *testing.T) {
			t.Parallel()

			// Start from the initialContribution.json defined in the spec.
			contributionFile, err := os.ReadFile(subTest.initialFile)
			require.NoError(t, err)

			// Parse it.
			c, err := DecodeContribution(contributionFile)
			require.NoError(t, err)

			// Calculate the new batch contribution using **our** client logic.
			err = c.contributeWithSecret(subTest.secret)
			require.NoError(t, err)

			// In `bc` we have **our** calculated BatchContribution for the defined secrets.
			// Now we have to compare it against the test-vector result.

			// Pull the expected batch contribution test-vector.
			expectedContributionFile, err := os.ReadFile(subTest.updatedFile)
			require.NoError(t, err)

			// Parse it.
			expectedC, err := DecodeContribution(expectedContributionFile)
			require.NoError(t, err)

			// Now we'll compare `bc` (our batch contribution) with `expectedBc` (test-vector result).

			// check that:
			// - NumG1Powers and NumG2Powers match.
			// - PotPubkey match (or not, if we're expecting a failure).
			// - Calculated G1Affines and G2Affines from PowersOfTau match (or not, if we're expecting a failure).
			require.Equal(t, expectedC.NumG1Powers, c.NumG1Powers)
			require.Equal(t, expectedC.NumG2Powers, c.NumG2Powers)

			if subTest.mustFail {
				require.False(t, expectedC.PotPubKey.Equal(&c.PotPubKey))
			} else {
				require.True(t, expectedC.PotPubKey.Equal(&c.PotPubKey))
			}

			for j := range expectedC.PowersOfTau.G1Affines {
				expG1 := expectedC.PowersOfTau.G1Affines[j]
				gotG1 := c.PowersOfTau.G1Affines[j]

				// Note that for j==0 we have to make an exception, since G1[0] is always the generator
				// independently from the secret!.
				if j != 0 && subTest.mustFail {
					require.False(t, expG1.Equal(&gotG1))
				} else {
					require.True(t, expG1.Equal(&gotG1))
				}
			}
			for j := range expectedC.PowersOfTau.G2Affines {
				expG2 := expectedC.PowersOfTau.G2Affines[j]
				gotG2 := c.PowersOfTau.G2Affines[j]
				if j != 0 && subTest.mustFail {
					require.False(t, expG2.Equal(&gotG2))
				} else {
					require.True(t, expG2.Equal(&gotG2))
				}
			}
		})
	}
}

func TestExternalRandomness(t *testing.T) {
	t.Parallel()

	contributionFile, err := os.ReadFile("testdata/initialContribution4096.json")
	require.NoError(t, err)

	c, err := DecodeContribution(contributionFile)
	require.NoError(t, err)

	// Add drand randomness.
	drandres, round, err := extrand.GetFromDrand(context.Background())
	require.NoError(t, err)
	require.NotZero(t, round)
	require.NotEmpty(t, drandres)

	// Add external URL randomness
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("imexternalrandomness")); err != nil {
			panic(err)
		}
	}))
	urlrandmness, err := extrand.GetFromURL(context.Background(), s.URL)
	require.NoError(t, err)

	// Contribute with the two available external randomness bytes.
	err = c.Contribute(drandres, urlrandmness)
	require.NoError(t, err)
}

func BenchmarkDecodeJSON4096(b *testing.B) {
	contributionFile, err := os.ReadFile("testdata/initialContribution4096.json")
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeContribution(contributionFile)
	}
}

func BenchmarkDecodeJSON8192(b *testing.B) {
	contributionFile, err := os.ReadFile("testdata/initialContribution8192.json")
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeContribution(contributionFile)
	}
}

func BenchmarkDecodeJSON16384(b *testing.B) {
	contributionFile, err := os.ReadFile("testdata/initialContribution16384.json")
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeContribution(contributionFile)
	}
}

func BenchmarkDecodeJSON32768(b *testing.B) {
	contributionFile, err := os.ReadFile("testdata/initialContribution32768.json")
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeContribution(contributionFile)
	}
}

func BenchmarkContribute4096(b *testing.B) {
	contributionFile, err := os.ReadFile("testdata/initialContribution4096.json")
	require.NoError(b, err)
	c, err := DecodeContribution(contributionFile)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Contribute()
	}
}

func BenchmarkContribute8192(b *testing.B) {
	contributionFile, err := os.ReadFile("testdata/initialContribution8192.json")
	require.NoError(b, err)
	c, err := DecodeContribution(contributionFile)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Contribute()
	}
}

func BenchmarkContribute16384(b *testing.B) {
	contributionFile, err := os.ReadFile("testdata/initialContribution16384.json")
	require.NoError(b, err)
	c, err := DecodeContribution(contributionFile)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Contribute()
	}
}

func BenchmarkContribute32768(b *testing.B) {
	contributionFile, err := os.ReadFile("testdata/initialContribution32768.json")
	require.NoError(b, err)
	c, err := DecodeContribution(contributionFile)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Contribute()
	}
}

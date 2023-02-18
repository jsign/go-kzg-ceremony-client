package contribution

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jsign/go-kzg-ceremony-client/extrand"
	"github.com/stretchr/testify/require"
)

func TestWithInitialContributionFile(t *testing.T) {
	t.Parallel()

	contributionFile, err := os.ReadFile("testdata/initialContribution.json")
	require.NoError(t, err)

	bc, err := DecodeBatchContribution(contributionFile)
	require.NoError(t, err)

	err = bc.Contribute()
	require.NoError(t, err)

	originalbc, err := DecodeBatchContribution(contributionFile)
	require.NoError(t, err)
	ok, err := bc.Verify(originalbc)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestReferenceImplementationTestVector(t *testing.T) {
	t.Parallel()

	subTests := []struct {
		name     string
		secrets  []string
		mustFail bool
	}{
		{
			name: "success",
			// Use the same secrets for sub-ceremonies as in the test vector.
			// See github.com/jsign/kzg-ceremony-test-vectors
			secrets: []string{"0x111100", "0x221100", "0x331100", "0x441100"},
		},
		{
			name: "must fail",
			// Use slightly modified secrets. This should cause a failure!
			// PotPubkey and the powers of Tau will mismatch!
			secrets:  []string{"0x111101", "0x221101", "0x331101", "0x441101"},
			mustFail: true,
		},
	}

	for _, subTest := range subTests {
		subTest := subTest
		t.Run(subTest.name, func(t *testing.T) {
			t.Parallel()

			// Start from the initialContribution.json defined in the spec.
			contributionFile, err := os.ReadFile("testdata/initialContribution.json")
			require.NoError(t, err)

			// Parse it.
			bc, err := DecodeBatchContribution(contributionFile)
			require.NoError(t, err)

			// Calculate the new batch contribution using **our** client logic.
			err = bc.contributeWithSecrets(subTest.secrets)
			require.NoError(t, err)

			// In `bc` we have **our** calculated BatchContribution for the defined secrets.
			// Now we have to compare it against the test-vector result.

			// Pull the expected batch contribution test-vector.
			res, err := http.DefaultClient.Get("https://raw.githubusercontent.com/jsign/kzg-ceremony-test-vectors/main/updatedContribution.json")
			require.NoError(t, err)
			defer res.Body.Close()
			expectedBatchContributionJSON, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			// Parse it.
			expectedBc, err := DecodeBatchContribution(expectedBatchContributionJSON)
			require.NoError(t, err)

			// Now we'll compare `bc` (our batch contribution) with `expectedBc` (test-vector result).

			// Check # of sub-ceremonies.
			require.Equal(t, len(expectedBc.Contributions), len(bc.Contributions))

			// For each contribution check that:
			// - NumG1Powers and NumG2Powers match.
			// - PotPubkey match (or not, if we're expecting a failure).
			// - Calculated G1Affines and G2Affines from PowersOfTau match (or not, if we're expecting a failure).
			for i := range expectedBc.Contributions {
				require.Equal(t, expectedBc.Contributions[i].NumG1Powers, bc.Contributions[i].NumG1Powers)
				require.Equal(t, expectedBc.Contributions[i].NumG2Powers, bc.Contributions[i].NumG2Powers)

				if subTest.mustFail {
					require.False(t, expectedBc.Contributions[i].PotPubKey.Equal(&bc.Contributions[i].PotPubKey))
				} else {
					require.True(t, expectedBc.Contributions[i].PotPubKey.Equal(&bc.Contributions[i].PotPubKey))
				}

				for j := range expectedBc.Contributions[i].PowersOfTau.G1Affines {
					expG1 := expectedBc.Contributions[i].PowersOfTau.G1Affines[j]
					gotG1 := bc.Contributions[i].PowersOfTau.G1Affines[j]

					// Note that for j==0 we have to make an exception, since G1[0] is always the generator
					// independently from the secret!.
					if j != 0 && subTest.mustFail {
						require.False(t, expG1.Equal(&gotG1))
					} else {
						require.True(t, expG1.Equal(&gotG1))
					}
				}
				for j := range expectedBc.Contributions[i].PowersOfTau.G2Affines {
					expG2 := expectedBc.Contributions[i].PowersOfTau.G2Affines[j]
					gotG2 := bc.Contributions[i].PowersOfTau.G2Affines[j]
					if j != 0 && subTest.mustFail {
						require.False(t, expG2.Equal(&gotG2))
					} else {
						require.True(t, expG2.Equal(&gotG2))
					}
				}
			}
		})
	}
}

func TestExternalRandomness(t *testing.T) {
	t.Parallel()

	contributionFile, err := os.ReadFile("testdata/initialContribution.json")
	require.NoError(t, err)

	bc, err := DecodeBatchContribution(contributionFile)
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
			t.Fatal(err)
		}
	}))
	urlrandmness, err := extrand.GetFromURL(context.Background(), s.URL)
	require.NoError(t, err)

	// Contribute with the two available external randomness bytes.
	err = bc.Contribute(drandres, urlrandmness)
	require.NoError(t, err)
}

func BenchmarkDecodeJSON(b *testing.B) {
	contributionFile, err := os.ReadFile("testdata/initialContribution.json")
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeBatchContribution(contributionFile)
	}
}

func BenchmarkContribute(b *testing.B) {
	contributionFile, err := os.ReadFile("testdata/initialContribution.json")
	require.NoError(b, err)
	bc, err := DecodeBatchContribution(contributionFile)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bc.Contribute()
	}
}

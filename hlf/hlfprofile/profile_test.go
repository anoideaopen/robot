//go:build !integration
// +build !integration

package hlfprofile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseProfile(t *testing.T) {
	hlfProfile, err := ParseProfile("connection.yaml")
	require.Nil(t, err)
	require.NotNil(t, hlfProfile)

	require.EqualValues(t, "Atomyze", hlfProfile.OrgName)
	require.EqualValues(t, "atomyzeMSP", hlfProfile.MspID)
	require.EqualValues(t, "dev-data/hlf-test-stage-04/crypto/backend@atomyze.atmz-test-04.ledger.n-t.io/msp/signcerts", hlfProfile.CredentialStorePath)
	require.EqualValues(t, "dev-data/hlf-test-stage-04/crypto/backend@atomyze.atmz-test-04.ledger.n-t.io/msp", hlfProfile.CryptoStorePath)
}

package unit

import (
	"testing"

	"github.com/anoideaopen/robot/hlf/hlfprofile"
	"github.com/stretchr/testify/require"
)

func TestParseProfile(t *testing.T) {
	hlfProfile, err := hlfprofile.ParseProfile("../../hlf/hlfprofile/connection.yaml")
	require.NoError(t, err)
	require.NotNil(t, hlfProfile)

	require.Equal(t, "Testnet", hlfProfile.OrgName)
	require.Equal(t, "TestnetMSP", hlfProfile.MspID)
	require.Equal(t, "dev-data/hlf-test-stage-04/crypto/backend@testnet.anoideaopen-04.scientificideas.org/msp/signcerts", hlfProfile.CredentialStorePath)
	require.Equal(t, "dev-data/hlf-test-stage-04/crypto/backend@testnet.anoideaopen-04.scientificideas.org/msp", hlfProfile.CryptoStorePath)
}

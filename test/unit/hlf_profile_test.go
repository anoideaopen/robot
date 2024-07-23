package unit

import (
	"testing"

	"github.com/anoideaopen/robot/hlf/hlfprofile"
	"github.com/stretchr/testify/require"
)

func TestParseProfile(t *testing.T) {
	hlfProfile, err := hlfprofile.ParseProfile("../../hlf/hlfprofile/connection.yaml")
	require.Nil(t, err)
	require.NotNil(t, hlfProfile)

	require.EqualValues(t, "Testnet", hlfProfile.OrgName)
	require.EqualValues(t, "TestnetMSP", hlfProfile.MspID)
	require.EqualValues(t, "dev-data/hlf-test-stage-04/crypto/backend@testnet.anoideaopen-04.scientificideas.org/msp/signcerts", hlfProfile.CredentialStorePath)
	require.EqualValues(t, "dev-data/hlf-test-stage-04/crypto/backend@testnet.anoideaopen-04.scientificideas.org/msp", hlfProfile.CryptoStorePath)
}

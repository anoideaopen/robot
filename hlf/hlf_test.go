package hlf

import (
	"testing"

	"github.com/anoideaopen/robot/helpers/ntesting"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/stretchr/testify/require"
)

func TestSign(t *testing.T) {
	t.Skip()
	ciData := ntesting.CI(t)

	configPrvdr := config.FromFile(ciData.HlfProfilePath)
	require.NotNil(t, configPrvdr)

	sdk, err := fabsdk.New(configPrvdr)
	require.NoError(t, err)
	require.NotNil(t, sdk)

	channelPrvdr := sdk.ChannelContext(
		ciData.HlfFiatChannel,
		fabsdk.WithUser(ciData.HlfUserName),
		fabsdk.WithOrg(ciData.HlfProfile.OrgName))
	require.NotNil(t, channelPrvdr)

	clientPrvdr := sdk.Context()

	client, err := clientPrvdr()
	require.NoError(t, err)
	require.NotNil(t, client)

	im, ok := client.IdentityManager(ciData.HlfProfile.OrgName)
	require.True(t, ok)
	require.NotNil(t, im)

	signingIdentity, err := im.GetSigningIdentity(ciData.HlfUserName)
	require.NoError(t, err)
	require.NotNil(t, signingIdentity)

	sk := signingIdentity.PrivateKey()
	require.NotNil(t, sk)

	signingManager := client.SigningManager()
	require.NotNil(t, signingManager)

	msg := []byte{1, 2, 3, 4, 5}
	signature, err := signingManager.Sign(msg, sk)
	require.NoError(t, err)
	require.NotEmpty(t, signature)
}

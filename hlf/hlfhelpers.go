package hlf

import (
	"context"
	"strings"
	"sync"

	"github.com/atomyze-foundation/cartridge"
	"github.com/atomyze-foundation/cartridge/manager"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/status"
	hlfcontext "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/pkg/errors"
)

type sdkComponents struct {
	chProvider hlfcontext.ChannelProvider
}

var (
	createSdkLock sync.RWMutex
	fabricSDK     *fabsdk.FabricSDK
)

func createOrGetSdk(configBackends []core.ConfigBackend, cryptoManager manager.Manager) (*fabsdk.FabricSDK, error) {
	createSdkLock.RLock()
	if fabricSDK != nil {
		return fabricSDK, nil
	}
	createSdkLock.RUnlock()

	createSdkLock.Lock()
	defer createSdkLock.Unlock()

	if fabricSDK != nil {
		return fabricSDK, nil
	}
	var opts []fabsdk.Option
	if cryptoManager != nil {
		connectOpts, err := cartridge.NewConnector(cryptoManager, cartridge.NewVaultConnectProvider(configBackends...)).Opts()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		opts = connectOpts
	}
	skd, err := fabsdk.New(configBackendsToProvider(configBackends), opts...)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	fabricSDK = skd
	return fabricSDK, nil
}

func createSdkComponentsWithoutCryptoMng(
	_ context.Context,
	chName, orgName, userName string,
	configBackends []core.ConfigBackend,
) (*sdkComponents, error) {
	sdk, err := createOrGetSdk(configBackends, nil)
	if err != nil {
		return nil, err
	}

	return &sdkComponents{
		chProvider: sdk.ChannelContext(chName, fabsdk.WithUser(userName), fabsdk.WithOrg(orgName)),
	}, nil
}

func createSdkComponentsWithCryptoMng(
	_ context.Context,
	chName, orgName string,
	configBackends []core.ConfigBackend,
	cryptoManager manager.Manager,
) (*sdkComponents, error) {
	sdk, err := createOrGetSdk(configBackends, cryptoManager)
	if err != nil {
		return nil, err
	}

	return &sdkComponents{
		chProvider: sdk.ChannelContext(chName, fabsdk.WithIdentity(cryptoManager.SigningIdentity()), fabsdk.WithOrg(orgName)),
	}, nil
}

func configBackendsToProvider(cb []core.ConfigBackend) func() ([]core.ConfigBackend, error) {
	return func() ([]core.ConfigBackend, error) {
		return cb, nil
	}
}

// IsEndorsementMismatchErr checks if the error is a mismatch.
func IsEndorsementMismatchErr(err error) bool {
	if err == nil {
		return false
	}
	s, ok := status.FromError(err)
	return ok && status.Code(s.Code) == status.EndorsementMismatch
}

const errReqSizeMarker = "is bigger than request max bytes"

func isOrderingReqSizeExceededErr(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), errReqSizeMarker)
}

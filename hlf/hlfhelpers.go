package hlf

import (
	"context"
	"strings"
	"sync"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/status"
	hlfcontext "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

type sdkComponents struct {
	chProvider hlfcontext.ChannelProvider
}

var (
	createSdkLock sync.RWMutex
	fabricSDK     *fabsdk.FabricSDK
)

func createOrGetSdk(configBackends []core.ConfigBackend) (*fabsdk.FabricSDK, error) {
	createSdkLock.RLock()
	if fabricSDK != nil {
		createSdkLock.RUnlock()
		return fabricSDK, nil
	}
	createSdkLock.RUnlock()

	createSdkLock.Lock()
	defer createSdkLock.Unlock()

	if fabricSDK != nil {
		return fabricSDK, nil
	}
	var opts []fabsdk.Option
	skd, err := fabsdk.New(configBackendsToProvider(configBackends), opts...)
	if err != nil {
		return nil, err
	}
	fabricSDK = skd
	return fabricSDK, nil
}

func createSdkComponents(
	_ context.Context,
	chName, orgName, userName string,
	configBackends []core.ConfigBackend,
) (*sdkComponents, error) {
	sdk, err := createOrGetSdk(configBackends)
	if err != nil {
		return nil, err
	}

	return &sdkComponents{
		chProvider: sdk.ChannelContext(chName, fabsdk.WithUser(userName), fabsdk.WithOrg(orgName)),
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

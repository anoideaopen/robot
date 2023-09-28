package cc

import (
	"fmt"

	basetoken "github.com/atomyze-foundation/robot/hlf/sdkwrapper/chaincode-api/base-token"
	"github.com/atomyze-foundation/robot/hlf/sdkwrapper/service"
)

// API - base token api struct
type API struct {
	*basetoken.BaseTokenAPI
}

// NewAPI - create new base token api
func NewAPI(channelName string, chaincodeName string, hlfClient *service.HLFClient, ownerKeyPair *basetoken.KeyPair) (*API, error) {
	cc := &API{basetoken.NewBaseTokenAPI(channelName, chaincodeName, hlfClient, ownerKeyPair)}
	err := cc.Validate()
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	return cc, nil
}

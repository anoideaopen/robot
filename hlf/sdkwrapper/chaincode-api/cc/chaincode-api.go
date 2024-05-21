package cc

import (
	"fmt"

	"github.com/anoideaopen/robot/hlf/sdkwrapper/chaincode-api/basetoken"
	"github.com/anoideaopen/robot/hlf/sdkwrapper/service"
)

type API struct {
	*basetoken.BaseTokenAPI
}

func NewAPI(channelName string, chaincodeName string, hlfClient *service.HLFClient, ownerKeyPair *basetoken.KeyPair) (*API, error) {
	cc := &API{basetoken.NewBaseTokenAPI(channelName, chaincodeName, hlfClient, ownerKeyPair)}
	err := cc.Validate()
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	return cc, nil
}

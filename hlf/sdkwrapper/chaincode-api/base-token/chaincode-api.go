package base_token

import (
	"errors"

	"github.com/atomyze-foundation/robot/hlf/sdkwrapper/service"
)

// ChaincodeAPI - base token api struct
type ChaincodeAPI struct {
	// ChannelName - channel name for chaincode
	ChannelName string
	// ChaincodeName - chaincode name for base token
	ChaincodeName string
	hlfClient     *service.HLFClient
}

// GetChannelName - get channel name
func (b *ChaincodeAPI) GetChannelName() string {
	return b.ChannelName
}

// GetChaincodeName - get chaincode name
func (b *ChaincodeAPI) GetChaincodeName() string {
	return b.ChaincodeName
}

// GetHlfClient - get hlf client for chaincode
func (b *ChaincodeAPI) GetHlfClient() *service.HLFClient {
	return b.hlfClient
}

// Validate - validate chaincode api
func (b *ChaincodeAPI) Validate() error {
	if b.hlfClient == nil {
		return errors.New("hlf can't be nil")
	}
	if len(b.ChaincodeName) == 0 {
		return errors.New("ChaincodeName can't be nil")
	}
	if len(b.ChannelName) == 0 {
		return errors.New("ChannelName can't be nil")
	}
	return nil
}

// NewChaincodeAPI - create new base token api
func NewChaincodeAPI(channelName string, chaincodeName string, hlfClient *service.HLFClient) *ChaincodeAPI {
	c := &ChaincodeAPI{
		ChannelName:   channelName,
		ChaincodeName: chaincodeName,
		hlfClient:     hlfClient,
	}
	return c
}

package basetoken

import (
	"errors"

	"github.com/anoideaopen/robot/hlf/sdkwrapper/service"
)

type ChaincodeAPI struct {
	ChannelName   string
	ChaincodeName string
	hlfClient     *service.HLFClient
}

func (b *ChaincodeAPI) GetChannelName() string {
	return b.ChannelName
}

func (b *ChaincodeAPI) GetChaincodeName() string {
	return b.ChaincodeName
}

func (b *ChaincodeAPI) GetHlfClient() *service.HLFClient {
	return b.hlfClient
}

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

func NewChaincodeAPI(channelName string, chaincodeName string, hlfClient *service.HLFClient) *ChaincodeAPI {
	c := &ChaincodeAPI{
		ChannelName:   channelName,
		ChaincodeName: chaincodeName,
		hlfClient:     hlfClient,
	}
	return c
}

package fiat

import (
	"fmt"

	basetoken "github.com/anoideaopen/robot/hlf/sdkwrapper/chaincode-api/base-token"
	"github.com/anoideaopen/robot/hlf/sdkwrapper/service"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
)

type API struct {
	*basetoken.BaseTokenAPI
}

func NewAPI(channelName string, chaincodeName string, hlfClient *service.HLFClient, ownerKeyPair *basetoken.KeyPair) (*API, error) {
	err := hlfClient.AddChannel(channelName, service.BatchExecuteEvent)
	if err != nil {
		return nil, err
	}
	cc := &API{basetoken.NewBaseTokenAPI(channelName, chaincodeName, hlfClient, ownerKeyPair)}
	err = cc.Validate()
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	return cc, nil
}

func (b *API) Emit(toAddress string, emitAmount uint64) (*channel.Response, error) {
	err := b.Validate()
	if err != nil {
		return nil, err
	}
	methodArgs := []string{
		toAddress,
		fmt.Sprintf("%d", emitAmount),
	}
	methodName := "emit"
	peers := ""

	return b.GetHlfClient().InvokeWithPublicAndPrivateKey(b.OwnerKeyPair.PrivateKey, b.OwnerKeyPair.PublicKey, b.ChannelName, b.ChaincodeName, methodName, methodArgs, false, peers)
}

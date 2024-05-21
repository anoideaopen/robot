package acl

import (
	"fmt"

	"github.com/anoideaopen/robot/hlf/sdkwrapper/chaincode-api/basetoken"
	"github.com/anoideaopen/robot/hlf/sdkwrapper/service"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
)

type API struct {
	*basetoken.ChaincodeAPI
}

func NewAPI(channelName string, chaincodeName string, hlfClient *service.HLFClient) (*API, error) {
	err := hlfClient.AddChannel(channelName)
	if err != nil {
		return nil, err
	}
	acl := &API{basetoken.NewChaincodeAPI(channelName, chaincodeName, hlfClient)}
	err = acl.Validate()
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	return acl, nil
}

// AddUser - add user
// encodedBase58PublicKey := args[0]
// kycHash := args[1]
// userId := args[2]
// isIndustrial := args[3] == "true"
func (b *API) AddUser(encodedBase58PublicKey string, kycHash string, userID string, isIndustrial string) (*channel.Response, error) {
	err := b.Validate()
	if err != nil {
		return nil, err
	}
	methodArgs := []string{
		encodedBase58PublicKey,
		kycHash,
		userID,
		isIndustrial,
	}
	methodName := "addUser"
	peers := ""

	return b.GetHlfClient().Invoke(b.ChannelName, b.GetChaincodeName(), methodName, methodArgs, true, peers)
}

func (b *API) CheckKeys(publicKeyEncodedBase58 string) (*channel.Response, error) {
	err := b.Validate()
	if err != nil {
		return nil, err
	}
	methodArgs := []string{
		publicKeyEncodedBase58,
	}
	methodName := "checkKeys"
	return b.GetHlfClient().Query(b.ChannelName, b.ChaincodeName, methodName, methodArgs)
}

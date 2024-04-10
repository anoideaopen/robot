package test_cases

import (
	"errors"

	"github.com/anoideaopen/robot/hlf/sdkwrapper/chaincode-api/acl"
	base_token "github.com/anoideaopen/robot/hlf/sdkwrapper/chaincode-api/base-token"
	"github.com/anoideaopen/robot/hlf/sdkwrapper/service"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
	"golang.org/x/crypto/ed25519"
)

func AddUserByUserId(aclAPI *acl.API, userID string) (*base_token.KeyPair, error) {
	publicKey, privateKey, err := service.GeneratePrivateAndPublicKey()
	if err != nil {
		return nil, err
	}

	err = AddUserByUserIdAndPublicKey(aclAPI, userID, publicKey)
	if err != nil {
		return nil, err
	}

	return &base_token.KeyPair{
		UserID:     userID,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, nil
}

func AddUserByUserIdAndPublicKey(aclAPI *acl.API, userID string, publicKey ed25519.PublicKey) error {
	encodedBase58PublicKey := service.ConvertPublicKeyToBase58(publicKey)

	kycHash := "kychash"
	isIndustrial := "true"

	resp, err := aclAPI.AddUser(encodedBase58PublicKey, kycHash, userID, isIndustrial)
	if err != nil {
		return err
	}
	if resp.TxValidationCode != peer.TxValidationCode_VALID ||
		resp.ChaincodeStatus != int32(common.Status_SUCCESS) {
		return errors.New("error validate transaction")
	}

	_, err = aclAPI.CheckKeys(encodedBase58PublicKey)
	if err != nil {
		return err
	}

	return nil
}

package test_cases

import (
	"errors"
	"fmt"
	"strconv"

	basetoken "github.com/atomyze-foundation/robot/hlf/sdkwrapper/chaincode-api/base-token"
	"github.com/atomyze-foundation/robot/hlf/sdkwrapper/service"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
)

// GetBalance - get balance from token channel
func GetBalance(tokenChannel basetoken.BaseTokenInterface, address string) (uint64, error) {
	var err error
	var amount uint64

	amountResp, err := tokenChannel.BalanceOf(address)
	if err != nil {
		return amount, err
	}
	if amountResp == nil {
		return amount, errors.New("amount resp can't be nil")
	}

	amount, err = getAmountFromResponse(amountResp)
	if err != nil {
		return amount, err
	}

	return amount, nil
}

// GetAllowedBalance - get allowed balance from external channel
func GetAllowedBalance(externalChannel basetoken.BaseTokenInterface, address string, token string) (uint64, error) {
	var err error
	var allowedAmount uint64

	allowedAmountResp, err := externalChannel.AllowedBalanceOf(address, token)
	if err != nil {
		return allowedAmount, err
	}
	if allowedAmountResp == nil {
		return allowedAmount, errors.New("allowed amount resp can't be nil")
	}

	allowedAmount, err = getAmountFromResponse(allowedAmountResp)
	if err != nil {
		return allowedAmount, err
	}

	return allowedAmount, nil
}

// UserBalanceShouldBe - check user balance
func UserBalanceShouldBe(keyPair *basetoken.KeyPair, tokenOwnChannel basetoken.BaseTokenInterface, expectedAmount uint64) error {
	address, err := service.GetAddressByPublicKey(keyPair.PublicKey)
	if err != nil {
		return err
	}
	amount, err := GetBalance(tokenOwnChannel, address)
	if err != nil {
		return err
	}

	if amount != expectedAmount {
		return errors.New(fmt.Sprintf("amount %d but expect amount is %d", amount, expectedAmount))
	}

	return nil
}

// UserAllowedBalanceShouldBe - check user allowed balance
func UserAllowedBalanceShouldBe(keyPair *basetoken.KeyPair, externalChannel basetoken.BaseTokenInterface, token string, expectedAllowedAmount uint64) error {
	address, err := service.GetAddressByPublicKey(keyPair.PublicKey)
	if err != nil {
		return err
	}

	allowedAmount, err := GetAllowedBalance(externalChannel, address, token)
	if err != nil {
		return err
	}

	if allowedAmount != expectedAllowedAmount {
		return errors.New(fmt.Sprintf("allowed amount %d but expect allowed amount is %d", allowedAmount, expectedAllowedAmount))
	}

	return nil
}

func getAmountFromResponse(response *channel.Response) (uint64, error) {
	// remove quotes from around a string
	payload := string(response.Payload)
	amountString := payload[1 : len(payload)-1]
	return strconv.ParseUint(amountString, 10, 64)
}

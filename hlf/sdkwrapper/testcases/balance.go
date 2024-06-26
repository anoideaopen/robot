package testcases

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/anoideaopen/robot/hlf/sdkwrapper/chaincode-api/basetoken"
	"github.com/anoideaopen/robot/hlf/sdkwrapper/service"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
)

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
		return fmt.Errorf("amount %d but expect amount is %d", amount, expectedAmount)
	}

	return nil
}

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
		return fmt.Errorf("allowed amount %d but expect allowed amount is %d", allowedAmount, expectedAllowedAmount)
	}

	return nil
}

func getAmountFromResponse(response *channel.Response) (uint64, error) {
	// remove quotes from around a string
	payload := string(response.Payload)
	amountString := payload[1 : len(payload)-1]
	return strconv.ParseUint(amountString, 10, 64)
}

package test_cases

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	base_token "github.com/atomyze-foundation/robot/hlf/sdkwrapper/chaincode-api/base-token"
	"github.com/atomyze-foundation/robot/hlf/sdkwrapper/logger"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"go.uber.org/zap"
	"golang.org/x/crypto/sha3"
)

// Swap user token from one cc to another cc
// fromChaincodeAPI - source chaincode api
// toChaincodeAPI - target chaincode api
// user - who will swap token - token owner
// token - title token
// swapAmount - amount of token
func Swap(fromChaincodeAPI base_token.BaseTokenInterface, toChaincodeAPI base_token.BaseTokenInterface, keyPair *base_token.KeyPair, token string, swapAmount uint64) error {
	swapKey := strconv.Itoa(int(time.Now().UnixNano()))
	swapHash := sha3.Sum256([]byte(swapKey))

	response, err := fromChaincodeAPI.SwapBegin(
		keyPair,
		token,
		toChaincodeAPI.GetChannelName(),
		swapAmount,
		hex.EncodeToString(swapHash[:]),
	)
	if err != nil {
		panic(err)
	}
	if response.TxValidationCode != peer.TxValidationCode_VALID {
		err = errors.New(fmt.Sprintf("TxValidationCode %d", response.TxValidationCode))
		panic(err)
	}
	swapTransactionId := string(response.TransactionID)

	// check swap is created in both channels
	resp, err := WaitSwapInOtherChannel(fromChaincodeAPI, swapTransactionId)
	if err != nil {
		panic(err)
	}
	logger.Info("waitSwapInOtherChannel", zap.ByteString("from fiat to cc", resp.Payload))

	resp, err = WaitSwapInOtherChannel(toChaincodeAPI, swapTransactionId)
	if err != nil {
		panic(err)
	}
	logger.Info("waitSwapInOtherChannel", zap.ByteString("from fiat to cc", resp.Payload))

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	err = Retry(ctx, func() error {
		response, err = toChaincodeAPI.SwapDone(swapTransactionId, swapKey)
		return err
	}, 5000*time.Millisecond)

	return err
}

// WaitSwapInOtherChannel wait swap in other channel
func WaitSwapInOtherChannel(chaincodeAPI base_token.BaseTokenInterface, swapTransactionId string) (*channel.Response, error) {
	var response *channel.Response
	var err error

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	err = Retry(ctx, func() error {
		response, err = chaincodeAPI.SwapGet(swapTransactionId)
		return err
	}, 5000*time.Millisecond)

	return response, err
}

// Retry function for retrying some action
func Retry(
	ctx context.Context,
	what func() error,
	d time.Duration,
) error {
	for {
		err := what()
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout (%w)", err)
		case <-time.After(d):
		}
	}
}

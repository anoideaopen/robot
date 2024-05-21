package testcases

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/anoideaopen/robot/hlf/sdkwrapper/chaincode-api/basetoken"
	"github.com/anoideaopen/robot/hlf/sdkwrapper/logger"
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
func Swap(fromChaincodeAPI basetoken.BaseTokenInterface, toChaincodeAPI basetoken.BaseTokenInterface, keyPair *basetoken.KeyPair, token string, swapAmount uint64) error {
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
		err = fmt.Errorf("TxValidationCode %d", response.TxValidationCode)
		panic(err)
	}
	swapTransactionID := string(response.TransactionID)

	// check swap is created in both channels
	resp, err := WaitSwapInOtherChannel(fromChaincodeAPI, swapTransactionID)
	if err != nil {
		panic(err)
	}
	logger.Info("waitSwapInOtherChannel", zap.ByteString("from fiat to cc", resp.Payload))

	resp, err = WaitSwapInOtherChannel(toChaincodeAPI, swapTransactionID)
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
		response, err = toChaincodeAPI.SwapDone(swapTransactionID, swapKey)
		return err
	}, 5000*time.Millisecond)

	return err
}

func WaitSwapInOtherChannel(chaincodeAPI basetoken.BaseTokenInterface, swapTransactionID string) (*channel.Response, error) {
	var response *channel.Response
	var err error

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	err = Retry(ctx, func() error {
		response, err = chaincodeAPI.SwapGet(swapTransactionID)
		return err
	}, 5000*time.Millisecond)

	return response, err
}

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

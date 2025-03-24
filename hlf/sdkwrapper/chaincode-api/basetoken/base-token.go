package basetoken

import (
	"crypto/ed25519"
	"fmt"
	"strconv"
	"strings"

	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/robot/hlf/sdkwrapper/service"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
)

type dealType string

type KeyPair struct {
	UserID     string
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

type BaseTokenInterface interface { //nolint:interfacebloat
	SetRate(dealType dealType, currency string, rate uint64) (*channel.Response, error)
	DeleteRate(dealType dealType, currency string, rate uint64) (*channel.Response, error)
	Metadata() (*channel.Response, error)
	BalanceOf(address string) (*channel.Response, error)
	AllowedBalanceOf(address string, token string) (*channel.Response, error)
	Transfer(keyPair *KeyPair, toAddress string, amount *big.Int, ref string) (*channel.Response, error)
	SwapBegin(keyPair *KeyPair, token string, contractTo string, amount uint64, hash string) (*channel.Response, error)
	SwapCancel(keyPair *KeyPair, swapTransactionID string) (*channel.Response, error)
	SwapDone(swapTransactionID string, swapKey string) (*channel.Response, error)
	SwapGet(swapTransactionID string) (*channel.Response, error)
	GetChannelName() string
	GetChaincodeName() string
	GetHlfClient() *service.HLFClient
}

type BaseTokenAPI struct {
	*ChaincodeAPI
	OwnerKeyPair *KeyPair
}

func (b *BaseTokenAPI) GetChannelName() string {
	return b.ChannelName
}

func (b *BaseTokenAPI) GetChaincodeName() string {
	return b.ChaincodeName
}

func (b *BaseTokenAPI) GetHlfClient() *service.HLFClient {
	return b.hlfClient
}

func NewBaseTokenAPI(channelName string, chaincodeName string, hlfClient *service.HLFClient, ownerKeyPair *KeyPair) *BaseTokenAPI {
	return &BaseTokenAPI{
		ChaincodeAPI: NewChaincodeAPI(channelName, chaincodeName, hlfClient),
		OwnerKeyPair: ownerKeyPair,
	}
}

// SetRate - Signed by the issuer
// dealType - type of deal
// currency - currency type
// rate - value
func (b *BaseTokenAPI) SetRate(dealType dealType, currency string, rate uint64) (*channel.Response, error) {
	err := b.Validate()
	if err != nil {
		return nil, err
	}
	methodArgs := []string{
		string(dealType),
		currency,
		strconv.FormatUint(rate, 10),
	}
	methodName := "setRate"
	peers := ""

	return b.GetHlfClient().InvokeWithPublicAndPrivateKey(b.OwnerKeyPair.PrivateKey, b.OwnerKeyPair.PublicKey, b.ChannelName, b.ChaincodeName, methodName, methodArgs, true, peers)
}

// DeleteRate - Signed by the issuer
// dealType - type of deal
// currency - currency type
// rate - value
func (b *BaseTokenAPI) DeleteRate(dealType dealType, currency string, rate uint64) (*channel.Response, error) {
	err := b.Validate()
	if err != nil {
		return nil, err
	}
	methodArgs := []string{
		string(dealType),
		currency,
		strconv.FormatUint(rate, 10),
	}
	methodName := "deleteRate"
	peers := ""

	return b.GetHlfClient().InvokeWithPublicAndPrivateKey(b.OwnerKeyPair.PrivateKey, b.OwnerKeyPair.PublicKey, b.ChannelName, b.ChaincodeName, methodName, methodArgs, true, peers)
}

// Method signature: TxBuyToken(sender types.Sender, amount *big.Int, currency string) error
// Signed by the user. If signed by the issuer, an error is returned.
// The method checks flights and limits, so they must be set in advance.
// Example from the test:
// issuer.SignedInvoke("vt", "setRate", "buyToken", "usd", "100000000")
// issuer.SignedInvoke("vt", "setLimits", "buyToken", "usd", "1", "10")

// BuyBack - buyback of the token.
//
// Method signature: TxBuyBack(sender types.Sender, amount *big.Int, currency string) error
// Signed by the user. If signed by the issuer, an error is returned. Similarly to the previous method,
// raits and limits are checked.
// Example from the test:
// issuer.SignedInvoke("vt", "setRate", "buyBack", "usd", "100000000")
// issuer.SignedInvoke("vt", "setLimits", "buyBack", "usd", "1", "10")

// Metadata - request the metadata of the token.
// Method signature: QueryMetadata() (metadata, error)
// No additional conditions are required.
func (b *BaseTokenAPI) Metadata() (*channel.Response, error) {
	err := b.Validate()
	if err != nil {
		return nil, err
	}
	var methodArgs []string
	methodName := "metadata"
	return b.hlfClient.Query(b.ChannelName, b.ChaincodeName, methodName, methodArgs)
}

// BalanceOf - balance query.
// Method signature: QueryBalanceOf(address types.Address) (*big.Int, error)
// No additional conditions are required.
func (b *BaseTokenAPI) BalanceOf(address string) (*channel.Response, error) {
	err := b.Validate()
	if err != nil {
		return nil, err
	}
	methodArgs := []string{
		address,
	}
	methodName := "balanceOf"
	return b.hlfClient.Query(b.ChannelName, b.ChaincodeName, methodName, methodArgs)
}

// AllowedBalanceOf - request for allowed balance.
// Method signature: QueryAllowedBalanceOf(address types.Address, token string)
// No additional conditions are required.
func (b *BaseTokenAPI) AllowedBalanceOf(address string, token string) (*channel.Response, error) {
	err := b.Validate()
	if err != nil {
		return nil, err
	}
	methodArgs := []string{
		address,
		strings.ToUpper(token),
	}
	methodName := "allowedBalanceOf"
	return b.hlfClient.Query(b.ChannelName, b.ChaincodeName, methodName, methodArgs)
}

// Method signature: QueryDocumentsList() ([]core.Doc, error)
// No additional conditions are required.

// AddDocs - adding documents to the token.
// Signed by the issuer.
//
// Method signature: TxAddDocs(sender types.Sender, rawDocs string) error

// Method signature: TxDeleteDoc(sender types.Sender, docID string) error
// Signed by the issuer.

// SetLimits - set limit.
// Method signature: TxSetLimits(sender types.Sender, dealType string, currency string, min *big.Int, max *big.Int) error
// Signed by the issuer.

// Transfer - transfer tokens to the specified address.
// Method signature: TxTransfer(sender types.Sender, to types.Address, amount *big.Int, ref string) error
// The number of tokens to be transferred must not be zero, tokens cannot be transferred to oneself,
// if a commission is set and the commission currency is not empty, the sender will be charged a commission.
func (b *BaseTokenAPI) Transfer(keyPair *KeyPair, toAddress string, amount *big.Int, ref string) (*channel.Response, error) {
	err := b.Validate()
	if err != nil {
		return nil, err
	}
	methodArgs := []string{
		toAddress,
		fmt.Sprintf("%d", amount),
		ref,
	}
	methodName := "transfer"
	peers := ""

	return b.hlfClient.InvokeWithPublicAndPrivateKey(keyPair.PrivateKey, keyPair.PublicKey, b.ChannelName, b.ChaincodeName, methodName, methodArgs, false, peers)
}

// toAddress - address, usually base58check from the public key when creating a user

// Method signature: QueryPredictFee(amount *big.Int) (predict, error)
// No additional conditions are required.

// SetFee - setting the fee.
// Method signature: TxSetFee(sender types.Sender, currency string, fee *big.Int, floor *big.Int, cap *big.Int) error
// Signed by FeeSetter.
// Check for the value of commission - it should be not more than 100%,
// also check for the values of floor and cap limits.

// SetFeeAddress - setting the commission address.
// Method signature: TxSetFeeAddress(sender types.Sender, address types.Address) error
// Signed by FeeAddressSetter.

// SwapBegin - the beginning of the atomic swap process.
// Method signature: TxSwapBegin(sender types.Sender, token string, contractTo string, amount *big.Int, hash types.Hex) (string, error)
// No additional conditions are required.
func (b *BaseTokenAPI) SwapBegin(keyPair *KeyPair, token string, contractTo string, amount uint64, hash string) (*channel.Response, error) {
	err := b.Validate()
	if err != nil {
		return nil, err
	}
	methodArgs := []string{
		strings.ToUpper(token),
		strings.ToUpper(contractTo),
		strconv.FormatUint(amount, 10),
		hash,
	}
	methodName := "swapBegin"
	peers := ""

	return b.hlfClient.InvokeWithPublicAndPrivateKey(keyPair.PrivateKey, keyPair.PublicKey, b.ChannelName, b.ChaincodeName, methodName, methodArgs, false, peers)
}

// SwapCancel - swap reset.
// Method signature: TxSwapCancel(sender types.Sender, swapID string) error
// No additional conditions are required.
func (b *BaseTokenAPI) SwapCancel(keyPair *KeyPair, swapTransactionID string) (*channel.Response, error) {
	err := b.Validate()
	if err != nil {
		return nil, err
	}
	methodArgs := []string{
		swapTransactionID,
	}
	methodName := "swapCancel"
	peers := ""

	return b.hlfClient.InvokeWithPublicAndPrivateKey(
		keyPair.PrivateKey,
		keyPair.PublicKey,
		b.ChannelName,
		b.ChaincodeName,
		methodName,
		methodArgs,
		false,
		peers,
	)
}

// SwapDone - start of the atomic swap process.
// No additional conditions are required.
func (b *BaseTokenAPI) SwapDone(swapTransactionID string, swapKey string) (*channel.Response, error) {
	err := b.Validate()
	if err != nil {
		return nil, err
	}
	methodArgs := []string{
		swapTransactionID,
		swapKey,
	}
	methodName := "swapDone"
	peers := ""

	return b.hlfClient.Invoke(b.ChannelName, b.ChaincodeName, methodName, methodArgs, true, peers)
}

// SwapGet - swap information.
// Method signature: QuerySwapGet(swapID string) (*proto.Swap, error)
// No additional conditions are required.
func (b *BaseTokenAPI) SwapGet(swapTransactionID string) (*channel.Response, error) {
	err := b.Validate()
	if err != nil {
		return nil, err
	}
	methodArgs := []string{
		swapTransactionID,
	}
	methodName := "swapGet"
	return b.hlfClient.Query(b.ChannelName, b.ChaincodeName, methodName, methodArgs)
}

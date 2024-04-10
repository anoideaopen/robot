package base_token

import (
	"crypto/ed25519"
	"fmt"
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

type BaseTokenInterface interface {
	SetRate(dealType dealType, currency string, rate uint64) (*channel.Response, error)
	DeleteRate(dealType dealType, currency string, rate uint64) (*channel.Response, error)
	Metadata() (*channel.Response, error)
	BalanceOf(address string) (*channel.Response, error)
	AllowedBalanceOf(address string, token string) (*channel.Response, error)
	Transfer(keyPair *KeyPair, toAddress string, amount *big.Int, ref string) (*channel.Response, error)
	SwapBegin(keyPair *KeyPair, token string, contractTo string, amount uint64, hash string) (*channel.Response, error)
	SwapCancel(keyPair *KeyPair, swapTransactionId string) (*channel.Response, error)
	SwapDone(swapTransactionId string, swapKey string) (*channel.Response, error)
	SwapGet(swapTransactionId string) (*channel.Response, error)
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

// SetRate - установка рейта
// - Подписывается эмитентом
// dealType - тип сделки
// currency - тип валюты
// rate - значение
func (b *BaseTokenAPI) SetRate(dealType dealType, currency string, rate uint64) (*channel.Response, error) {
	err := b.Validate()
	if err != nil {
		return nil, err
	}
	methodArgs := []string{
		string(dealType),
		currency,
		fmt.Sprintf("%d", rate),
	}
	methodName := "setRate"
	peers := ""

	return b.GetHlfClient().InvokeWithPublicAndPrivateKey(b.OwnerKeyPair.PrivateKey, b.OwnerKeyPair.PublicKey, b.ChannelName, b.ChaincodeName, methodName, methodArgs, true, peers)
}

// DeleteRate - установка рейта
// - Подписывается эмитентом
// dealType - тип сделки
// currency - тип валюты
// rate - значение
func (b *BaseTokenAPI) DeleteRate(dealType dealType, currency string, rate uint64) (*channel.Response, error) {
	err := b.Validate()
	if err != nil {
		return nil, err
	}
	methodArgs := []string{
		string(dealType),
		currency,
		fmt.Sprintf("%d", rate),
	}
	methodName := "deleteRate"
	peers := ""

	return b.GetHlfClient().InvokeWithPublicAndPrivateKey(b.OwnerKeyPair.PrivateKey, b.OwnerKeyPair.PublicKey, b.ChannelName, b.ChaincodeName, methodName, methodArgs, true, peers)
}

// Сигнатура метода: TxBuyToken(sender types.Sender, amount *big.Int, currency string) error
// Подписывается пользователем. При подписи эмитентом возвращается ошибка. В рамках работы метода проверяются рейсы и лимиты, поэтому они должны быть заданы заранее.
// Пример из теста:
// issuer.SignedInvoke("vt", "setRate", "buyToken", "usd", "100000000")
// issuer.SignedInvoke("vt", "setLimits", "buyToken", "usd", "1", "10")

// BuyBack - обратный выкуп токена.
//
// Сигнатура метода: TxBuyBack(sender types.Sender, amount *big.Int, currency string) error
// Подписывается пользователем. При подписи эмитентом возвращается ошибка. Аналогично предыдущему методу, производится проверка рейтов и лимитов.
// Пример из теста:
// issuer.SignedInvoke("vt", "setRate", "buyBack", "usd", "100000000")
// issuer.SignedInvoke("vt", "setLimits", "buyBack", "usd", "1", «10")

// Metadata - запрос метадаты токена.
// Сигнатура метода: QueryMetadata() (metadata, error)
// Дополнительные условий не требуется.
func (b *BaseTokenAPI) Metadata() (*channel.Response, error) {
	err := b.Validate()
	if err != nil {
		return nil, err
	}
	var methodArgs []string
	methodName := "metadata"
	return b.hlfClient.Query(b.ChannelName, b.ChaincodeName, methodName, methodArgs)
}

// BalanceOf - запрос баланса.
// Сигнатура метода: QueryBalanceOf(address types.Address) (*big.Int, error)
// Дополнительных условий не требуется.
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

// AllowedBalanceOf - запрос allowed баланса.
// Сигнатура метода: QueryAllowedBalanceOf(address types.Address, token string)
// Дополнительных условий не требуется.
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

// Сигнатура метода: QueryDocumentsList() ([]core.Doc, error)
// Дополнительных условий не требуется.

// AddDocs - добавление документов к токену.
// Подписывается эмитентом.
//
// Сигнатура метода: TxAddDocs(sender types.Sender, rawDocs string) error

// Сигнатура метода: TxDeleteDoc(sender types.Sender, docID string) error
// Подписывается эмитентом.

// SetLimits - установка лимита.
// Сигнатура метода: TxSetLimits(sender types.Sender, dealType string, currency string, min *big.Int, max *big.Int) error
// Подписывается эмитентом.

// Transfer - передача токенов на указанный адрес.
// Сигнатура метода: TxTransfer(sender types.Sender, to types.Address, amount *big.Int, ref string) error
// Количество передаваемых токенов не должно быть нулевым, токены нельзя переслать самому себе, если установлена комиссия и комиссионная валюта не пустая, то с отправителя будет списана комиссия.
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

// toAddress - адрес, обычно base58check от публичного ключа при создании пользователя

// Сигнатура метода: QueryPredictFee(amount *big.Int) (predict, error)
// Дополнительных условий не требуется.

// SetFee - установка комиссии.
// Сигнатура метода: TxSetFee(sender types.Sender, currency string, fee *big.Int, floor *big.Int, cap *big.Int) error
// Подписывается FeeSetter’ом.
// Производится проверка на значение комиссии - она должна быть не более 100%, также производится проверка на значения лимитов floor и cap.

// SetFeeAddress - установка комиссионного адреса.
// Сигнатура метода: TxSetFeeAddress(sender types.Sender, address types.Address) error
// Подписывется FeeAddressSetter’ом.

// SwapBegin - начало процесса атомарного свопа.
// Сигнатура метода: TxSwapBegin(sender types.Sender, token string, contractTo string, amount *big.Int, hash types.Hex) (string, error)
// Дополнительных условий не требуется.
func (b *BaseTokenAPI) SwapBegin(keyPair *KeyPair, token string, contractTo string, amount uint64, hash string) (*channel.Response, error) {
	err := b.Validate()
	if err != nil {
		return nil, err
	}
	methodArgs := []string{
		strings.ToUpper(token),
		strings.ToUpper(contractTo),
		fmt.Sprintf("%d", amount),
		hash,
	}
	methodName := "swapBegin"
	peers := ""

	return b.hlfClient.InvokeWithPublicAndPrivateKey(keyPair.PrivateKey, keyPair.PublicKey, b.ChannelName, b.ChaincodeName, methodName, methodArgs, false, peers)
}

// SwapCancel - сброс свопа.
// Сигнатура метода: TxSwapCancel(sender types.Sender, swapID string) error
// Дополнительных условий не требуется.
func (b *BaseTokenAPI) SwapCancel(keyPair *KeyPair, swapTransactionId string) (*channel.Response, error) {
	err := b.Validate()
	if err != nil {
		return nil, err
	}
	methodArgs := []string{
		swapTransactionId,
	}
	methodName := "swapCancel"
	peers := ""

	return b.hlfClient.InvokeWithPublicAndPrivateKey(keyPair.PrivateKey, keyPair.PublicKey, b.ChannelName, b.ChaincodeName, methodName, methodArgs, false, peers)
}

// SwapDone - начало процесса атомарного свопа.
// Дополнительных условий не требуется.
func (b *BaseTokenAPI) SwapDone(swapTransactionId string, swapKey string) (*channel.Response, error) {
	err := b.Validate()
	if err != nil {
		return nil, err
	}
	methodArgs := []string{
		swapTransactionId,
		swapKey,
	}
	methodName := "swapDone"
	peers := ""

	return b.hlfClient.Invoke(b.ChannelName, b.ChaincodeName, methodName, methodArgs, true, peers)
}

// SwapGet - информация по свопу.
// Сигнатура метода: QuerySwapGet(swapID string) (*proto.Swap, error)
// Дополнительных условий не требуется.
func (b *BaseTokenAPI) SwapGet(swapTransactionId string) (*channel.Response, error) {
	err := b.Validate()
	if err != nil {
		return nil, err
	}
	methodArgs := []string{
		swapTransactionId,
	}
	methodName := "swapGet"
	return b.hlfClient.Query(b.ChannelName, b.ChaincodeName, methodName, methodArgs)
}

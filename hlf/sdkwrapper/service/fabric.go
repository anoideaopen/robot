package service

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/atomyze-foundation/cartridge"
	"github.com/atomyze-foundation/cartridge/manager"
	"github.com/atomyze-foundation/foundation/core"
	"github.com/atomyze-foundation/foundation/proto"
	"github.com/atomyze-foundation/robot/hlf/sdkwrapper/logger"
	pb "github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	contextApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	core2 "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// HLFClient - Hyperledger Fabric client
type HLFClient struct {
	resMgmt                 *resmgmt.Client
	sdk                     *fabsdk.FabricSDK
	channels                channelComponents
	NotifierChaincodeEvents map[string]<-chan *fab.CCEvent
	afterInvokeHandler      HlfAfterInvokeHandler
	beforeInvokeHandler     HlfBeforeInvokeHandler
	ContextOptions          []fabsdk.ContextOption
}

type channelComponents struct {
	sync.RWMutex
	components map[string]*channelConnection
}

const (
	// KeyEvent - event name for key
	KeyEvent = "key"
	// BatchExecuteEvent - event name for batch execute
	BatchExecuteEvent = "batchExecute"
)

// NewHLFClient - create new HLFClient instance
func NewHLFClient(connectionConfigPath string, username string, organization string, vaultConfig *VaultConfig) (*HLFClient, error) {
	var hlfClient *HLFClient
	var err error
	if vaultConfig != nil && len(vaultConfig.Address) != 0 && len(vaultConfig.Token) != 0 && len(vaultConfig.Path) != 0 &&
		len(vaultConfig.UserCertName) != 0 && len(vaultConfig.UserOrgMspID) != 0 {
		hlfClient, err = newHLFClientVault(connectionConfigPath, organization, vaultConfig)
		if err != nil {
			return nil, err
		}
	} else {
		hlfClient, err = newHLFClientFile(connectionConfigPath, username, organization)
		if err != nil {
			return nil, err
		}
	}
	return hlfClient, nil
}

// VaultConfig - config for vault
type VaultConfig struct {
	Token string `mapstructure:"token,omitempty"`
	Path  string `mapstructure:"path,omitempty"`

	Address      string `mapstructure:"address,omitempty"`
	UserCertName string `mapstructure:"user_cert_name,omitempty"`
	UserOrgMspID string `mapstructure:"user_org_msp_id,omitempty"`
}

func newHLFClientVault(connectionConfigPath string, organization string, vaultConfig *VaultConfig) (*HLFClient, error) {
	hlfClient, err := newHLFClient()
	if err != nil {
		logger.Error("Failed to create new hlf client", zap.Error(err))
		return nil, err
	}

	configProvider := config.FromFile(connectionConfigPath)
	configBackends, err := configProvider()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	vaultManager, err := manager.NewVaultManager(
		vaultConfig.UserOrgMspID,
		vaultConfig.UserCertName,
		vaultConfig.Address,
		vaultConfig.Token,
		vaultConfig.Path,
	)
	if err != nil {
		logger.Error("Failed to create new vault manager", zap.Error(err))
		return nil, err
	}

	var connectOpts []fabsdk.Option

	connector := cartridge.NewConnector(vaultManager, cartridge.NewVaultConnectProvider(configBackends...))
	if connector != nil {
		connectOpts, err = connector.Opts()
		if err != nil {
			logger.Error("Failed to get connector Opts", zap.Error(err))
			return nil, err
		}
	}

	err = hlfClient.AddFabsdk(configProvider, connectOpts...)
	if err != nil {
		logger.Error("Failed to create new channel client", zap.Error(err))
		return nil, err
	}
	if hlfClient == nil {
		logger.Error("Failed to create new channel client")
		return nil, err
	}

	hlfClient.ContextOptions = append(hlfClient.ContextOptions, fabsdk.WithIdentity(vaultManager.SigningIdentity()))
	hlfClient.ContextOptions = append(hlfClient.ContextOptions, fabsdk.WithOrg(organization))

	return hlfClient, nil
}

func newHLFClientFile(connectionConfigPath string, username string, organization string) (*HLFClient, error) {
	hlfClient, err := newHLFClient()
	if err != nil {
		logger.Error("Failed to create new hlf client", zap.Error(err))
		return nil, err
	}

	configProvider := config.FromFile(connectionConfigPath)

	err = hlfClient.AddFabsdk(configProvider)
	if err != nil {
		logger.Error("Failed to create new channel client", zap.Error(err))
		return nil, err
	}
	if hlfClient == nil {
		logger.Error("Failed to create new channel client")
		return nil, err
	}

	hlfClient.ContextOptions = append(hlfClient.ContextOptions, fabsdk.WithUser(username))
	hlfClient.ContextOptions = append(hlfClient.ContextOptions, fabsdk.WithOrg(organization))

	return hlfClient, nil
}

func newHLFClient() (*HLFClient, error) {
	hlf := &HLFClient{
		channels:                channelComponents{components: make(map[string]*channelConnection)},
		NotifierChaincodeEvents: map[string]<-chan *fab.CCEvent{},
	}

	return hlf, nil
}

// AddFabsdk - add fabsdk to hlf client
func (hlf *HLFClient) AddFabsdk(configProvider core2.ConfigProvider, opts ...fabsdk.Option) error {
	sdk, err := fabsdk.New(configProvider, opts...)
	if err != nil {
		msg := "failed to create new SDK"
		logger.Error(msg, zap.Error(err))
		return fmt.Errorf("%s: %v", msg, err)
	}

	hlf.sdk = sdk
	return nil
}

// AddChannel - add channel to hlf client
func (hlf *HLFClient) AddChannel(channelID string, events ...string) error {
	channelConnection, err := hlf.getOrCreateChannelConnection(channelID)
	if err != nil {
		return fmt.Errorf("failed to getOrCreateChannelConnection: %s", err)
	}
	if channelConnection == nil {
		return errors.New("channelConnection can't be nil")
	}
	// support empty event list (for example 'acl' without events)
	if events != nil && len(events) != 0 {
		for _, event := range events {
			batchExecuteEventEventNotifier, err := hlf.GetCCEventNotifier(channelConnection.channelClient, channelID, event)
			if err != nil {
				return fmt.Errorf("failed to GetCCEventNotifier: %s", err)
			}
			if batchExecuteEventEventNotifier == nil {
				return errors.New("batchExecuteEventEventNotifier can't be nil")
			}
		}
	}
	return nil
}

// AddBeforeInvokeHandler - add before invoke handler
func (hlf *HLFClient) AddBeforeInvokeHandler(beforeInvokeHandler HlfBeforeInvokeHandler) {
	hlf.beforeInvokeHandler = beforeInvokeHandler
}

// AddAfterInvokeHandler - add after invoke handler
func (hlf *HLFClient) AddAfterInvokeHandler(afterInvokeHandler HlfAfterInvokeHandler) {
	hlf.afterInvokeHandler = afterInvokeHandler
}

func (hlf *HLFClient) createChannelClient(channelID string, options ...fabsdk.ContextOption) (*channel.Client, contextApi.ChannelProvider, error) {
	clientChannelContext := hlf.sdk.ChannelContext(channelID, options...)
	// Channel client is used to query and execute transactions (Org1 is default org)
	client, err := channel.New(clientChannelContext)
	if err != nil {
		return nil, nil, err
	}

	return client, clientChannelContext, err
}

type channelConnection struct {
	channelClient   *channel.Client
	channelProvider contextApi.ChannelProvider
}

func (hlf *HLFClient) getOrCreateChannelConnection(channelID string) (*channelConnection, error) {
	var result *channelConnection
	hlf.channels.RLock()
	val, isExists := hlf.channels.components[channelID]
	hlf.channels.RUnlock()

	if !isExists {
		channelClient, channelProvider, err := hlf.createChannelClient(channelID, hlf.ContextOptions...)
		if err != nil {
			return nil, err
		}
		result = &channelConnection{
			channelClient:   channelClient,
			channelProvider: channelProvider,
		}
		hlf.channels.Lock()
		hlf.channels.components[channelID] = result
		hlf.channels.Unlock()
	} else {
		result = val
	}

	return result, nil
}

// Query - method to send query request to hlf
func (hlf *HLFClient) Query(
	channelID string, chaincodeName string, methodName string, methodArgs []string,
) (*channel.Response, error) {
	channelConnection, err := hlf.getOrCreateChannelConnection(channelID)
	if err != nil {
		return nil, err
	}

	client := channelConnection.channelClient

	return hlf.Request(
		methodArgs, chaincodeName, methodName, true,
		client, client.Query,
		channel.WithRetry(retry.DefaultChannelOpts),
	)
}

// InvokeWithSecretKey - method to sign arguments and send invoke request to hlf
// methodArgs []string -
// secretKey string - private key ed25519 - in base58check, or hex or base58
// chaincodeName string - chaincode name for invoke
// methodName string - chaincode method name for invoke
// noBatch bool - if wait batchTransaction set 'true'
// peers string - peer0.atomyze
func (hlf *HLFClient) InvokeWithSecretKey(channelID string, chaincodeName string, methodName string, methodArgs []string, secretKey string, noBatch bool, peers string) (*channel.Response, error) {
	if len(secretKey) != 0 {
		privateKey, publicKey, err := EncodedPrivKeyToEd25519(secretKey)
		if err != nil {
			logger.Error("failed getPrivateKey", zap.Error(err))
			return nil, err
		}
		methodArgs, err = hlf.SignArgs(channelID, chaincodeName, methodName, methodArgs, privateKey, publicKey)
		if err != nil {
			logger.Error("failed signArgs", zap.Error(err))
			return nil, err
		}
	}

	return hlf.Invoke(channelID, chaincodeName, methodName, methodArgs, noBatch, peers)
}

// InvokeWithPublicAndPrivateKey - method to sign arguments and send invoke request to hlf
// privateKey string - private key in ed25519
// publicKey string - private key in ed25519
// channelID string - channel name for invoke
// chaincodeName string - chaincode name for invoke
// methodName string - chaincode method name for invoke
// methodArgs []string -
// noBatch bool - if wait batchTransaction set 'true'
// peers string - peer0.atomyze
func (hlf *HLFClient) InvokeWithPublicAndPrivateKey(privateKey ed25519.PrivateKey, publicKey ed25519.PublicKey, channelID string, chaincodeName string, methodName string, methodArgs []string, noBatch bool, peers string) (*channel.Response, error) {
	if len(privateKey) == 0 {
		return nil, errors.New("privateKey can't be empty")
	}
	if len(publicKey) == 0 {
		return nil, errors.New("publicKey can't be empty")
	}

	methodArgs, err := hlf.SignArgs(channelID, chaincodeName, methodName, methodArgs, privateKey, publicKey)
	if err != nil {
		logger.Error("failed signArgs", zap.Error(err))
		return nil, err
	}

	return hlf.Invoke(channelID, chaincodeName, methodName, methodArgs, noBatch, peers)
}

// Invoke - method to sign arguments and send invoke request to hlf
// channelID string - channel name for invoke
// chaincodeName string - chaincode name for invoke
// methodName string - chaincode method name for invoke
// methodArgs []string -
// noBatch bool - if wait batchTransaction set 'true'
// peers string - target peer for invoke, if empty use default peer count by policy
func (hlf *HLFClient) Invoke(channelID string, chaincodeName string, methodName string, methodArgs []string, noBatch bool, peers ...string) (*channel.Response, error) {
	channelConnection, err := hlf.getOrCreateChannelConnection(channelID)
	if err != nil {
		return nil, err
	}

	options := make([]channel.RequestOption, 0)
	options = append(options, channel.WithRetry(retry.DefaultChannelOpts))
	if len(peers) != 0 {
		logger.Debug(fmt.Sprintf("targetPeers: %v\n", peers))
		options = append(options, channel.WithTargetEndpoints(peers...))
	}

	client := channelConnection.channelClient

	var beforeInvokeData interface{}
	if hlf.beforeInvokeHandler != nil {
		beforeInvokeData, err = hlf.beforeInvokeHandler(channelID, chaincodeName, methodName, methodArgs, noBatch, peers...)
		if err != nil {
			return nil, err
		}
	}

	response, err := hlf.Request(
		methodArgs,
		chaincodeName,
		methodName,
		noBatch,
		client,
		client.Execute,
		options...,
	)

	if hlf.afterInvokeHandler != nil {
		err = hlf.afterInvokeHandler(beforeInvokeData, response, err, channelID, chaincodeName, methodName, methodArgs, noBatch, peers...)
		if err != nil {
			return nil, err
		}
	}

	return response, errors.WithStack(err)
}

type (
	// HlfBeforeInvokeHandler - handler before invoke
	HlfBeforeInvokeHandler func(channelID string, chaincodeName string, methodName string, methodArgs []string, noBatch bool, peers ...string) (interface{}, error)
	// HlfAfterInvokeHandler - handler after invoke
	HlfAfterInvokeHandler func(beforeInvokeData interface{}, r *channel.Response, invokeErr error, channelID string, chaincodeName string, methodName string, methodArgs []string, noBatch bool, peers ...string) error
)

// Request - method to send request to hlf
func (hlf *HLFClient) Request(
	methodArgs []string, chaincodeName string, methodName string, noBatch bool,
	client *channel.Client,
	requestFunc func(channel.Request, ...channel.RequestOption) (channel.Response, error),
	options ...channel.RequestOption,
) (*channel.Response, error) {
	var (
		err         error
		notifier    <-chan *fab.CCEvent
		batchWaiter sync.WaitGroup
	)

	if !noBatch {
		notifier, err = hlf.GetCCEventNotifier(client, chaincodeName, BatchExecuteEvent)
		if err != nil {
			logger.Error("failed RegisterChaincodeEvent", zap.Error(err))
			return nil, err
		}

		batchWaiter.Add(1)
		go func() {
			defer batchWaiter.Done()
			waitBatchAndPrintContents(notifier)
		}()
	}

	printInvokeArgs(methodArgs, chaincodeName, methodName)

	channelRequest := channel.Request{
		ChaincodeID: chaincodeName,
		Fcn:         methodName,
		Args:        AsBytes(methodArgs),
	}

	response, err := requestFunc(
		channelRequest,
		options...,
	)
	if err != nil {
		logger.Error("error", zap.Error(err))
		return nil, err
	}

	printResponse(&response)

	batchWaiter.Wait() // wait if noBatch==false

	return &response, nil
}

func printResponse(response *channel.Response) {
	logger.Debug("response",
		zap.String("TransactionID", string(response.TransactionID)))
	logger.Debug("payload", zap.ByteString("payload", response.Payload))

	logger.Debug("response.Responses[0].ProposalRespons",
		zap.ByteString("Payload", response.Responses[0].ProposalResponse.GetResponse().Payload),
		zap.Int32("ChaincodeStatus", response.Responses[0].ChaincodeStatus),
	)
}

func printInvokeArgs(methodArgs []string, chaincodeName string, methodName string) {
	logger.Debug("chaincodeName")
	logger.Debug(fmt.Sprintf("%v\n", chaincodeName))

	logger.Debug("methodName")
	logger.Debug(fmt.Sprintf("%v\n", methodName))

	logger.Debug("methodArgs")
	for i, arg := range methodArgs {
		logger.Debug(fmt.Sprintf("[%d]", i))
		logger.Debug(fmt.Sprintf("    - '%v'", arg))
	}
}

func waitBatchAndPrintContents(notifier <-chan *fab.CCEvent) {
	select {
	case ccEvent := <-notifier:
		event := &proto.BatchEvent{}
		if err := pb.Unmarshal(ccEvent.Payload, event); err != nil {
			logger.Error("unmarshaling error", zap.Error(err))
			return
		}

		logger.Debug("ccEvent.EventName", zap.String("ccEvent.EventName", ccEvent.EventName))
		if ccEvent.EventName == BatchExecuteEvent {
			logger.Debug("ccEvent.Payload", zap.ByteString("ccEvent.Payload", ccEvent.Payload))
			batchEvent := &proto.BatchEvent{}
			if err := pb.Unmarshal(ccEvent.Payload, batchEvent); err != nil {
				logger.Error("err", zap.Error(err))
				return
			}

			for _, event := range batchEvent.Events {
				if event.Error != nil {
					logger.Error("err",
						zap.String("event.Id", hex.EncodeToString(event.Id)),
						zap.Int32("event.Error.Code", event.Error.Code),
						zap.Error(errors.New(event.Error.GetError())),
					)

					continue
				}
			}
		}

		if ccEvent.EventName == KeyEvent || ccEvent.EventName == core.MultiSwapKeyEvent {
			logger.Debug("received key event [%s]", zap.ByteString("ccEvent.Payload", ccEvent.Payload))
			args := strings.Split(string(ccEvent.Payload), "\t")
			if len(args) < 3 {
				logger.Error("incorrect key event on channel %s with payload [%s]\n", zap.ByteString("ccEvent.Payload", ccEvent.Payload))
				return
			}

			toToken := strings.ToLower(args[0])
			swapID := args[1]
			swapkey := args[2]

			logger.Debug("args",
				zap.String("toToken", toToken),
				zap.String("swapID", swapID),
				zap.String("swapkey", swapkey),
			)
		}

		logger.Debug("payload", zap.ByteString("payload", ccEvent.Payload))

	case <-time.After(time.Second * 20):
		logger.Debug(fmt.Sprintf("Did NOT receive CC for eventId(%s)\n", BatchExecuteEvent))
	}
}

// GetCCEventNotifier - get notifier for chaincode event
func (hlf *HLFClient) GetCCEventNotifier(client *channel.Client, chaincodeName string, event string) (<-chan *fab.CCEvent, error) {
	key := chaincodeName + event
	notifier := hlf.NotifierChaincodeEvents[key]
	if notifier == nil {
		var err error

		// Register chaincode event (pass in channel which receives event details when the event is complete)
		_, notifier, err = client.RegisterChaincodeEvent(chaincodeName, event)
		if err != nil {
			logger.Error("failed RegisterChaincodeEvent", zap.Error(err))
			return nil, err
		}
		if notifier == nil {
			logger.Error("failed RegisterChaincodeEvent notifier can't be nil")
			return nil, err
		}

		hlf.NotifierChaincodeEvents[key] = notifier
	}

	return notifier, nil
}

// SignArgs - sign arguments for invoke
func (hlf *HLFClient) SignArgs(channelID string, chaincodeName string, methodName string, methodArgs []string, privateKey ed25519.PrivateKey, publicKey ed25519.PublicKey) ([]string, error) {
	logger.Debug("SignArgs")
	logger.Debug("--- methodArgs")
	for i, arg := range methodArgs {
		logger.Debug(fmt.Sprintf("%d\n", i))
		logger.Debug(fmt.Sprintf("%v\n", arg))
	}
	signedMessage, _, err := Sign(privateKey, publicKey, channelID, chaincodeName, methodName, methodArgs)
	logger.Debug("--- signedMessage")
	for i, arg := range methodArgs {
		logger.Debug(fmt.Sprintf("%d\n", i))
		logger.Debug(fmt.Sprintf("%v\n", arg))
	}
	if err != nil {
		return nil, err
	}

	return signedMessage, nil
}

// QueryBlockByTxID - return block by transaction id
func (hlf *HLFClient) QueryBlockByTxID(channelID string, transactionId string, peer string) (*common.Block, error) {
	channelConnection, err := hlf.getOrCreateChannelConnection(channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to getOrCreateChannelConnection: %s", err)
	}
	ledgerClient, err := ledger.New(channelConnection.channelProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create new ledger client: %s", err)
	}

	block, err := ledgerClient.QueryBlockByTxID(fab.TransactionID(transactionId), ledger.WithTargetEndpoints(peer))

	return block, nil
}

// ChaincodeVersion - only for admin user, get version for chaincode, firstly try to get version for 1.4, secondly try to get for 2.3 lifecycle
func (hlf *HLFClient) ChaincodeVersion(chaincode string, peer string) (string, error) {
	clientProvider := hlf.sdk.Context(hlf.ContextOptions...)
	client, err := resmgmt.New(clientProvider)
	if err != nil {
		return "", err
	}

	chaincodeVersion14, err := hlf.ChaincodeVersion14(client, chaincode, peer)
	if err == nil {
		return chaincodeVersion14, nil
	}

	chaincodeVersion23, err := hlf.ChaincodeVersion23Lifecycle(client, chaincode, peer)
	if err == nil {
		return chaincodeVersion23, nil
	}

	return "", fmt.Errorf("chaincode %s in channel %s not found", chaincode, chaincode)
}

// ChaincodeVersion14 - only for admin user, get version for chaincode for 1.4 hlf
func (hlf *HLFClient) ChaincodeVersion14(client *resmgmt.Client, chaincode string, peer string) (string, error) {
	chaincodeQueryResponse, err := client.QueryInstalledChaincodes(resmgmt.WithTargetEndpoints(peer))
	if err != nil {
		return "", err
	}

	for _, a := range chaincodeQueryResponse.Chaincodes {
		if a.Name == chaincode {
			return a.GetVersion(), nil
		}
	}

	return "", fmt.Errorf("chaincode %s in channel %s not found", chaincode, chaincode)
}

// ChaincodeVersion23Lifecycle - only for admin user, get version for Committed chaincode for 2.3 hlf - lifecycle
func (hlf *HLFClient) ChaincodeVersion23Lifecycle(client *resmgmt.Client, chaincode string, peer string) (string, error) {
	lifecycleQueryCommittedCC, err := client.LifecycleQueryCommittedCC(chaincode, resmgmt.LifecycleQueryCommittedCCRequest{Name: chaincode}, resmgmt.WithTargetEndpoints(peer))
	if err != nil {
		return "", err
	}

	for _, a := range lifecycleQueryCommittedCC {
		if a.Name == chaincode {
			return a.Version, nil
		}
	}

	return "", fmt.Errorf("chaincode %s in channel %s not found", chaincode, chaincode)
}

// GetBlockchainInfo queries and return information about ledger (height, current block hash, and previous block hash) from peer.
func (hlf *HLFClient) GetBlockchainInfo(channelID string, peer string) (*fab.BlockchainInfoResponse, error) {
	channelConnection, err := hlf.getOrCreateChannelConnection(channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to getOrCreateChannelConnection: %s", err)
	}
	ledgerClient, err := ledger.New(channelConnection.channelProvider)
	blockchainInfoResponse, err := ledgerClient.QueryInfo(ledger.WithTargetEndpoints(peer))
	if err != nil {
		return nil, err
	}

	return blockchainInfoResponse, err
}

// GetTransactionByID - return transaction by id from peer
func (hlf *HLFClient) GetTransactionByID(channelID string, transactionID string, peer string) (*peer.ProcessedTransaction, error) {
	channelConnection, err := hlf.getOrCreateChannelConnection(channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to getOrCreateChannelConnection: %s", err)
	}
	ledgerClient, err := ledger.New(channelConnection.channelProvider)
	processedTransaction, err := ledgerClient.QueryTransaction(fab.TransactionID(transactionID), ledger.WithTargetEndpoints(peer))
	if err != nil {
		return nil, err
	}

	return processedTransaction, err
}

// QueryBlock - return block by id from peer
func (hlf *HLFClient) QueryBlock(channelID string, blockID string, endpoints string) (*common.Block, error) {
	channelConnection, err := hlf.getOrCreateChannelConnection(channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to getOrCreateChannelConnection: %s", err)
	}
	ledgerClient, err := ledger.New(channelConnection.channelProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create new ledger client: %s", err)
	}

	blockIDUint, err := strconv.ParseUint(blockID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("Parse blockID: %s", err)
	}
	block, err := ledgerClient.QueryBlock(blockIDUint, ledger.WithTargetEndpoints(endpoints))

	return block, nil
}

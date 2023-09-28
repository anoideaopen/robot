package common

import (
	"fmt"
	"os"
	"sync"

	"github.com/atomyze-foundation/cartridge"
	"github.com/atomyze-foundation/cartridge/manager"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

// NewInstanceSDK returns a new instance of the fabric-sdk-go
func NewInstanceSDK(connectionYaml string, user string, vaultConfig *VaultConfig) *InstanceSDK {
	theSDK := &InstanceSDK{
		connectionProfilePath: connectionYaml,
		user:                  user,
		vaultConfig:           vaultConfig,
	}

	theSDK.Do(func() {
		if err := theSDK.init(); err != nil {
			panic(fmt.Errorf("initialization fabric sdk : %w", err))
		}
	})

	return theSDK
}

// ChannelClient returns a channel client for the specified channel
func (ins *InstanceSDK) ChannelClient(channel string) (*channel.Client, error) {
	if !ins.initialized {
		panic(fmt.Errorf("fabric sdk not initialized"))
	}

	return ins.channelClient(channel)
}

// Channels returns a list of channels
func (ins *InstanceSDK) Channels() []string {
	return ins.channels
}

// InstanceSDK is a wrapper for the fabric-sdk-go
type InstanceSDK struct {
	connectionProfilePath string
	user                  string
	vaultConfig           *VaultConfig

	mspID          string
	org            string
	channels       []string
	connectOpts    []fabsdk.Option
	contextOpts    []fabsdk.ContextOption
	configProvider core.ConfigProvider
	vaultManager   *manager.VaultManager
	initialized    bool

	sync.Once
}

func (ins *InstanceSDK) init() error {
	if ins.connectionProfilePath == "" {
		return fmt.Errorf("connection.yaml undefined")
	}

	if _, err := os.Stat(ins.connectionProfilePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("connection.yaml : %w", err)
		}
	}

	if ins.user == "" {
		return fmt.Errorf("hlf user undefined")
	}

	ins.configProvider = config.FromFile(ins.connectionProfilePath)

	ins.contextOpts = []fabsdk.ContextOption{fabsdk.WithOrg(ins.org)}

	if ins.vaultConfig.Address == "" {
		ins.contextOpts = append(ins.contextOpts, fabsdk.WithUser(ins.user))

		ins.initialized = true

		return nil
	}

	backends, err := ins.configMeta()
	if err != nil {
		return err
	}

	ins.vaultManager, err = manager.NewVaultManager(
		ins.mspID,
		ins.vaultConfig.UserCertName,
		ins.vaultConfig.Address,
		ins.vaultConfig.Token,
		ins.vaultConfig.Path,
	)
	if err != nil {
		return err
	}

	connector := cartridge.NewConnector(ins.vaultManager, cartridge.NewVaultConnectProvider(backends...))
	if connector != nil {
		ins.connectOpts, err = connector.Opts()
		if err != nil {
			return err
		}
	}

	ins.contextOpts = append(ins.contextOpts, fabsdk.WithIdentity(ins.vaultManager.SigningIdentity()))
	ins.initialized = true

	return nil
}

func (ins *InstanceSDK) channelClient(channelID string) (*channel.Client, error) {
	sdk, err := fabsdk.New(ins.configProvider, ins.connectOpts...)
	if err != nil {
		return nil, fmt.Errorf("initializes the SDK : %w", err)
	}
	defer sdk.Close()

	client, err := channel.New(sdk.ChannelContext(channelID, ins.contextOpts...))
	if err != nil {
		return nil, fmt.Errorf("failed to create new channel client : %w", err)
	}

	return client, nil
}

func (ins *InstanceSDK) configMeta() ([]core.ConfigBackend, error) {
	backends, err := ins.configProvider()
	if err != nil {
		return nil, err
	}

	value, ok := backends[0].Lookup("client.organization")
	if !ok {
		return nil, fmt.Errorf("no client organization defined in the config")
	}

	ins.org = value.(string)

	value, ok = backends[0].Lookup("organizations." + ins.org + ".mspid")
	if !ok {
		return nil, fmt.Errorf("no client organization defined in the config")
	}

	ins.mspID = value.(string)

	channelsIface, ok := backends[0].Lookup("channels")
	if !ok {
		return nil, fmt.Errorf("failed to find channels in connection profile")
	}

	channelsMap, ok := channelsIface.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to parse connection profile")
	}

	ins.channels = nil

	for ch := range channelsMap {
		if ch == "" || ch == "_default" {
			continue
		}

		ins.channels = append(ins.channels, ch)
	}

	return backends, nil
}

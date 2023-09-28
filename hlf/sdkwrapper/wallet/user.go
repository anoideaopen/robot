package wallet

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/atomyze-foundation/foundation/proto"
	"github.com/atomyze-foundation/robot/hlf/sdkwrapper/logger"
	"github.com/atomyze-foundation/robot/hlf/sdkwrapper/service"
	"github.com/btcsuite/btcutil/base58"
	pb "github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
	"go.uber.org/zap"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"
)

// InvokeResponse is a response from invoke
type InvokeResponse struct {
	Event    *fab.TxStatusEvent
	Response []byte
}

// User is a user for invoke
type User struct {
	// ID is a user ID for invoke
	ID string
	// PublicKey is a public key for invoke
	PublicKey ed25519.PublicKey
	// SecretKey is a secret key for invoke
	SecretKey ed25519.PrivateKey
	config    core.ConfigProvider
	wallet    *gateway.Wallet
	backend   *gateway.Gateway
	robot     *gateway.Gateway
	isBackend bool
	sem       chan struct{}
}

// Option is a user option for invoke
type Option func(u *User) error

// NewUser is a constructor for user
func NewUser(msp, backendCert, backendKey, connection, id, privateKey string, opts ...Option) (*User, error) {
	user := &User{
		ID: id,
	}

	for _, opt := range opts {
		if err := opt(user); err != nil {
			return nil, err
		}
	}

	if user.backend == nil {
		certData, err := os.Open(backendCert)
		if err != nil {
			return nil, err
		}
		cert, err := ioutil.ReadAll(certData)
		certData.Close()
		if err != nil {
			return nil, err
		}
		keyData, err := os.Open(backendKey)
		if err != nil {
			return nil, err
		}
		key, err := ioutil.ReadAll(keyData)
		keyData.Close()
		if err != nil {
			return nil, err
		}

		user.wallet = gateway.NewInMemoryWallet()
		if err = user.wallet.Put("user", gateway.NewX509Identity(msp, string(cert), string(key))); err != nil {
			return nil, err
		}

		user.config = config.FromFile(filepath.Clean(connection))

		backend, err := gateway.Connect(gateway.WithConfig(user.config), gateway.WithIdentity(user.wallet, "user"))
		if err != nil {
			return nil, err
		}

		user.backend = backend
	}

	if !user.isBackend {
		sKey, pKey, err := NewKeyPair(privateKey)
		if err != nil {
			return nil, err
		}
		user.SecretKey = sKey
		user.PublicKey = pKey

		return user, user.AddUser()
	}

	return user, nil
}

// NewKeyPair is a constructor for key pair
func NewKeyPair(private string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	if private == "" {
		pKey, sKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		return sKey, pKey, nil
	}

	decoded, ver, err := base58.CheckDecode(private)
	if err != nil {
		return nil, nil, err
	}
	privateKey := ed25519.PrivateKey(append([]byte{ver}, decoded...))

	pk := privateKey.Public().(ed25519.PublicKey)
	return privateKey, pk, nil
}

// Base58Pk is a base58 public key
func (u *User) Base58Pk() string {
	if err := checkIsNotBackend(u); err != nil {
		return ""
	}
	return base58.Encode(u.PublicKey)
}

// ChangeKeys is a method for changing keys
func (u *User) ChangeKeys(secretKey ed25519.PrivateKey, publicKey ed25519.PublicKey) {
	u.SecretKey = secretKey
	u.PublicKey = publicKey
}

// Addr is a method for getting address
func (u *User) Addr() string {
	if err := checkIsNotBackend(u); err != nil {
		return ""
	}
	hash := sha3.Sum256(u.PublicKey)
	return base58.CheckEncode(hash[1:], hash[0])
}

// AddrFromKey is a method for getting address from key
func (u *User) AddrFromKey(key ed25519.PublicKey) string {
	if err := checkIsNotBackend(u); err != nil {
		return ""
	}
	hash := sha3.Sum256(key)
	return base58.CheckEncode(hash[1:], hash[0])
}

// Sign is a method for signing
func (u *User) Sign(msg []byte) []byte {
	if err := checkIsNotBackend(u); err != nil {
		return nil
	}
	return ed25519.Sign(u.SecretKey, msg)
}

// AddUser is a method for adding user
func (u *User) AddUser() error {
	n, err := u.backend.GetNetwork("acl")
	if err != nil {
		return err
	}

	tx, err := n.GetContract("acl").CreateTransaction("addUser")
	if err != nil {
		return err
	}

	_, err = tx.Submit(append([]string{}, u.Base58Pk(), "kychash", u.ID, "true")...)

	logger.Info("add to acl", zap.String("ID", u.ID))
	if err != nil && strings.Contains(err.Error(), "exists") {
		logger.Info("already exists in ACL, skipping", zap.String("ID", u.ID))
	} else if err != nil {
		return err
	}

	return nil
}

// Invoke is a method for invoke
func (u *User) Invoke(ch, cc, fn string, args ...string) (*InvokeResponse, error) {
	n, err := u.backend.GetNetwork(ch)
	if err != nil {
		return nil, err
	}

	contract := n.GetContract(cc)
	tx, err := contract.CreateTransaction(fn)
	if err != nil {
		return nil, err
	}

	resp, err := tx.Submit(args...)
	if err != nil {
		return nil, err
	}

	return &InvokeResponse{
		Response: resp,
	}, err
}

// InvokeWithListener is a method for invoke with listener
func (u *User) InvokeWithListener(ch, cc, fn string, args ...string) (*InvokeResponse, error) {
	var (
		n   *gateway.Network
		err error
	)
	if u.robot == nil {
		n, err = u.backend.GetNetwork(ch)
	} else {
		n, err = u.robot.GetNetwork(ch)
	}
	if err != nil {
		return nil, err
	}

	contract := n.GetContract(cc)

	tx, err := contract.CreateTransaction(fn)
	if err != nil {
		return nil, err
	}

	listener := tx.RegisterCommitEvent()

	resp, err := tx.Submit(args...)
	if err != nil {
		return nil, err
	}

	event := <-listener

	return &InvokeResponse{
		Event:    event,
		Response: resp,
	}, err
}

// SignedInvoke is a method for signed invoke
func (u *User) SignedInvoke(nonce *int64, ch, cc, fn string, args ...string) (*InvokeResponse, error) {
	if err := checkIsNotBackend(u); err != nil {
		return nil, err
	}

	if u.sem != nil {
		u.sem <- struct{}{}
	}

	var newnonce int64

	if nonce == nil || *nonce == 0 {
		newnonce = service.GetNonceInt64()
	} else {
		newnonce = atomic.AddInt64(nonce, 1)
	}
	if u.sem != nil {
		<-u.sem
	}

	n := strconv.FormatInt(newnonce, 10)

	result := append(append([]string{fn, "", ch, cc}, args...), n, base58.Encode(u.PublicKey))
	message := sha3.Sum256([]byte(strings.Join(result, "")))
	newArgs := append(result[1:], base58.Encode(u.Sign(message[:])))
	resp, err := u.InvokeWithListener(ch, cc, fn, newArgs...)
	if err != nil {
		return nil, err
	}

	if u.robot != nil {
		time.Sleep(1000 * time.Millisecond)
		binaryTx, err := hex.DecodeString(resp.Event.TxID)
		if err != nil {
			return nil, err
		}

		b := &proto.Batch{
			TxIDs: [][]byte{binaryTx},
		}
		e, batchResp, err := u.SendBatch(ch, cc, b)
		if err != nil {
			return nil, err
		}

		for _, txResp := range batchResp.TxResponses {
			if bytes.Equal(txResp.Id, binaryTx) {
				logger.Info("batch",
					zap.String("channel", cc),
					zap.String("tx ID", e.TxID),
					zap.Int32("validation code:", int32(e.TxValidationCode)),
					zap.String("batch ID", hex.EncodeToString(txResp.Id)),
					zap.Any("writes", txResp.GetWrites()),
					zap.Any("error", txResp.Error),
				)
				return &InvokeResponse{
					Event:    resp.Event,
					Response: resp.Response,
				}, nil
			}
		}
		return nil, fmt.Errorf("batch doesn't contain response for transaction")
	}

	return &InvokeResponse{
		Event:    resp.Event,
		Response: resp.Response,
	}, nil
}

// SignedInvokeAsync is a method for signed invoke async
func (u *User) SignedInvokeAsync(ch, cc, fn string, args ...string) (*InvokeResponse, error) {
	if err := checkIsNotBackend(u); err != nil {
		return nil, err
	}

	nonce := strconv.FormatInt(time.Now().UnixNano()/1000000, 10)
	result := append(append([]string{fn, "", ch, cc}, args...), nonce, base58.Encode(u.PublicKey))
	message := sha3.Sum256([]byte(strings.Join(result, "")))
	newArgs := append(result[1:], base58.Encode(u.Sign(message[:])))
	resp, err := u.Invoke(ch, cc, fn, newArgs...)
	if err != nil {
		return nil, err
	}

	return &InvokeResponse{
		Event:    nil,
		Response: resp.Response,
	}, nil
}

// Query is a method for query
func (u *User) Query(ch, cc, fn string, args ...string) ([]byte, error) {
	var (
		n   *gateway.Network
		err error
	)
	if u.robot == nil {
		n, err = u.backend.GetNetwork(ch)
	} else {
		n, err = u.robot.GetNetwork(ch)
	}
	if err != nil {
		return nil, err
	}
	t, err := n.GetContract(cc).CreateTransaction(fn)

	return t.Evaluate(args...)
}

// BalanceShouldBe is a method for balance should be check
func (u *User) BalanceShouldBe(expected uint64) error {
	expectedBalance := "\"" + strconv.FormatUint(expected, 10) + "\""
	var (
		balance string
		n       int = 0
	)
	for balance == "" && n < 10 {
		tokenBalanceResponse, err := u.Query("fiat", "fiat", "balanceOf", u.Addr())
		if err != nil {
			break
		}
		tokenBalanceString := string(tokenBalanceResponse)
		if expectedBalance == tokenBalanceString {
			balance = tokenBalanceString
			break
		} else {
			err = fmt.Errorf("expected balance: %d, got: %s", expected, tokenBalanceString)
			time.Sleep(500 * time.Millisecond)
		}
		n++
	}

	if expectedBalance != balance {
		return fmt.Errorf("expected %s, got %s", expectedBalance, balance)
	}
	return nil
}

// SwapAnswerAndDone is a method for swap answer and done
func (u *User) SwapAnswerAndDone(ch, cc, swapId string, swapKey string) error {
	swapBytes, err := u.Query(ch, cc, "swapGet", swapId)
	if err != nil {
		return err
	}

	swap := &proto.Swap{}
	if err = json.Unmarshal(swapBytes, swap); err != nil {
		return err
	}
	swap.Creator = []byte{}

	batchForTargetChan := &proto.Batch{
		Swaps: []*proto.Swap{swap},
	}
	to := strings.ToLower(swap.To)
	txEvent, batchResp, err := u.SendBatch(to, to, batchForTargetChan)
	if err != nil {
		return err
	}

	found := false
	for _, swapResp := range batchResp.SwapResponses {
		swapRespId := hex.EncodeToString(swapResp.Id)
		if swapRespId == swapId {
			logger.Info("batch",
				zap.String("to", to),
				zap.String("txEvent.TxID", txEvent.TxID),
				zap.Int32("txEvent.TxValidationCode", int32(txEvent.TxValidationCode)),
				zap.String("swapRespId", swapRespId),
				zap.Any("swapResp.Writes", swapResp.Writes),
				zap.Any("swapResp.Error", swapResp.Error),
			)
			found = true
		}
	}
	if !found {
		return fmt.Errorf("batch doesn't contain response for swap response")
	}

	time.Sleep(time.Millisecond * 1000)

	if _, err := u.Invoke(to, to, "swapDone", swapId, swapKey); err != nil {
		return err
	}

	binarySwapId, err := hex.DecodeString(swapId)
	if err != nil {
		return err
	}

	time.Sleep(time.Millisecond * 1000)
	batchForOriginChan := &proto.Batch{
		Keys: []*proto.SwapKey{{Id: binarySwapId, Key: swapKey}},
	}

	txEvent, batchResp, err = u.SendBatch(ch, cc, batchForOriginChan)
	if err != nil {
		return err
	}

	for _, swapResp := range batchResp.SwapKeyResponses {
		swapRespId := hex.EncodeToString(swapResp.Id)
		if swapRespId == swapId {
			logger.Info("batch",
				zap.String("channel", cc),
				zap.String("txEvent.TxID", txEvent.TxID),
				zap.Int32("txEvent.TxValidationCode", int32(txEvent.TxValidationCode)),
				zap.String("swapRespId", swapRespId),
				zap.Any("swapResp.Writes", swapResp.Writes),
				zap.Any("swapResp.Error", swapResp.Error),
			)
			return nil
		}
	}

	return fmt.Errorf("batch doesn't contain response for swap key")
}

// SendBatch is a method for send batch of transactions
func (u *User) SendBatch(ch, cc string, batch *proto.Batch) (*fab.TxStatusEvent, *proto.BatchResponse, error) {
	data, err := pb.Marshal(batch)
	if err != nil {
		return nil, nil, err
	}
	resp, err := u.InvokeWithListener(ch, cc, "batchExecute", string(data))
	if err != nil {
		return nil, nil, err
	}
	result := &proto.BatchResponse{}
	if err = pb.Unmarshal(resp.Response, result); err != nil {
		return nil, nil, err
	}
	return resp.Event, result, nil
}

func checkIsNotBackend(u *User) error {
	if u == nil {
		return fmt.Errorf("user can't be nil")
	}
	if u.isBackend {
		return fmt.Errorf("backend is not allowed to call this method")
	}
	return nil
}

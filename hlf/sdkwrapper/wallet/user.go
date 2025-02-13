package wallet

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha3"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/robot/hlf/sdkwrapper/logger"
	"github.com/anoideaopen/robot/hlf/sdkwrapper/service"
	"github.com/btcsuite/btcutil/base58"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/status"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
	"go.uber.org/zap"
	pb "google.golang.org/protobuf/proto"
)

const (
	retryCount = 20
	retrySleep = time.Millisecond * 500
)

type InvokeResponse struct {
	Event    *fab.TxStatusEvent
	Response []byte
}

type User struct {
	ID        string
	PublicKey ed25519.PublicKey
	SecretKey ed25519.PrivateKey
	config    core.ConfigProvider
	wallet    *gateway.Wallet
	backend   *gateway.Gateway
	robot     *gateway.Gateway
	isBackend bool
	sem       chan struct{}
}

type Option func(u *User) error

func NewUser(msp, backendCert, backendKey, connection, id, privateKey string, opts ...Option) (*User, error) {
	user := &User{
		ID: id,
	}

	for _, opt := range opts {
		if err := opt(user); err != nil {
			return nil, err
		}
	}

	if user.backend == nil { //nolint:nestif
		certData, err := os.Open(backendCert)
		if err != nil {
			return nil, err
		}
		cert, err := io.ReadAll(certData)
		certData.Close()
		if err != nil {
			return nil, err
		}
		keyData, err := os.Open(backendKey)
		if err != nil {
			return nil, err
		}
		key, err := io.ReadAll(keyData)
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

	pk, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok {
		return nil, nil, errors.New("invalid private key")
	}
	return privateKey, pk, nil
}

func (u *User) Base58Pk() string {
	if err := checkIsNotBackend(u); err != nil {
		return ""
	}
	return base58.Encode(u.PublicKey)
}

func (u *User) ChangeKeys(secretKey ed25519.PrivateKey, publicKey ed25519.PublicKey) {
	u.SecretKey = secretKey
	u.PublicKey = publicKey
}

func (u *User) Addr() string {
	if err := checkIsNotBackend(u); err != nil {
		return ""
	}
	hash := sha3.Sum256(u.PublicKey)
	return base58.CheckEncode(hash[1:], hash[0])
}

func (u *User) AddrFromKey(key ed25519.PublicKey) string {
	if err := checkIsNotBackend(u); err != nil {
		return ""
	}
	hash := sha3.Sum256(key)
	return base58.CheckEncode(hash[1:], hash[0])
}

func (u *User) Sign(msg []byte) []byte {
	if err := checkIsNotBackend(u); err != nil {
		return nil
	}
	return ed25519.Sign(u.SecretKey, msg)
}

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
	var newArgs []string
	newArgs = append(append(newArgs, result[1:]...), base58.Encode(u.Sign(message[:])))
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

		for _, txResp := range batchResp.GetTxResponses() {
			if bytes.Equal(txResp.GetId(), binaryTx) {
				logger.Info("batch",
					zap.String("channel", cc),
					zap.String("tx ID", e.TxID),
					zap.Int32("validation code:", int32(e.TxValidationCode)),
					zap.String("batch ID", hex.EncodeToString(txResp.GetId())),
					zap.Any("writes", txResp.GetWrites()),
					zap.Any("error", txResp.GetError()),
				)
				return &InvokeResponse{
					Event:    resp.Event,
					Response: resp.Response,
				}, nil
			}
		}
		return nil, errors.New("batch doesn't contain response for transaction")
	}

	return &InvokeResponse{
		Event:    resp.Event,
		Response: resp.Response,
	}, nil
}

func (u *User) SignedInvokeAsync(ch, cc, fn string, args ...string) (*InvokeResponse, error) {
	if err := checkIsNotBackend(u); err != nil {
		return nil, err
	}

	nonce := strconv.FormatInt(time.Now().UnixNano()/1000000, 10)
	result := append(append([]string{fn, "", ch, cc}, args...), nonce, base58.Encode(u.PublicKey))
	message := sha3.Sum256([]byte(strings.Join(result, "")))
	var newArgs []string
	newArgs = append(append(newArgs, result[1:]...), base58.Encode(u.Sign(message[:])))
	resp, err := u.Invoke(ch, cc, fn, newArgs...)
	if err != nil {
		return nil, err
	}

	return &InvokeResponse{
		Event:    nil,
		Response: resp.Response,
	}, nil
}

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
	if err != nil {
		return nil, err
	}
	// ToDo temporarily
	/*
		gateway.WithEvaluateRequestOptions(
			channel.WithRetry(evaluateRetryOpts),
		),*/

	return t.Evaluate(args...)
}

func (u *User) QueryWithRetryIfEndorsementMismatch(ch, cc, fn string, args ...string) ([]byte, error) {
	var (
		n   *gateway.Network
		r   []byte
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
	if err != nil {
		return nil, err
	}

	for range retryCount {
		r, err = t.Evaluate(args...)
		if err != nil {
			if isEndorsementMismatchErr(err) {
				time.Sleep(retrySleep)
				continue
			}
			return r, err
		}
		return r, err
	}
	return r, err
}

func (u *User) BalanceShouldBe(expected uint64) error {
	expectedBalance := "\"" + strconv.FormatUint(expected, 10) + "\""
	var balance string
	for range 10 {
		tokenBalanceResponse, err := u.QueryWithRetryIfEndorsementMismatch("fiat", "fiat", "balanceOf", u.Addr())
		if err != nil {
			break
		}
		tokenBalanceString := string(tokenBalanceResponse)
		if expectedBalance == tokenBalanceString {
			balance = tokenBalanceString
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if expectedBalance != balance {
		return fmt.Errorf("expected %s, got %s", expectedBalance, balance)
	}
	return nil
}

func (u *User) SwapAnswerAndDone(ch, cc, swapID string, swapKey string) error {
	swapBytes, err := u.Query(ch, cc, "swapGet", swapID)
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
	to := strings.ToLower(swap.GetTo())
	txEvent, batchResp, err := u.SendBatch(to, to, batchForTargetChan)
	if err != nil {
		return err
	}

	found := false
	for _, swapResp := range batchResp.GetSwapResponses() {
		swapRespID := hex.EncodeToString(swapResp.GetId())
		if swapRespID == swapID {
			logger.Info("batch",
				zap.String("to", to),
				zap.String("txEvent.TxID", txEvent.TxID),
				zap.Int32("txEvent.TxValidationCode", int32(txEvent.TxValidationCode)),
				zap.String("swapRespId", swapRespID),
				zap.Any("swapResp.Writes", swapResp.GetWrites()),
				zap.Any("swapResp.Error", swapResp.GetError()),
			)
			found = true
		}
	}
	if !found {
		return errors.New("batch doesn't contain response for swap response")
	}

	time.Sleep(time.Millisecond * 1000)

	if _, err := u.Invoke(to, to, "swapDone", swapID, swapKey); err != nil {
		return err
	}

	binarySwapID, err := hex.DecodeString(swapID)
	if err != nil {
		return err
	}

	time.Sleep(time.Millisecond * 1000)
	batchForOriginChan := &proto.Batch{
		Keys: []*proto.SwapKey{{Id: binarySwapID, Key: swapKey}},
	}

	txEvent, batchResp, err = u.SendBatch(ch, cc, batchForOriginChan)
	if err != nil {
		return err
	}

	for _, swapResp := range batchResp.GetSwapKeyResponses() {
		swapRespID := hex.EncodeToString(swapResp.GetId())
		if swapRespID == swapID {
			logger.Info("batch",
				zap.String("channel", cc),
				zap.String("txEvent.TxID", txEvent.TxID),
				zap.Int32("txEvent.TxValidationCode", int32(txEvent.TxValidationCode)),
				zap.String("swapRespId", swapRespID),
				zap.Any("swapResp.Writes", swapResp.GetWrites()),
				zap.Any("swapResp.Error", swapResp.GetError()),
			)
			return nil
		}
	}

	return errors.New("batch doesn't contain response for swap key")
}

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
		return errors.New("user can't be nil")
	}
	if u.isBackend {
		return errors.New("backend is not allowed to call this method")
	}
	return nil
}

func isEndorsementMismatchErr(err error) bool {
	if err == nil {
		return false
	}
	s, ok := status.FromError(err)
	return ok && status.Code(s.Code) == status.EndorsementMismatch
}

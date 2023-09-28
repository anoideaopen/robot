package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/atomyze-foundation/robot/hlf/sdkwrapper/logger"
	"github.com/btcsuite/btcutil/base58"
	"go.uber.org/zap"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"
)

// SignMessage - sign message by private key
func SignMessage(signerInfo SignerInfo, result []string) ([]byte, [32]byte, error) {
	message := sha3.Sum256([]byte(strings.Join(result, "")))
	sig := ed25519.Sign(signerInfo.PrivateKey, message[:])
	if !ed25519.Verify(signerInfo.PublicKey, message[:], sig) {
		err := fmt.Errorf("valid signature rejected")
		logger.Error("ed25519.Verify", zap.Error(err))
		return nil, message, err
	}
	return sig, message, nil
}

// Sign - sign message by private key
func Sign(privateKey ed25519.PrivateKey, publicKey ed25519.PublicKey, channel string, chaincode string, methodName string, args []string) ([]string, string, error) {
	nonce := GetNonce()
	result := append(append([]string{methodName, "", chaincode, channel}, args...), nonce, ConvertPublicKeyToBase58(publicKey))

	logger.Debug(
		"For sign",
		zap.Strings("result", result),
	)

	signerInfo := SignerInfo{PublicKey: publicKey, PrivateKey: privateKey}
	signMessage, message, err := SignMessage(signerInfo, result)
	if err != nil {
		return nil, "", err
	}

	messageWithSig := append(result[1:], base58.Encode(signMessage))
	hash := hex.EncodeToString(message[:])

	logger.Debug(
		"Sign result",
		zap.Strings("messageWithSig", messageWithSig),
		zap.String("hash", hash),
	)

	return messageWithSig, hash, nil
}

// SignerInfo - struct for sign message
type SignerInfo struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

// GenerateMessage - generate message for sign
func GenerateMessage(validatorPublicKeys []string, channelID string, chaincodeName string, methodName string, args []string) string {
	requestId := ""
	nonce := GetNonce()
	result := append(append([]string{methodName, requestId, chaincodeName, channelID}, args...), nonce)
	for _, publicKey := range validatorPublicKeys {
		result = append(result, publicKey)
	}

	logger.Debug(
		"For sign",
		zap.Strings("result", result),
	)

	return strings.Join(result, "\n")
}

// EncodedPrivKeyToEd25519 - get private key type Ed25519 by encoded private key in string
// secretKey string - private key in base58check, base58 or hex
func EncodedPrivKeyToEd25519(secretKey string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	privateKey, publicKey, err := GetPrivateKeySKFromBase58Check(secretKey)
	if err != nil {
		privateKey, publicKey, err = GetPrivateKeySKFromHex(secretKey)
		if err != nil {
			privateKey, publicKey, err = GetPrivateKeySKFromBase58(secretKey)
		}
	}

	return privateKey, publicKey, err
}

// GetPrivateKeySKFromHex - get private key type Ed25519 by string - hex encoded private key
// secretKey string - private key in hex
func GetPrivateKeySKFromHex(secretKey string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	privateKey, err := hex.DecodeString(secretKey)
	if err != nil {
		return nil, nil, err
	}

	return privateKey, ed25519.PrivateKey(privateKey).Public().(ed25519.PublicKey), nil
}

// GetPrivateKeySKFromBase58 - get private key type Ed25519 by string - Base58 encoded private key
// secretKey string - private key in Base58
func GetPrivateKeySKFromBase58(secretKey string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	privateKey := base58.Decode(secretKey)
	return privateKey, ed25519.PrivateKey(privateKey).Public().(ed25519.PublicKey), nil
}

// GetPrivateKeySKFromBase58Check - get private key type Ed25519 by string - Base58Check encoded private key
// secretKey string - private key in Base58Check
func GetPrivateKeySKFromBase58Check(secretKey string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	decode, ver, err := base58.CheckDecode(secretKey)
	if err != nil {
		return nil, nil, err
	}
	privateKey := ed25519.PrivateKey(append([]byte{ver}, decode...))
	return privateKey, privateKey.Public().(ed25519.PublicKey), nil
}

// GetAddress - get address by encoded string in standard encoded for project is 'base58.Check'
// secretKey string - private key in base58check, or hex or base58
func GetAddress(secretKey string) (string, error) {
	var publicKey ed25519.PublicKey
	var err error

	_, publicKey, err = EncodedPrivKeyToEd25519(secretKey)
	if err != nil {
		return "", err
	}

	return GetAddressByPublicKey(publicKey)
}

// GetAddressByPublicKey - get address by encoded string in standard encoded for project is 'base58.Check'
// secretKey string - private key in base58check, or hex or base58
func GetAddressByPublicKey(publicKey ed25519.PublicKey) (string, error) {
	if len(publicKey) == 0 {
		return "", errors.New("publicKey can't be empty")
	}

	hash := sha3.Sum256(publicKey)
	return base58.CheckEncode(hash[1:], hash[0]), nil
}

// GetBase58PubKey returns base58-encoded pubkey.
// secretKey string - private key in base58check, or hex or base58
func GetBase58PubKey(secretKey string) (string, error) {
	var publicKey ed25519.PublicKey
	var err error

	_, publicKey, err = EncodedPrivKeyToEd25519(secretKey)
	if err != nil {
		return "", err
	}

	return base58.Encode(publicKey), nil
}

// GeneratePrivateAndPublicKey - generate private and public key
func GeneratePrivateAndPublicKey() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	return publicKey, privateKey, err
}

// GeneratePrivateKey - generate private key
func GeneratePrivateKey() (string, error) {
	_, privateKey, err := GeneratePrivateAndPublicKey()
	if err != nil {
		return "", err
	}

	return ConvertPrivateKeyToBase58Check(privateKey), nil
}

// SignACL - sign message by private key
func SignACL(signerInfoArray []SignerInfo, methodName string, address string, reason string, reasonId string, newPkey string) ([]string, string, error) {
	nonce := GetNonce()
	result := []string{methodName, address, reason, reasonId, newPkey, nonce}
	for _, signerInfo := range signerInfoArray {
		result = append(result, ConvertPublicKeyToBase58(signerInfo.PublicKey))
	}

	logger.Debug(
		"For sign",
		zap.Strings("result", result),
	)

	message := sha3.Sum256([]byte(strings.Join(result, "")))

	signatures := make([]string, 0)
	for _, signerInfo := range signerInfoArray {
		sig := ed25519.Sign(signerInfo.PrivateKey, message[:])
		if !ed25519.Verify(signerInfo.PublicKey, message[:], sig) {
			err := fmt.Errorf("valid signature rejected")
			logger.Error("ed25519.Verify", zap.Error(err))
			return nil, "", err
		}
		signatures = append(signatures, hex.EncodeToString(sig))
	}

	messageWithSig := append(result[1:], signatures...)
	hash := hex.EncodeToString(message[:])

	logger.Debug(
		"Sign result",
		zap.Strings("messageWithSig", messageWithSig),
		zap.String("hash", hash),
	)

	return messageWithSig, hash, nil
}

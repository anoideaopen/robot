package service

import (
	"crypto/ed25519"
	"strconv"
	"time"

	"github.com/btcsuite/btcutil/base58"
)

// AsBytes converts a slice of strings to a slice of byte slices.
func AsBytes(args []string) [][]byte {
	bytes := make([][]byte, len(args))
	for i, arg := range args {
		bytes[i] = []byte(arg)
	}
	return bytes
}

// NowMillisecond returns current time in milliseconds
func NowMillisecond() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// GetNonce - get nonce for transaction
func GetNonce() string {
	return strconv.FormatInt(NowMillisecond(), 10)
}

// GetNonceInt64 - get nonce for transaction
func GetNonceInt64() int64 {
	return NowMillisecond()
}

// ConvertPrivateKeyToBase58Check - use privateKey with standard encoded type - Base58Check
func ConvertPrivateKeyToBase58Check(privateKey ed25519.PrivateKey) string {
	hash := []byte(privateKey)
	encoded := base58.CheckEncode(hash[1:], hash[0])
	return encoded
}

// ConvertPublicKeyToBase58 - use publicKey with standard encoded type - Base58
func ConvertPublicKeyToBase58(publicKey ed25519.PublicKey) string {
	encoded := base58.Encode(publicKey)
	return encoded
}

// ConvertSignatureToBase58 - use signature with standard encoded type - Base58
func ConvertSignatureToBase58(publicKey []byte) string {
	encoded := base58.Encode(publicKey)
	return encoded
}

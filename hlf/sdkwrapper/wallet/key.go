package wallet

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/anoideaopen/robot/hlf/sdkwrapper/service"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
)

// Key is a core.Key wrapper for *ecdsa.PublicKey
type Key struct {
	PrivKey ed25519.PrivateKey
	PubKey  ed25519.PublicKey
}

func NewKey(private string) (*Key, error) {
	if private == "" {
		pKey, sKey, err := service.GeneratePrivateAndPublicKey()
		if err != nil {
			return nil, err
		}

		return &Key{sKey, pKey}, nil
	}

	privateKey, publicKey, err := service.EncodedPrivKeyToEd25519(private)
	if err != nil {
		return nil, err
	}
	return &Key{privateKey, publicKey}, nil
}

// Bytes converts this key to its byte representation.
func (k *Key) Bytes() (raw []byte, err error) {
	raw, err = x509.MarshalPKIXPublicKey(k.PubKey)
	if err != nil {
		return nil, fmt.Errorf("Failed marshalling key [%s]", err)
	}

	return
}

// SKI returns the subject key identifier of this key.
func (k *Key) SKI() (ski []byte) {
	if k.PubKey == nil {
		return nil
	}
	hash := sha256.New()
	hash.Write(k.PubKey)
	return hash.Sum(nil)
}

// Symmetric returns true if this key is a symmetric key, false otherwise.
func (k *Key) Symmetric() bool {
	return false
}

// Private returns true if this key is a private key, false otherwise.
func (k *Key) Private() bool {
	return k.PrivKey != nil
}

// PublicKey returns the corresponding public key part of an asymmetric public/private key pair.
func (k *Key) PublicKey() (core.Key, error) {
	return k, nil
}

func PEMToPrivateKey(raw []byte, pwd []byte) (interface{}, error) {
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("failed decoding PEM. Block must be different from nil [% x]", raw)
	}

	if x509.IsEncryptedPEMBlock(block) {
		if len(pwd) == 0 {
			return nil, errors.New("encrypted Key. Need a password")
		}

		decrypted, err := x509.DecryptPEMBlock(block, pwd)
		if err != nil {
			return nil, fmt.Errorf("failed PEM decryption: [%s]", err)
		}

		key, err := derToPrivateKey(decrypted)
		if err != nil {
			return nil, err
		}

		return key, err
	}

	cert, err := derToPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return cert, err
}

func derToPrivateKey(der []byte) (key interface{}, err error) {
	if key, err = x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}

	if key, err = x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key.(type) {
		case *ecdsa.PrivateKey:
			return
		default:
			return nil, errors.New("found unknown private key type in PKCS#8 wrapping")
		}
	}

	if key, err = x509.ParseECPrivateKey(der); err == nil {
		return
	}

	return nil, errors.New("invalid key type. The DER must contain an ecdsa.PrivateKey")
}

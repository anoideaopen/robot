package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/anoideaopen/cartridge/manager"
	"github.com/anoideaopen/robot/config"
	"github.com/anoideaopen/robot/hlf/hlfprofile"
	"github.com/newity/glog"
	"github.com/pkg/errors"
)

func createCryptoManager(ctx context.Context, cfg *config.Config, hlfProfile *hlfprofile.HlfProfile) (manager.Manager, error) {
	log := glog.FromContext(ctx)

	switch cfg.CryptoSrc {
	case config.VaultCryptoSrc:
		vaultToken := cfg.VaultCryptoSettings.VaultToken

		if cfg.VaultCryptoSettings.UseRenewableVaultTokens {
			token, err := getRenewableVaultToken(
				ctx,
				cfg.VaultCryptoSettings.VaultAddress,
				cfg.VaultCryptoSettings.VaultAuthPath,
				cfg.VaultCryptoSettings.VaultRole,
				cfg.VaultCryptoSettings.VaultServiceTokenPath)
			if err != nil {
				return nil, err
			}

			vaultToken = token
		}

		m, err := manager.NewVaultManager(
			hlfProfile.MspID,
			cfg.VaultCryptoSettings.UserCert,
			cfg.VaultCryptoSettings.VaultAddress,
			vaultToken,
			cfg.VaultCryptoSettings.VaultNamespace,
		)
		if err != nil {
			return nil, err
		}

		log.Infof("Vault manager created")

		return m, nil
	case config.GoogleCryptoSrc:
		m, err := manager.NewSecretManager(
			hlfProfile.MspID,
			cfg.GoogleCryptoSettings.GcloudProject,
			cfg.GoogleCryptoSettings.UserCert,
			cfg.GoogleCryptoSettings.GcloudCreds,
		)
		if err != nil {
			return nil, err
		}

		log.Infof("Google Secret manager created")

		return m, nil
	case config.LocalCryptoSrc:
		return nil, nil
	default:
		return nil, errors.Errorf("unknown crypto src in config %s (must be %s | %s | %s)", cfg.CryptoSrc,
			config.LocalCryptoSrc,
			config.VaultCryptoSrc,
			config.GoogleCryptoSrc,
		)
	}
}

func getRenewableVaultToken(ctx context.Context, vaultAddress, vaultAuthPath, vaultRole, vaultServiceTokenPath string) (string, error) {
	log := glog.FromContext(ctx)

	log.Info("Use renewable tokens Vault auth scheme")
	log.Infof("Read JWT from k8s service account")

	serviceAccountJWT, err := ioutil.ReadFile(vaultServiceTokenPath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read token from %s", vaultServiceTokenPath)
	}

	vaultURL := strings.TrimSuffix(vaultAddress, "/")
	vaultPath := strings.TrimPrefix(vaultAuthPath, "/")
	tokenRequestPath := vaultURL + "/" + vaultPath

	log.Infof("Send request for Vault token to %s", tokenRequestPath)

	tokenRequest, err := json.Marshal(
		map[string]string{
			"jwt":  string(serviceAccountJWT),
			"role": vaultRole,
		})
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal tokenRequest")
	}

	resp, err := http.Post(tokenRequestPath, "application/json", bytes.NewBuffer(tokenRequest)) //nolint:noctx
	if err != nil {
		return "", errors.Wrap(err, "failed to get Vault token")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	log.Debugf("Vault response status: %s code: %d", resp.Status, resp.StatusCode)

	dec := json.NewDecoder(resp.Body)

	var renewableToken struct {
		Auth struct {
			ClientToken string `json:"client_token"`
		} `json:"auth"`
	}
	for dec.More() {
		if err = dec.Decode(&renewableToken); err != nil {
			return "", errors.Wrap(err, "failed to decode renewableToken")
		}
	}

	return renewableToken.Auth.ClientToken, nil
}

package hlfprofile

import (
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type HlfProfile struct {
	OrgName             string
	MspID               string
	CredentialStorePath string
	CryptoStorePath     string
}

type ConnProfile struct {
	Client struct {
		Org             string `yaml:"organization"`
		CredentialStore struct {
			Path        string `yaml:"path"`
			CryptoStore struct {
				Path string `yaml:"path"`
			} `yaml:"cryptoStore"`
		} `yaml:"credentialStore"`
	} `yaml:"client"`
	Organizations map[string]struct {
		Mspid string `yaml:"mspid"`
	} `yaml:"organizations"`
}

func ParseProfile(profilePath string) (*HlfProfile, error) {
	b, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, errors.Wrap(errors.WithStack(err), "error read connection profile path")
	}

	cp := ConnProfile{}
	if err = yaml.Unmarshal(b, &cp); err != nil {
		return nil, errors.Wrap(errors.WithStack(err), "error unmarshal connection profile")
	}

	org, ok := cp.Organizations[cp.Client.Org]
	if !ok {
		return nil, errors.Wrap(errors.Errorf("cannot find mspid for org: %s", cp.Client.Org),
			"error unmarshal connection profile")
	}

	res := &HlfProfile{
		OrgName:             cp.Client.Org,
		MspID:               org.Mspid,
		CredentialStorePath: cp.Client.CredentialStore.Path,
		CryptoStorePath:     cp.Client.CredentialStore.CryptoStore.Path,
	}

	return res, err
}

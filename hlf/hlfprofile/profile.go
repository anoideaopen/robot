package hlfprofile

import (
	"fmt"
	"os"

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
		return nil, fmt.Errorf("error read connection profile path: %w", err)
	}

	cp := ConnProfile{}
	if err = yaml.Unmarshal(b, &cp); err != nil {
		return nil, fmt.Errorf("error unmarshal connection profile: %w", err)
	}

	org, ok := cp.Organizations[cp.Client.Org]
	if !ok {
		return nil, fmt.Errorf("error unmarshal connection profile: cannot find mspid for org: %s", cp.Client.Org)
	}

	res := &HlfProfile{
		OrgName:             cp.Client.Org,
		MspID:               org.Mspid,
		CredentialStorePath: cp.Client.CredentialStore.Path,
		CryptoStorePath:     cp.Client.CredentialStore.CryptoStore.Path,
	}

	return res, err
}

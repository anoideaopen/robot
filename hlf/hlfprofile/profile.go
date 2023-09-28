package hlfprofile

import (
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// HlfProfile is a struct with info about hlf profile
type HlfProfile struct {
	// OrgName is a name of organization
	OrgName string
	// MsPID is a mspid of organization
	MspID string
	// CredentialStorePath is a path to credential store
	CredentialStorePath string
	// CryptoStorePath is a path to crypto store
	CryptoStorePath string
}

// ConnProfile is a struct with info about connection profile
type ConnProfile struct {
	// Client is a struct with info about client
	Client struct {
		// Org is a name of organization
		Org string `yaml:"organization"`
		// CredentialStore is a struct with info about credential store
		CredentialStore struct {
			// Path is a path to credential store
			Path string `yaml:"path"`
			// CryptoStore is a struct with info about crypto store
			CryptoStore struct {
				// Path is a path to crypto store
				Path string `yaml:"path"`
			} `yaml:"cryptoStore"`
		} `yaml:"credentialStore"`
	} `yaml:"client"`
	// Organizations is a map with info about organizations
	Organizations map[string]struct {
		// Mspid is a mspid of organization
		Mspid string `yaml:"mspid"`
	} `yaml:"organizations"`
}

// ParseProfile parses hlf profile from path
func ParseProfile(profilePath string) (*HlfProfile, error) {
	b, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, errors.Wrap(errors.WithStack(err), "error read connection profile path")
	}

	cp := ConnProfile{}
	if err := yaml.Unmarshal(b, &cp); err != nil {
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

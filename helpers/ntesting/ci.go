package ntesting

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/anoideaopen/common-component/testshlp"
	"github.com/anoideaopen/robot/config"
	"github.com/anoideaopen/robot/hlf/hlfprofile"
	"github.com/stretchr/testify/require"
)

const (
	envTestPrefix = config.EnvPrefix + "_TEST"
)

// CI returns data for integration test
func CI(t *testing.T) CiTestData {
	/*
		if !isIntegration {
			t.Skip()
		}
	*/

	d, err := getCiData()
	require.NoError(t, err, "error get ci test data")
	return d
}

type CiTestData struct {
	RedisAddr             string
	RedisPass             string
	HlfProfilePath        string
	HlfFiatChannel        string
	HlfCcChannel          string
	HlfIndustrialChannel  string
	HlfNoCcChannel        string
	HlfUserName           string
	HlfCert               string
	HlfFiatOwnerKey       string
	HlfCcOwnerKey         string
	HlfIndustrialOwnerKey string
	HlfSk                 string
	HlfIndustrialGroup1   string
	HlfIndustrialGroup2   string
	HlfDoSwapTests        bool
	HlfDoMultiSwapTests   bool

	HlfProfile *hlfprofile.HlfProfile
}

func getCiData() (CiTestData, error) {
	ciDataOnce.Do(func() {
		ciData, errCiData = initCiData()
	})

	return ciData, errCiData
}

var (
	ciData     CiTestData
	errCiData  error
	ciDataOnce sync.Once

	isIntegration = testshlp.EnvVar{Name: envTestPrefix + "_IS_INTEGRATION", DontUseDefaultVal: true}.GetBool()

	// for develop using
	envSetEnvFromFile = testshlp.EnvVar{Name: envTestPrefix + "_SET_ENV_FROM_FILE", DontUseDefaultVal: true}

	envHlfProfilePath        = testshlp.EnvVar{Name: envTestPrefix + "_HLF_PROFILE", DontUseDefaultVal: true}
	envHlfUserName           = testshlp.EnvVar{Name: envTestPrefix + "_HLF_USER", DontUseDefaultVal: true}
	envHlfCert               = testshlp.EnvVar{Name: envTestPrefix + "_HLF_CERT", DontUseDefaultVal: true}
	envHlfSk                 = testshlp.EnvVar{Name: envTestPrefix + "_HLF_SK", DontUseDefaultVal: true}
	envHlfFiatOwnerKey       = testshlp.EnvVar{Name: envTestPrefix + "_HLF_FIAT_OWNER_KEY_BASE58CHECK", DontUseDefaultVal: true}
	envHlfCcOwnerKey         = testshlp.EnvVar{Name: envTestPrefix + "_HLF_CC_OWNER_KEY_BASE58CHECK", DontUseDefaultVal: true}
	envHlfIndustrialOwnerKey = testshlp.EnvVar{Name: envTestPrefix + "_HLF_INDUSTRIAL_OWNER_KEY_BASE58CHECK", DontUseDefaultVal: true}
	envHlfFiatChannel        = testshlp.EnvVar{Name: envTestPrefix + "_HLF_CH_FIAT", DontUseDefaultVal: true}
	envHlfCcChannel          = testshlp.EnvVar{Name: envTestPrefix + "_HLF_CH_CC", DontUseDefaultVal: true}
	envHlfIndustrialChannel  = testshlp.EnvVar{Name: envTestPrefix + "_HLF_CH_INDUSTRIAL", DontUseDefaultVal: true}
	envHlfNoCcChannel        = testshlp.EnvVar{Name: envTestPrefix + "_HLF_CH_NO_CC", DontUseDefaultVal: true}
	envHlfDoSwapTests        = testshlp.EnvVar{Name: envTestPrefix + "_HLF_DO_SWAPS", DefaultVal: "false"}
	envHlfDoMultiSwapTests   = testshlp.EnvVar{Name: envTestPrefix + "_HLF_DO_MSWAPS", DefaultVal: "false"}
	envHlfIndustrialGroup1   = testshlp.EnvVar{Name: envTestPrefix + "_HLF_INDUSTRIAL_GROUP1", DontUseDefaultVal: true}
	envHlfIndustrialGroup2   = testshlp.EnvVar{Name: envTestPrefix + "_HLF_INDUSTRIAL_GROUP2", DontUseDefaultVal: true}
	envRedisAddr             = testshlp.EnvVar{Name: envTestPrefix + "_REDIS_ADDR", DefaultVal: "127.0.0.1:6379"}
	envRedisPass             = testshlp.EnvVar{Name: envTestPrefix + "_REDIS_PASS", DefaultVal: "test"}
)

func initCiData() (CiTestData, error) {
	if envPath := envSetEnvFromFile.GetValOrDefault(); envPath != "" {
		if err := testshlp.SetEnvFromFile(envPath); err != nil {
			return CiTestData{}, err
		}
	}

	d := CiTestData{
		RedisAddr:             envRedisAddr.GetValOrDefault(),
		RedisPass:             envRedisPass.GetValOrDefault(),
		HlfProfilePath:        envHlfProfilePath.GetValOrDefault(),
		HlfFiatChannel:        envHlfFiatChannel.GetValOrDefault(),
		HlfCcChannel:          envHlfCcChannel.GetValOrDefault(),
		HlfIndustrialChannel:  envHlfIndustrialChannel.GetValOrDefault(),
		HlfNoCcChannel:        envHlfNoCcChannel.GetValOrDefault(),
		HlfUserName:           envHlfUserName.GetValOrDefault(),
		HlfCert:               envHlfCert.GetValOrDefault(),
		HlfFiatOwnerKey:       envHlfFiatOwnerKey.GetValOrDefault(),
		HlfIndustrialOwnerKey: envHlfIndustrialOwnerKey.GetValOrDefault(),
		HlfCcOwnerKey:         envHlfCcOwnerKey.GetValOrDefault(),
		HlfSk:                 envHlfSk.GetValOrDefault(),
		HlfIndustrialGroup1:   envHlfIndustrialGroup1.GetValOrDefault(),
		HlfIndustrialGroup2:   envHlfIndustrialGroup2.GetValOrDefault(),
		HlfDoSwapTests:        envHlfDoSwapTests.GetBool(),
		HlfDoMultiSwapTests:   envHlfDoMultiSwapTests.GetBool(),
	}

	hlfProfile, err := hlfprofile.ParseProfile(d.HlfProfilePath)
	if err != nil {
		return d, err
	}
	d.HlfProfile = hlfProfile

	if d.HlfCert == "" {
		d.HlfCert = filepath.Join(d.HlfProfile.CredentialStorePath, fmt.Sprintf("%s@%s-cert.pem", d.HlfUserName, d.HlfProfile.OrgName))
	}

	if d.HlfSk == "" {
		d.HlfSk = filepath.Join(d.HlfProfile.CryptoStorePath, "keystore/priv_sk")
	}

	return d, nil
}

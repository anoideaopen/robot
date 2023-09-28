package ntesting

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/atomyze-foundation/common-component/testshlp"
	"github.com/atomyze-foundation/robot/config"
	"github.com/atomyze-foundation/robot/hlf/hlfprofile"
	"github.com/stretchr/testify/require"
)

const (
	envTestPrefix = config.EnvPrefix + "_TEST"
)

// CI returns data for integration test
func CI(t *testing.T) CiTestData {
	if !isIntegration {
		t.Skip()
	}

	d, err := getCiData()
	require.NoError(t, err, "error get ci test data")
	return d
}

// CiTestData is a struct with data for integration test
type CiTestData struct {
	// RedisAddr is a redis address
	RedisAddr string
	// RedisPass is a redis password
	RedisPass string
	// HlfProfilePath is a path to hlf profile
	HlfProfilePath string
	// HlfFiatChannel is a fiat channel name
	HlfFiatChannel string
	// HlfCcChannel is a cc channel name
	HlfCcChannel string
	// HlfIndustrialChannel is a industrial channel name
	HlfIndustrialChannel string
	// HlfNoCcChannel is a no cc channel name
	HlfNoCcChannel string
	// HlfUserName is a hlf user name
	HlfUserName string
	// HlfCert is a hlf cert
	HlfCert string
	// HlfFiatOwnerKey is a fiat owner key
	HlfFiatOwnerKey string
	// HlfCcOwnerKey is a cc owner key
	HlfCcOwnerKey string
	// HlfIndustrialOwnerKey is a industrial owner key
	HlfIndustrialOwnerKey string
	// HlfUseSmartBFT is a flag for using smart bft
	HlfUseSmartBFT bool
	// HlfSk is a hlf sk
	HlfSk string
	// HlfIndustrialGroup1 is a industrial group 1
	HlfIndustrialGroup1 string
	// HlfIndustrialGroup2 is a industrial group 2
	HlfIndustrialGroup2 string
	// HlfDoSwapTests is a flag for doing swap tests
	HlfDoSwapTests bool
	// HlfDoMultiSwapTests is a flag for doing multi swap tests
	HlfDoMultiSwapTests bool
	// HlfProfile is a hlf profile
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

	isIntegration = testshlp.EnvVar{Name: fmt.Sprintf("%s_IS_INTEGRATION", envTestPrefix), DontUseDefaultVal: true}.GetBool()

	// for develop using
	envSetEnvFromFile = testshlp.EnvVar{Name: fmt.Sprintf("%s_SET_ENV_FROM_FILE", envTestPrefix), DontUseDefaultVal: true}

	envHlfProfilePath        = testshlp.EnvVar{Name: fmt.Sprintf("%s_HLF_PROFILE", envTestPrefix), DontUseDefaultVal: true}
	envHlfUserName           = testshlp.EnvVar{Name: fmt.Sprintf("%s_HLF_USER", envTestPrefix), DontUseDefaultVal: true}
	envHlfCert               = testshlp.EnvVar{Name: fmt.Sprintf("%s_HLF_CERT", envTestPrefix), DontUseDefaultVal: true}
	envHlfSk                 = testshlp.EnvVar{Name: fmt.Sprintf("%s_HLF_SK", envTestPrefix), DontUseDefaultVal: true}
	envHlfFiatOwnerKey       = testshlp.EnvVar{Name: fmt.Sprintf("%s_HLF_FIAT_OWNER_KEY_BASE58CHECK", envTestPrefix), DontUseDefaultVal: true}
	envHlfCcOwnerKey         = testshlp.EnvVar{Name: fmt.Sprintf("%s_HLF_CC_OWNER_KEY_BASE58CHECK", envTestPrefix), DontUseDefaultVal: true}
	envHlfIndustrialOwnerKey = testshlp.EnvVar{Name: fmt.Sprintf("%s_HLF_INDUSTRIAL_OWNER_KEY_BASE58CHECK", envTestPrefix), DontUseDefaultVal: true}
	envHlfFiatChannel        = testshlp.EnvVar{Name: fmt.Sprintf("%s_HLF_CH_FIAT", envTestPrefix), DontUseDefaultVal: true}
	envHlfCcChannel          = testshlp.EnvVar{Name: fmt.Sprintf("%s_HLF_CH_CC", envTestPrefix), DontUseDefaultVal: true}
	envHlfIndustrialChannel  = testshlp.EnvVar{Name: fmt.Sprintf("%s_HLF_CH_INDUSTRIAL", envTestPrefix), DontUseDefaultVal: true}
	envHlfNoCcChannel        = testshlp.EnvVar{Name: fmt.Sprintf("%s_HLF_CH_NO_CC", envTestPrefix), DontUseDefaultVal: true}
	envHlfUseSmartBFT        = testshlp.EnvVar{Name: fmt.Sprintf("%s_HLF_USE_SMART_BFT", envTestPrefix), DefaultVal: "true"}
	envHlfDoSwapTests        = testshlp.EnvVar{Name: fmt.Sprintf("%s_HLF_DO_SWAPS", envTestPrefix), DefaultVal: "false"}
	envHlfDoMultiSwapTests   = testshlp.EnvVar{Name: fmt.Sprintf("%s_HLF_DO_MSWAPS", envTestPrefix), DefaultVal: "false"}
	envHlfIndustrialGroup1   = testshlp.EnvVar{Name: fmt.Sprintf("%s_HLF_INDUSTRIAL_GROUP1", envTestPrefix), DontUseDefaultVal: true}
	envHlfIndustrialGroup2   = testshlp.EnvVar{Name: fmt.Sprintf("%s_HLF_INDUSTRIAL_GROUP2", envTestPrefix), DontUseDefaultVal: true}
	envRedisAddr             = testshlp.EnvVar{Name: fmt.Sprintf("%s_REDIS_ADDR", envTestPrefix), DefaultVal: "127.0.0.1:6379"}
	envRedisPass             = testshlp.EnvVar{Name: fmt.Sprintf("%s_REDIS_PASS", envTestPrefix), DefaultVal: "test"}
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
		HlfUseSmartBFT:        envHlfUseSmartBFT.GetBool(),
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

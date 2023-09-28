package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	// EnvPrefix is a prefix for env variables
	EnvPrefix         = "ROBOT"
	sensitiveDataMask = "****"
)

// Config is a struct with config
type Config struct {
	// LogLevel is a log level (debug, info, warn, error, fatal, panic)
	LogLevel string `mapstructure:"logLevel" validate:"required"`
	// LogType is a log type (json, text)
	LogType string `mapstructure:"logType" validate:"required"`
	// ServerPort is a port for http server
	ServerPort uint `mapstructure:"serverPort"`
	// ProfilePath is a path for profiling
	ProfilePath string `mapstructure:"profilePath" validate:"required"`
	// UserName is a user name
	UserName string `mapstructure:"userName" validate:"required"`
	// UseSmartBFT is a flag for using smart bft
	UseSmartBFT bool `mapstructure:"useSmartBFT"`
	// TxSwapPrefix is a prefix for swap tx
	TxSwapPrefix string `mapstructure:"txSwapPrefix" validate:"required"`
	// TxMultiSwapPrefix is a prefix for multi swap tx
	TxMultiSwapPrefix string `mapstructure:"txMultiSwapPrefix" validate:"required"`
	// TxPreimagePrefix is a prefix for preimage tx
	TxPreimagePrefix string `mapstructure:"txPreimagePrefix" validate:"required"`
	// DelayAfterChRobotError is a delay after channel robot error
	DelayAfterChRobotError time.Duration `mapstructure:"delayAfterChRobotError"`
	// DefaultBatchLimits is a default batch limits
	DefaultBatchLimits *BatchLimits `mapstructure:"defaultBatchLimits"`
	// RedisStorage is a redis storage config
	RedisStorage *RedisStorage `mapstructure:"redisStor" validate:"required"`
	// PromMetrics is a prometheus metrics config
	PromMetrics *PromMetrics `mapstructure:"promMetrics"`

	// CryptoSrc is a crypto manager kind can be local, vault, google
	CryptoSrc CryptoSrc `mapstructure:"cryptoSrc" validate:"required"`
	// VaultCryptoSettings is a vault crypto settings
	VaultCryptoSettings *VaultCryptoSettings `mapstructure:"vaultCryptoSettings"`
	// GoogleCryptoSettings is a google crypto settings
	GoogleCryptoSettings *GoogleCryptoSettings `mapstructure:"googleCryptoSettings"`
	// DefaultRobotExecOpts is a default robot execute options
	DefaultRobotExecOpts ExecuteOptions `mapstructure:"defaultRobotExecOpts" validate:"dive"`
	// Robots is a robots config
	Robots []*Robot `mapstructure:"robots" validate:"dive"`
}

// PromMetrics is a prometheus metrics config
type PromMetrics struct {
	PrefixForMetrics string `mapstructure:"prefix"`
}

// CryptoSrc is a crypto manager kind can be local, vault, google
type CryptoSrc string

const (
	// LocalCryptoSrc is a local crypto manager
	LocalCryptoSrc CryptoSrc = "local"
	// VaultCryptoSrc is a vault crypto manager
	VaultCryptoSrc CryptoSrc = "vault"
	// GoogleCryptoSrc is a google crypto manager
	GoogleCryptoSrc CryptoSrc = "google"
)

// VaultCryptoSettings is a vault crypto settings
type VaultCryptoSettings struct {
	// VaultToken is a vault token
	VaultToken string `mapstructure:"vaultToken"`
	// UseRenewableVaultTokens is a flag for using renewable vault tokens
	UseRenewableVaultTokens bool `mapstructure:"useRenewableVaultTokens"`
	// VaultAddress is a vault address
	VaultAddress string `mapstructure:"vaultAddress"`
	// VaultAuthPath is a vault auth path
	VaultAuthPath string `mapstructure:"vaultAuthPath"`
	// VaultRole is a vault role
	VaultRole string `mapstructure:"vaultRole"`
	// VaultServiceTokenPath is a vault service token path
	VaultServiceTokenPath string `mapstructure:"vaultServiceTokenPath"`
	// VaultNamespace is a vault namespace
	VaultNamespace string `mapstructure:"vaultNamespace"`
	// UserCert is a user cert
	UserCert string `mapstructure:"userCert"`
}

// GoogleCryptoSettings is a google crypto settings
type GoogleCryptoSettings struct {
	// GcloudProject is a gcloud project
	GcloudProject string `mapstructure:"gcloudProject"`
	// GcloudCreds is a gcloud creds
	GcloudCreds string `mapstructure:"gcloudCreds"`
	// UserCert is a user cert
	UserCert string `mapstructure:"userCert"`
}

// Robot is a robot config
type Robot struct {
	// ChName is a channel name for robot
	ChName string `mapstructure:"chName" validate:"required"`
	// InitMinExecBlockNum is a init min exec block num
	InitMinExecBlockNum uint64 `mapstructure:"initExecBlockNum"`
	// SrcChannels is a list of source channels
	SrcChannels []*SrcChannel `mapstructure:"src" validate:"dive"`
	// BatchLimits is a batch limits
	BatchLimits *BatchLimits `mapstructure:"batchLimits"`
	// CollectorsBufSize is a collectors buffer size
	CollectorsBufSize uint `mapstructure:"collectorsBufSize"`
	// ExecOpts is a execute options for robot
	ExecOpts ExecuteOptions `mapstructure:"execOpts" validate:"dive"`
}

// BatchLimits is a batch limits config for robot
type BatchLimits struct {
	// BatchBlocksCountLimit is a batch blocks count limit
	BatchBlocksCountLimit uint `mapstructure:"batchBlocksCountLimit"`
	// BatchLenLimit is a batch len limit
	BatchLenLimit uint `mapstructure:"batchLenLimit"`
	// BatchSizeLimit is a batch size limit
	BatchSizeLimit uint `mapstructure:"batchSizeLimit"`
	// BatchTimeoutLimit is a batch timeout limit
	BatchTimeoutLimit time.Duration `mapstructure:"batchTimeoutLimit"`
}

func (bl *BatchLimits) isEmpty() bool {
	return bl == nil || *bl == BatchLimits{}
}

// ExecuteOptions is a execute options for robot
type ExecuteOptions struct {
	// ExecuteTimeout is a execute timeout
	ExecuteTimeout *time.Duration `mapstructure:"executeTimeout"`
	// WaitCommitAttempts is a wait commit attempts
	WaitCommitAttempts *uint `mapstructure:"waitCommitAttempts"`
	// WaitCommitAttemptTimeout is a wait commit attempt timeout
	WaitCommitAttemptTimeout *time.Duration `mapstructure:"waitCommitAttemptTimeout"`
}

// EffExecuteTimeout returns effective execute timeout
func (eo ExecuteOptions) EffExecuteTimeout(defOpts ExecuteOptions) (time.Duration, error) {
	var val *time.Duration
	if eo.ExecuteTimeout != nil {
		val = eo.ExecuteTimeout
	} else if defOpts.ExecuteTimeout != nil {
		val = defOpts.ExecuteTimeout
	}
	if val == nil {
		return 0, errors.New("executeTimeout is not specified")
	}
	return *val, nil
}

// EffWaitCommitAttempts returns effective wait commit attempts
func (eo ExecuteOptions) EffWaitCommitAttempts(defOpts ExecuteOptions) (uint, error) {
	var val *uint
	if eo.WaitCommitAttempts != nil {
		val = eo.WaitCommitAttempts
	} else if defOpts.WaitCommitAttempts != nil {
		val = defOpts.WaitCommitAttempts
	}
	if val == nil {
		return 0, errors.New("waitCommitAttempts is not specified")
	}
	if *val == 0 {
		return 0, errors.New("waitCommitAttempts must be above 0")
	}
	return *val, nil
}

// EffWaitCommitAttemptTimeout returns effective wait commit attempt timeout
func (eo ExecuteOptions) EffWaitCommitAttemptTimeout(defOpts ExecuteOptions) (time.Duration, error) {
	var val *time.Duration
	if eo.WaitCommitAttemptTimeout != nil {
		val = eo.WaitCommitAttemptTimeout
	} else if defOpts.WaitCommitAttemptTimeout != nil {
		val = defOpts.WaitCommitAttemptTimeout
	}
	if val == nil {
		return 0, errors.New("waitCommitAttemptTimeout is not specified")
	}
	return *val, nil
}

// RedisStorage is a redis storage config
type RedisStorage struct {
	// DBPrefix is a db prefix
	DBPrefix string `mapstructure:"dbPrefix"`
	// Addr is a list of redis addresses
	Addr []string `mapstructure:"addr"`
	// Password is a password for redis
	Password string `mapstructure:"password"`
	// WithTLS is a flag for using tls
	WithTLS bool `mapstructure:"withTLS"`
	// RootCAs is a list of root ca files
	RootCAs []string `mapstructure:"rootCAs"`
}

// SrcChannel is a source channel config
type SrcChannel struct {
	// ChName is a channel name
	ChName string `mapstructure:"chName" validate:"required"`
	// InitBlockNum is a init block num
	InitBlockNum *uint64 `mapstructure:"initBlockNum" validate:"required"`
}

// GetConfig returns config from file
func GetConfig() (*Config, error) {
	p, ok, err := findConfigPath()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("config path is required - specify the -c parameter or ROBOT_CONFIG env variable")
	}
	cfg, err := getConfig(p)
	if err != nil {
		return nil, err
	}

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func validateConfig(cfg *Config) error {
	err := validateRequiredFields(cfg)
	if err != nil {
		return err
	}

	if cfg.DefaultBatchLimits.isEmpty() {
		for _, robot := range cfg.Robots {
			if robot.BatchLimits.isEmpty() {
				return errors.Errorf("invalid batch limits for chName: %v", robot.ChName)
			}
		}
	}

	if len(cfg.Robots) == 0 {
		return errors.Errorf("no robots")
	}

	for _, robot := range cfg.Robots {
		if len(robot.SrcChannels) == 0 {
			return errors.Errorf("no srcChannels for channel: %v", robot.ChName)
		}

		if robot.CollectorsBufSize == 0 {
			return errors.Errorf("collectorsBufSize must be positive for channel: %v", robot.ChName)
		}

		const errTempl = "%w for channel: %v"
		if _, err := robot.ExecOpts.EffExecuteTimeout(cfg.DefaultRobotExecOpts); err != nil {
			return errors.WithStack(fmt.Errorf(errTempl, err, robot.ChName))
		}
		if _, err := robot.ExecOpts.EffWaitCommitAttempts(cfg.DefaultRobotExecOpts); err != nil {
			return errors.WithStack(fmt.Errorf(errTempl, err, robot.ChName))
		}
		if _, err := robot.ExecOpts.EffWaitCommitAttemptTimeout(cfg.DefaultRobotExecOpts); err != nil {
			return errors.WithStack(fmt.Errorf(errTempl, err, robot.ChName))
		}
	}

	if err := validateSwaps(cfg.Robots); err != nil {
		return err
	}

	if cfg.CryptoSrc != LocalCryptoSrc &&
		cfg.CryptoSrc != GoogleCryptoSrc &&
		cfg.CryptoSrc != VaultCryptoSrc {
		return errors.Errorf("unknown crypto manager kind: %v", cfg.CryptoSrc)
	}

	if cfg.CryptoSrc == GoogleCryptoSrc && cfg.GoogleCryptoSettings == nil {
		return errors.New("googleCryptoSettings are empty")
	}

	if cfg.CryptoSrc == VaultCryptoSrc && cfg.VaultCryptoSettings == nil {
		return errors.New("vaultCryptoSettings are empty")
	}
	return nil
}

func validateSwaps(robots []*Robot) error {
	rm := make(map[string]map[string]struct{})
	// check for duplicates
	for _, dst := range robots {
		if _, ok := rm[dst.ChName]; ok {
			return errors.Errorf("robot duplicate for channel: %v", dst.ChName)
		}
		rm[dst.ChName] = map[string]struct{}{}
		for _, src := range dst.SrcChannels {
			if _, ok := rm[dst.ChName][src.ChName]; ok {
				return errors.Errorf("robot source duplicate for channel: %v source: %v", dst.ChName, src.ChName)
			}
			rm[dst.ChName][src.ChName] = struct{}{}
		}
	}

	// check for swaps
	for dst, srces := range rm {
		for src := range srces {
			if dst == src {
				continue
			}
			if _, ok := rm[src]; !ok {
				continue
			}
			if _, ok := rm[src][dst]; !ok {
				return errors.Errorf(
					"robot for channel: %v source: %v configured, but robot for channel: %v source: %v not found",
					dst, src, src, dst,
				)
			}
		}
	}

	return nil
}

func validateRequiredFields(cfg *Config) error {
	validate := validator.New()
	err := validate.Struct(*cfg)
	if err != nil {
		return errors.Errorf("err(s):\n%+v", err)
	}
	return nil
}

func findConfigPath() (string, bool, error) {
	// 1. params
	if p, ok := getConfigPathFromParams(); ok {
		return p, true, nil
	}
	// 2. env
	if p, ok := os.LookupEnv("ROBOT_CONFIG"); ok {
		return p, true, nil
	}
	// 3. config.yaml
	ex, err := os.Executable()
	if err != nil {
		return "", false, errors.Wrap(err, "error on finding config file")
	}
	p := filepath.Join(filepath.Dir(ex), "config.yaml")
	ok, err := existFile(p)
	if err != nil {
		return "", false, errors.Wrapf(err, "error on finding %s file", p)
	}
	if ok {
		return p, true, nil
	}
	// 4. /etc/config.yaml
	p = "/etc/config.yaml"
	ok, err = existFile(p)
	if err != nil {
		return "", false, errors.Wrapf(err, "error on finding %s file", p)
	}
	if ok {
		return p, true, nil
	}
	return "", false, nil
}

func getConfig(cfgPath string) (*Config, error) {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetTypeByDefaultValue(true)
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetConfigFile(cfgPath)

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		return nil, errors.Wrap(err, "failed viper.ReadInConfig")
	}

	cfg := Config{}
	if err := viper.UnmarshalExact(&cfg); err != nil {
		return nil, errors.Wrap(err, "failed viper.Unmarshal")
	}

	return &cfg, nil
}

func getConfigPathFromParams() (string, bool) {
	p := flag.String(
		"c",
		"",
		"Configuration file path",
	)
	flag.Parse()
	return *p, *p != ""
}

func existFile(p string) (bool, error) {
	_, err := os.Stat(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, errors.WithStack(err)
	}
	return true, nil
}

// WithoutSensitiveData returns copy of config with empty sensitive data. This config might be used for trace logging.
func (c Config) WithoutSensitiveData() Config {
	robots := make([]*Robot, 0, len(c.Robots))
	for _, r := range c.Robots {
		robots = append(robots, r.withoutSensitiveData())
	}

	return Config{
		LogLevel:               c.LogLevel,
		LogType:                c.LogType,
		ServerPort:             c.ServerPort,
		ProfilePath:            c.ProfilePath,
		UserName:               c.UserName,
		UseSmartBFT:            c.UseSmartBFT,
		TxSwapPrefix:           c.TxSwapPrefix,
		TxMultiSwapPrefix:      c.TxMultiSwapPrefix,
		TxPreimagePrefix:       c.TxPreimagePrefix,
		DelayAfterChRobotError: c.DelayAfterChRobotError,
		DefaultBatchLimits:     c.DefaultBatchLimits.withoutSensitiveData(),
		DefaultRobotExecOpts:   c.DefaultRobotExecOpts.withoutSensitiveData(),
		RedisStorage:           c.RedisStorage.withoutSensitiveData(),
		CryptoSrc:              c.CryptoSrc,
		VaultCryptoSettings:    c.VaultCryptoSettings.withoutSensitiveData(),
		GoogleCryptoSettings:   c.GoogleCryptoSettings.withoutSensitiveData(),
		Robots:                 robots,
		PromMetrics:            c.PromMetrics,
	}
}

func (r *Robot) withoutSensitiveData() *Robot {
	if r == nil {
		return nil
	}

	srcChannels := make([]*SrcChannel, 0, len(r.SrcChannels))
	for _, s := range r.SrcChannels {
		srcChannels = append(srcChannels, s.withoutSensitiveData())
	}

	return &Robot{
		ChName:              r.ChName,
		InitMinExecBlockNum: r.InitMinExecBlockNum,
		SrcChannels:         srcChannels,
		BatchLimits:         r.BatchLimits.withoutSensitiveData(),
		CollectorsBufSize:   r.CollectorsBufSize,
		ExecOpts:            r.ExecOpts.withoutSensitiveData(),
	}
}

func (bl *BatchLimits) withoutSensitiveData() *BatchLimits {
	if bl == nil {
		return nil
	}
	return &BatchLimits{
		BatchBlocksCountLimit: bl.BatchBlocksCountLimit,
		BatchLenLimit:         bl.BatchLenLimit,
		BatchSizeLimit:        bl.BatchSizeLimit,
		BatchTimeoutLimit:     bl.BatchTimeoutLimit,
	}
}

func (eo ExecuteOptions) withoutSensitiveData() ExecuteOptions {
	return ExecuteOptions{
		ExecuteTimeout:           eo.ExecuteTimeout,
		WaitCommitAttempts:       eo.WaitCommitAttempts,
		WaitCommitAttemptTimeout: eo.WaitCommitAttemptTimeout,
	}
}

func (s *SrcChannel) withoutSensitiveData() *SrcChannel {
	if s == nil {
		return nil
	}
	return &SrcChannel{
		ChName:       s.ChName,
		InitBlockNum: s.InitBlockNum,
	}
}

func (s *RedisStorage) withoutSensitiveData() *RedisStorage {
	if s == nil {
		return nil
	}
	return &RedisStorage{
		DBPrefix: sensitiveDataMask,
		Addr:     []string{sensitiveDataMask},
		Password: sensitiveDataMask,
		WithTLS:  s.WithTLS,
		RootCAs:  []string{sensitiveDataMask},
	}
}

func (s *VaultCryptoSettings) withoutSensitiveData() *VaultCryptoSettings {
	if s == nil {
		return nil
	}
	return &VaultCryptoSettings{
		VaultToken:              sensitiveDataMask,
		UseRenewableVaultTokens: s.UseRenewableVaultTokens,
		VaultAddress:            s.VaultAddress,
		VaultAuthPath:           s.VaultAuthPath,
		VaultRole:               sensitiveDataMask,
		VaultServiceTokenPath:   sensitiveDataMask,
		VaultNamespace:          sensitiveDataMask,
		UserCert:                sensitiveDataMask,
	}
}

func (s *GoogleCryptoSettings) withoutSensitiveData() *GoogleCryptoSettings {
	if s == nil {
		return nil
	}
	return &GoogleCryptoSettings{
		GcloudProject: sensitiveDataMask,
		GcloudCreds:   sensitiveDataMask,
		UserCert:      sensitiveDataMask,
	}
}

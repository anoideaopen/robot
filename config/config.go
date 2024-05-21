package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

const (
	EnvPrefix         = "ROBOT"
	sensitiveDataMask = "****"
)

type Config struct {
	LogLevel string `mapstructure:"logLevel" validate:"required"`
	LogType  string `mapstructure:"logType" validate:"required"`

	ServerPort uint `mapstructure:"serverPort"`

	ProfilePath string `mapstructure:"profilePath" validate:"required"`
	UserName    string `mapstructure:"userName" validate:"required"`

	TxSwapPrefix           string        `mapstructure:"txSwapPrefix" validate:"required"`
	TxMultiSwapPrefix      string        `mapstructure:"txMultiSwapPrefix" validate:"required"`
	TxPreimagePrefix       string        `mapstructure:"txPreimagePrefix" validate:"required"`
	DelayAfterChRobotError time.Duration `mapstructure:"delayAfterChRobotError"`

	DefaultBatchLimits *BatchLimits `mapstructure:"defaultBatchLimits"`

	RedisStorage *RedisStorage `mapstructure:"redisStor" validate:"required"`

	PromMetrics *PromMetrics `mapstructure:"promMetrics"`

	// can be local, vault, google
	CryptoSrc            CryptoSrc             `mapstructure:"cryptoSrc" validate:"required"`
	VaultCryptoSettings  *VaultCryptoSettings  `mapstructure:"vaultCryptoSettings"`
	GoogleCryptoSettings *GoogleCryptoSettings `mapstructure:"googleCryptoSettings"`

	DefaultRobotExecOpts ExecuteOptions `mapstructure:"defaultRobotExecOpts" validate:"dive"`
	Robots               []*Robot       `mapstructure:"robots" validate:"dive"`
}

type PromMetrics struct {
	PrefixForMetrics string `mapstructure:"prefix"`
}

type CryptoSrc string

const (
	LocalCryptoSrc  CryptoSrc = "local"
	VaultCryptoSrc  CryptoSrc = "vault"
	GoogleCryptoSrc CryptoSrc = "google"
)

type VaultCryptoSettings struct {
	VaultToken              string `mapstructure:"vaultToken"`
	UseRenewableVaultTokens bool   `mapstructure:"useRenewableVaultTokens"`
	VaultAddress            string `mapstructure:"vaultAddress"`
	VaultAuthPath           string `mapstructure:"vaultAuthPath"`
	VaultRole               string `mapstructure:"vaultRole"`
	VaultServiceTokenPath   string `mapstructure:"vaultServiceTokenPath"`
	VaultNamespace          string `mapstructure:"vaultNamespace"`
	UserCert                string `mapstructure:"userCert"`
}

type GoogleCryptoSettings struct {
	GcloudProject string `mapstructure:"gcloudProject"`
	GcloudCreds   string `mapstructure:"gcloudCreds"`
	UserCert      string `mapstructure:"userCert"`
}

type Robot struct {
	ChName              string         `mapstructure:"chName" validate:"required"`
	InitMinExecBlockNum uint64         `mapstructure:"initExecBlockNum"`
	SrcChannels         []*SrcChannel  `mapstructure:"src" validate:"dive"`
	BatchLimits         *BatchLimits   `mapstructure:"batchLimits"`
	CollectorsBufSize   uint           `mapstructure:"collectorsBufSize"`
	ExecOpts            ExecuteOptions `mapstructure:"execOpts" validate:"dive"`
}

type BatchLimits struct {
	BatchBlocksCountLimit uint          `mapstructure:"batchBlocksCountLimit"`
	BatchLenLimit         uint          `mapstructure:"batchLenLimit"`
	BatchSizeLimit        uint          `mapstructure:"batchSizeLimit"`
	BatchTimeoutLimit     time.Duration `mapstructure:"batchTimeoutLimit"`
}

func (bl *BatchLimits) isEmpty() bool {
	return bl == nil || *bl == BatchLimits{}
}

type ExecuteOptions struct {
	ExecuteTimeout *time.Duration `mapstructure:"executeTimeout"`
}

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

type RedisStorage struct {
	DBPrefix string   `mapstructure:"dbPrefix"`
	Addr     []string `mapstructure:"addr"`
	Password string   `mapstructure:"password"`
	WithTLS  bool     `mapstructure:"withTLS"`
	RootCAs  []string `mapstructure:"rootCAs"`
}

type SrcChannel struct {
	ChName       string  `mapstructure:"chName" validate:"required"`
	InitBlockNum *uint64 `mapstructure:"initBlockNum" validate:"required"`
}

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

	if err = validateConfig(cfg); err != nil {
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
				return fmt.Errorf("invalid batch limits for chName: %v", robot.ChName)
			}
		}
	}

	if len(cfg.Robots) == 0 {
		return errors.New("no robots")
	}

	for _, robot := range cfg.Robots {
		if len(robot.SrcChannels) == 0 {
			return fmt.Errorf("no srcChannels for channel: %v", robot.ChName)
		}

		if robot.CollectorsBufSize == 0 {
			return fmt.Errorf("collectorsBufSize must be positive for channel: %v", robot.ChName)
		}

		if _, err = robot.ExecOpts.EffExecuteTimeout(cfg.DefaultRobotExecOpts); err != nil {
			return fmt.Errorf("error for channel %v: %w", robot.ChName, err)
		}
	}

	if err = validateSwaps(cfg.Robots); err != nil {
		return err
	}

	if cfg.CryptoSrc != LocalCryptoSrc &&
		cfg.CryptoSrc != GoogleCryptoSrc &&
		cfg.CryptoSrc != VaultCryptoSrc {
		return fmt.Errorf("unknown crypto manager kind: %v", cfg.CryptoSrc)
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
			return fmt.Errorf("robot duplicate for channel: %v", dst.ChName)
		}
		rm[dst.ChName] = map[string]struct{}{}
		for _, src := range dst.SrcChannels {
			if _, ok := rm[dst.ChName][src.ChName]; ok {
				return fmt.Errorf("robot source duplicate for channel: %v source: %v", dst.ChName, src.ChName)
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
				return fmt.Errorf(
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
		return fmt.Errorf("err(s): %w", err)
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
		return "", false, fmt.Errorf("error on finding config file: %w", err)
	}
	p := filepath.Join(filepath.Dir(ex), "config.yaml")
	ok, err := existFile(p)
	if err != nil {
		return "", false, fmt.Errorf("error on finding %s file: %w", p, err)
	}
	if ok {
		return p, true, nil
	}
	// 4. /etc/config.yaml
	p = "/etc/config.yaml"
	ok, err = existFile(p)
	if err != nil {
		return "", false, fmt.Errorf("error on finding %s file: %w", p, err)
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
		return nil, fmt.Errorf("failed viper.ReadInConfig: %w", err)
	}

	cfg := Config{}
	if err := viper.UnmarshalExact(&cfg); err != nil {
		return nil, fmt.Errorf("failed viper.Unmarshal: %w", err)
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
		return false, err
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
		ExecuteTimeout: eo.ExecuteTimeout,
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

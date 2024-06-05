package config

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	testConfigName        = "config_test.yaml"
	testSwapConfigName    = "config_swap_test.yaml"
	testSwapErrConfigName = "config_swap_err_test.yaml"
)

func TestGetConfigSimple(t *testing.T) {
	c, err := getConfig(testConfigName)
	require.NoError(t, err)
	require.NotNil(t, c)

	require.Equal(t, "info", c.LogLevel)

	require.Len(t, c.Robots, 4)

	require.Equal(t, c.Robots[0].ChName, "ch1")
	require.Len(t, c.Robots[0].SrcChannels, 3)
	require.Equal(t, c.Robots[0].SrcChannels[0].ChName, "ch1")
	require.Equal(t, *c.Robots[0].SrcChannels[0].InitBlockNum, uint64(111))
	require.Equal(t, c.Robots[0].SrcChannels[1].ChName, "sch1")
	require.Equal(t, *c.Robots[0].SrcChannels[1].InitBlockNum, uint64(222))
	require.Equal(t, c.Robots[0].SrcChannels[2].ChName, "sch2")
	require.Equal(t, *c.Robots[0].SrcChannels[2].InitBlockNum, uint64(444))

	require.Equal(t, c.Robots[1].ChName, "sch1")
	require.Len(t, c.Robots[1].SrcChannels, 3)
	require.Equal(t, c.Robots[1].SrcChannels[0].ChName, "ch1")
	require.Equal(t, *c.Robots[1].SrcChannels[0].InitBlockNum, uint64(111))
	require.Equal(t, c.Robots[1].SrcChannels[1].ChName, "sch1")
	require.Equal(t, *c.Robots[1].SrcChannels[1].InitBlockNum, uint64(222))
	require.Equal(t, c.Robots[1].SrcChannels[2].ChName, "sch2")
	require.Equal(t, *c.Robots[1].SrcChannels[2].InitBlockNum, uint64(444))

	require.Equal(t, c.Robots[2].ChName, "sch2")
	require.Len(t, c.Robots[2].SrcChannels, 3)
	require.Equal(t, c.Robots[2].SrcChannels[0].ChName, "ch1")
	require.Equal(t, *c.Robots[2].SrcChannels[0].InitBlockNum, uint64(111))
	require.Equal(t, c.Robots[2].SrcChannels[1].ChName, "sch1")
	require.Equal(t, *c.Robots[2].SrcChannels[1].InitBlockNum, uint64(222))
	require.Equal(t, c.Robots[2].SrcChannels[2].ChName, "sch2")
	require.Equal(t, *c.Robots[2].SrcChannels[2].InitBlockNum, uint64(444))

	require.Equal(t, c.Robots[3].ChName, "ch2")
	require.Len(t, c.Robots[3].SrcChannels, 1)
	require.Equal(t, *c.Robots[3].SrcChannels[0].InitBlockNum, uint64(222))

	require.Equal(t, c.TxSwapPrefix, "swaps")
	require.Equal(t, c.TxMultiSwapPrefix, "multi_swap")
	require.Equal(t, c.TxPreimagePrefix, "batchTransactions")
}

func TestGetConfigOverrideEnv(t *testing.T) {
	err := os.Setenv(fmt.Sprintf("%s_LOGLEVEL", EnvPrefix), "myval")
	require.NoError(t, err)

	// now does not work
	// err = os.Setenv(fmt.Sprintf("%s_ROBOTS1_CHNAME", envPrefix), "myname")
	// require.NoError(t, err)

	c, err := getConfig(testConfigName)
	require.NoError(t, err)
	require.NotNil(t, c)
	require.Equal(t, "myval", c.LogLevel)
}

func TestValidateConfig(t *testing.T) {
	c, err := getConfig(testConfigName)
	require.NoError(t, err)
	err = validateConfig(c)
	require.NoError(t, err)
}

func TestValidateSwapConfig(t *testing.T) {
	c, err := getConfig(testSwapConfigName)
	require.NoError(t, err)
	err = validateConfig(c)
	require.NoError(t, err)

	c, err = getConfig(testSwapErrConfigName)
	require.NoError(t, err)
	err = validateConfig(c)
	require.Error(t, err)
}

func TestExecuteOptions(t *testing.T) {
	defExecuteTimeout := time.Duration(100)
	defOpts := ExecuteOptions{
		ExecuteTimeout: &defExecuteTimeout,
	}

	executeTimeout := time.Duration(10)

	// 1. full ExecOptions
	fullExecOptions := ExecuteOptions{
		ExecuteTimeout: &executeTimeout,
	}

	et, err := fullExecOptions.EffExecuteTimeout(defOpts)
	require.EqualValues(t, *fullExecOptions.ExecuteTimeout, et)
	require.NoError(t, err)

	// 2. empty ExecOptions
	emptyExecOptions := ExecuteOptions{}
	et, err = emptyExecOptions.EffExecuteTimeout(defOpts)
	require.EqualValues(t, *defOpts.ExecuteTimeout, et)
	require.NoError(t, err)

	// 4. check that we don't override default values occasionally
	require.EqualValues(t, defExecuteTimeout, *defOpts.ExecuteTimeout)
}

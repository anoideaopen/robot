//go:build integration
// +build integration

//nolint:unused
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/atomyze-foundation/common-component/loggerhlp"
	"github.com/atomyze-foundation/foundation/core/types"
	"github.com/atomyze-foundation/foundation/core/types/big"
	"github.com/atomyze-foundation/robot/chrobot"
	"github.com/atomyze-foundation/robot/config"
	"github.com/atomyze-foundation/robot/helpers/ntesting"
	"github.com/atomyze-foundation/robot/hlf"
	"github.com/atomyze-foundation/robot/hlf/sdkwrapper/service"
	"github.com/atomyze-foundation/robot/hlf/sdkwrapper/wallet"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	fsdkConfig "github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/newity/glog"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
	"golang.org/x/sync/errgroup"
)

func TestGetLedgerH(t *testing.T) {
	ciData := ntesting.CI(t)

	channels := []string{ciData.HlfFiatChannel}
	if ciData.HlfDoSwapTests {
		channels = append(channels, ciData.HlfCcChannel)
	}
	if ciData.HlfDoMultiSwapTests {
		channels = append(channels, ciData.HlfIndustrialChannel)
	}
	if ciData.HlfNoCcChannel != "" {
		channels = append(channels, ciData.HlfNoCcChannel)
	}

	for _, ch := range channels {
		if ch == "" {
			continue
		}
		h, err := getLedgerH(ciData, ch)
		require.NoError(t, err)
		require.True(t, h > 0)
		fmt.Println("h for", ch, h)
	}
}

func TestGetOwners(t *testing.T) {
	t.Skip("ATMCORE-6790")
	ciData := ntesting.CI(t)

	_, err := ntesting.GetFiatOwner(context.Background(), ciData)
	require.NoError(t, err)

	if ciData.HlfDoSwapTests {
		_, err := ntesting.GetCcOwner(context.Background(), ciData)
		require.NoError(t, err)
	}

	if ciData.HlfDoMultiSwapTests {
		_, err := ntesting.GetIndustrialOwner(context.Background(), ciData)
		require.NoError(t, err)
	}
}

func TestRobots(t *testing.T) {
	t.Skip("ATMCORE-6790")
	ciData := ntesting.CI(t)

	cfg, err := createConfig(t, ciData)
	require.NoError(t, err)

	l, err := createLogger(cfg, ciData.HlfProfile)
	require.NoError(t, err)

	eg, ctx := errgroup.WithContext(glog.NewContext(context.Background(), l))

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	cryptoMng, err := createCryptoManager(ctx, cfg, ciData.HlfProfile)
	require.NoError(t, err)

	robots, err := createRobots(ctx, cfg, ciData.HlfProfile, cryptoMng)
	require.NoError(t, err)
	require.NotEmpty(t, robots)

	createRunRobot := func(ctx context.Context, r *chrobot.ChRobot) func() error {
		return func() error {
			return runTestRobot(ctx, r)
		}
	}

	for _, r := range robots {
		eg.Go(createRunRobot(ctx, r))
	}

	eg.Go(func() error {
		if err := checkRobots(ctx, cfg, ciData); err != nil {
			return err
		}
		cancel()
		return nil
	})

	err = eg.Wait()
	require.NoError(t, err)
}

func runTestRobot(ctx context.Context, r *chrobot.ChRobot) error {
	l := glog.FromContext(ctx)
	l.Infof("run robot for:%s", r.ChName())
	err := r.Run(ctx)
	l.Infof("stopped robot for:%s, %v", r.ChName(), err)

	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}

func createConfig(_ *testing.T, ciData ntesting.CiTestData) (*config.Config, error) {
	fiatChannelH, err := getLedgerH(ciData, ciData.HlfFiatChannel)
	if err != nil {
		return nil, err
	}

	var ccRobot, industrialRobot *config.Robot

	fiatRobot := &config.Robot{
		ChName:            ciData.HlfFiatChannel,
		CollectorsBufSize: 1,
		SrcChannels: []*config.SrcChannel{
			{
				ChName:       ciData.HlfFiatChannel,
				InitBlockNum: &fiatChannelH,
			},
		},
	}

	if ciData.HlfDoSwapTests {
		ccChannelH, err := getLedgerH(ciData, ciData.HlfCcChannel)
		if err != nil {
			return nil, err
		}

		fiatRobot.SrcChannels = append(fiatRobot.SrcChannels, &config.SrcChannel{
			ChName:       ciData.HlfCcChannel,
			InitBlockNum: &ccChannelH,
		})

		ccRobot = &config.Robot{
			ChName:            ciData.HlfCcChannel,
			CollectorsBufSize: 1,
			SrcChannels: []*config.SrcChannel{
				{
					ChName:       ciData.HlfCcChannel,
					InitBlockNum: &ccChannelH,
				},
				{
					ChName:       ciData.HlfFiatChannel,
					InitBlockNum: &fiatChannelH,
				},
			},
		}
	}

	if ciData.HlfDoMultiSwapTests {
		industrialChannelH, err := getLedgerH(ciData, ciData.HlfIndustrialChannel)
		if err != nil {
			return nil, err
		}

		fiatRobot.SrcChannels = append(fiatRobot.SrcChannels, &config.SrcChannel{
			ChName:       ciData.HlfIndustrialChannel,
			InitBlockNum: &industrialChannelH,
		})

		industrialRobot = &config.Robot{
			ChName:            ciData.HlfIndustrialChannel,
			CollectorsBufSize: 1,
			SrcChannels: []*config.SrcChannel{
				{
					ChName:       ciData.HlfIndustrialChannel,
					InitBlockNum: &industrialChannelH,
				},
				{
					ChName:       ciData.HlfFiatChannel,
					InitBlockNum: &fiatChannelH,
				},
			},
		}
	}

	robots := []*config.Robot{fiatRobot}
	if ccRobot != nil {
		robots = append(robots, ccRobot)
	}
	if industrialRobot != nil {
		robots = append(robots, industrialRobot)
	}

	executeTimeout := time.Duration(0)
	waitCommitAttempts := uint(1)
	waitCommitAttemptTimeout := time.Duration(0)

	return &config.Config{
		LogLevel:               "debug",
		LogType:                "std",
		ProfilePath:            ciData.HlfProfilePath,
		UserName:               ciData.HlfUserName,
		UseSmartBFT:            ciData.HlfUseSmartBFT,
		TxSwapPrefix:           "swaps",
		TxMultiSwapPrefix:      "multi_swap",
		TxPreimagePrefix:       "batchTransactions",
		DelayAfterChRobotError: time.Second,
		DefaultBatchLimits: &config.BatchLimits{
			BatchBlocksCountLimit: 1,
			BatchTimeoutLimit:     time.Second,
		},

		RedisStorage: &config.RedisStorage{
			DBPrefix: "citests_",
			Addr:     []string{ciData.RedisAddr},
			Password: ciData.RedisPass,
			WithTLS:  false,
		},

		PromMetrics: nil,
		CryptoSrc:   config.LocalCryptoSrc,
		/*
			ProfilePath:            "/Users/kirill/Projects/nt/atmz/robot/dev-data/hlf-test-stage-04-vault/connection.yaml",
			CryptoSrc: config.VaultCryptoSrc,
			VaultCryptoSettings: &config.VaultCryptoSettings{
				VaultToken:              "123",
				UseRenewableVaultTokens: false,
				VaultAddress:            "http://localhost:8200",
				VaultAuthPath:           "",
				VaultRole:               "",
				VaultServiceTokenPath:   "",
				VaultNamespace:          "kv/",
				UserCert:                "backend@atomyzeMSP-cert.pem",
			},
		*/
		Robots: robots,
		DefaultRobotExecOpts: config.ExecuteOptions{
			ExecuteTimeout:           &executeTimeout,
			WaitCommitAttempts:       &waitCommitAttempts,
			WaitCommitAttemptTimeout: &waitCommitAttemptTimeout,
		},
	}, nil
}

// getLedgerH returns ledger height for channel.
func getLedgerH(ciData ntesting.CiTestData, ch string) (uint64, error) {
	if ch == "" {
		return 1, nil
	}

	tmpSdk, err := fabsdk.New(
		fsdkConfig.FromFile(ciData.HlfProfilePath))
	if err != nil {
		return 0, errors.WithStack(err)
	}
	defer tmpSdk.Close()

	lClient, err := ledger.New(tmpSdk.ChannelContext(ch,
		fabsdk.WithUser(ciData.HlfUserName),
		fabsdk.WithOrg(ciData.HlfProfile.OrgName)))
	if err != nil {
		return 0, errors.WithStack(err)
	}
	resp, err := lClient.QueryInfo()
	if err != nil {
		return 0, errors.WithStack(err)
	}

	return resp.BCI.Height, nil
}

// checkRobots checks that the robot has completed swap, multiswap, batches.
func checkRobots(ctx context.Context, _ *config.Config, ciData ntesting.CiTestData) error {
	l, err := loggerhlp.CreateLogger("std", "info")
	if err != nil {
		return err
	}

	ctx = glog.NewContext(ctx, l)

	newUser, emittedSum, err := emitFiatScenario(ctx, ciData)
	if err != nil {
		return err
	}

	if ciData.HlfDoSwapTests {
		if err := swapScenario(ctx, ciData, newUser, emittedSum); err != nil {
			return err
		}
	}

	if ciData.HlfDoMultiSwapTests {
		if err := multiSwapScenario(ctx, ciData); err != nil {
			return err
		}
	}

	return nil
}

// emitFiatScenario creates users (fiat owner and regular user) and makes emission.
func emitFiatScenario(ctx context.Context, ciData ntesting.CiTestData) (*wallet.User, uint64, error) {
	l := glog.FromContext(ctx)

	// create fiat owner and tokens recipient
	l.Infof("create fiat owner")
	fiatOwner, err := ntesting.GetFiatOwner(ctx, ciData)
	if err != nil {
		return nil, 0, err
	}
	l.Infof("fiat owner created")

	l.Infof("create user")
	u, err := ntesting.CreateTestUser(ctx, ciData, "user1")
	if err != nil {
		return nil, 0, err
	}
	l.Infof("user created")

	l.Infof("get user balance")
	oldFiatB, err := ntesting.GetBalance(ctx, u, ciData.HlfFiatChannel, ciData.HlfFiatChannel)
	if err != nil {
		return nil, 0, err
	}
	l.Infof("user balance is: %d", oldFiatB)

	// start emit
	const s0, s1, s2 = 100, 50, 30
	const (
		emitTemplate, emittedTemplate = "emit %d to user", "emitted %d to user"
	)
	l.Infof(emitTemplate, s0)
	if _, err := ntesting.EmitFiat(ctx, fiatOwner, u, s0, ciData.HlfFiatChannel, ciData.HlfFiatChannel); err != nil {
		return nil, 0, err
	}
	l.Infof(emittedTemplate, s0)

	l.Infof(emitTemplate, s1)
	if _, err := ntesting.EmitFiat(ctx, fiatOwner, u, s1, ciData.HlfFiatChannel, ciData.HlfFiatChannel); err != nil {
		return nil, 0, err
	}
	l.Infof(emittedTemplate, s1)

	l.Infof(emitTemplate, s2)
	if _, err := ntesting.EmitFiat(ctx, fiatOwner, u, s2, ciData.HlfFiatChannel, ciData.HlfFiatChannel); err != nil {
		return nil, 0, err
	}
	l.Infof(emittedTemplate, s2)

	// wait emitted balance
	l.Infof("check user balance after emissions")
	if err := waitUserBalance(ctx, u, ciData.HlfFiatChannel, ciData.HlfFiatChannel, s0+s1+s2+oldFiatB); err != nil {
		return nil, 0, err
	}
	l.Infof("user balance is: %d", s0+s1+s2+oldFiatB)

	return u, s0 + s1 + s2, nil
}

// swapScenario makes swaps fiat -> cc, cc -> fiat.
func swapScenario(ctx context.Context, ciData ntesting.CiTestData, user *wallet.User, emittedSum uint64) error {
	l := glog.FromContext(ctx)
	if emittedSum == 0 {
		return errors.New("emitted fiat balance must be greater than 0")
	}

	// swap fiat to cc
	var swapSum0 uint64 = 10
	if swapSum0 > emittedSum {
		swapSum0 = emittedSum
	}

	l.Infof("swap %d fiats to cc", swapSum0)
	if err := doSwap(ctx, user, ciData.HlfFiatChannel, ciData.HlfCcChannel, swapSum0); err != nil {
		return err
	}
	l.Infof("swapped %d fiats to cc", swapSum0)

	// get CC owner and set rate
	ccOwner, err := ntesting.GetCcOwner(ctx, ciData)
	if err != nil {
		return err
	}
	l.Infof("cc owner created")

	if err := setRate(ctx, ccOwner, ciData.HlfCcChannel, ciData.HlfFiatChannel, 100000000); err != nil {
		return err
	}

	// transfer from cc AllowedBalance to main balance
	l.Infof("buy %d cc in cc for fiat tokens (allowed -> real transfer)", swapSum0)
	if err := buy(ctx, user, ciData.HlfCcChannel, ciData.HlfFiatChannel, swapSum0); err != nil {
		return err
	}
	l.Info("tokens have been successfully transferred to the real balance")

	return nil
}

// multiSwapScenario makes multiswap from indistrial token to fiat.
func multiSwapScenario(ctx context.Context, ciData ntesting.CiTestData) error {
	l := glog.FromContext(ctx)

	// create user
	l.Infof("create industrial owner")
	industrialOwner, err := ntesting.GetIndustrialOwner(ctx, ciData)
	if err != nil {
		return err
	}
	l.Infof("industrial owner created")

	// init industrial token
	_, err = industrialOwner.SignedInvoke(nil, ciData.HlfIndustrialChannel, ciData.HlfIndustrialChannel, "initialize")
	if err != nil {
		return err
	}

	l.Infof("get issuer balance")

	time.Sleep(time.Second * 1)

	oldBGr1, err := getIndustrialBalance(ctx, industrialOwner, ciData.HlfIndustrialChannel, ciData.HlfIndustrialChannel, ciData.HlfIndustrialGroup1)
	if err != nil {
		return err
	}
	l.Infof("issuer balance for group %s is: %d", ciData.HlfIndustrialGroup1, oldBGr1)
	if oldBGr1 == 0 {
		return errors.WithStack(fmt.Errorf("industrial balance for group %s must be greater than 0", ciData.HlfIndustrialGroup1))
	}

	oldBGr2, err := getIndustrialBalance(ctx, industrialOwner, ciData.HlfIndustrialChannel, ciData.HlfIndustrialChannel, ciData.HlfIndustrialGroup2)
	if err != nil {
		return err
	}
	l.Infof("issuer balance for group %s is: %d", ciData.HlfIndustrialGroup2, oldBGr2)
	if oldBGr2 == 0 {
		return errors.WithStack(fmt.Errorf("industrial balance for group %s must be greater than 0", ciData.HlfIndustrialGroup2))
	}

	// multiswap industrial to fiat
	swapSumGr1, swapSumGr2 := uint64(12), uint64(10)
	if swapSumGr1 > oldBGr1 {
		swapSumGr1 = oldBGr1
	}
	if swapSumGr2 > oldBGr2 {
		swapSumGr2 = oldBGr2
	}

	assets := types.MultiSwapAssets{
		Assets: []*types.MultiSwapAsset{
			{
				Group:  fmt.Sprintf("%s_%s", strings.ToUpper(ciData.HlfIndustrialChannel), ciData.HlfIndustrialGroup1),
				Amount: strconv.Itoa(int(swapSumGr1)),
			},
			{
				Group:  fmt.Sprintf("%s_%s", strings.ToUpper(ciData.HlfIndustrialChannel), ciData.HlfIndustrialGroup2),
				Amount: strconv.Itoa(int(swapSumGr2)),
			},
		},
	}
	assetsGroups := map[string]string{
		assets.Assets[0].Group: ciData.HlfIndustrialGroup1,
		assets.Assets[1].Group: ciData.HlfIndustrialGroup2,
	}
	l.Infof("multiswap industrial tokens (symbol %s, groups: %s, %s) to fiat",
		ciData.HlfIndustrialChannel, ciData.HlfIndustrialGroup1, ciData.HlfIndustrialGroup2)
	if err := doMultiSwap(ctx, industrialOwner, ciData.HlfIndustrialChannel, ciData.HlfFiatChannel, assets, assetsGroups); err != nil {
		return err
	}
	l.Info("swapped")

	return nil
}

// doSwap makes swap from 'chFrom' to 'chTo'.
func doSwap(ctx context.Context, u *wallet.User, chFrom, chTo string, swapSum0 uint64) error {
	oldB, err := ntesting.GetBalance(ctx, u, chFrom, chFrom)
	if err != nil {
		return err
	}

	oldAb, err := getAllowedBalance(ctx, u, chTo, chTo, chFrom)
	if err != nil {
		return err
	}

	swapKey, swapID, err := swapBegin(ctx, u, swapSum0, chFrom, chTo)
	if err != nil {
		return err
	}

	if err := waitSwap(ctx, u, chTo, chTo, swapID); err != nil {
		return err
	}

	if err := swapDone(u, chTo, chTo, swapID, swapKey); err != nil {
		return err
	}

	if err := waitUserAllowedBalance(ctx, u,
		chTo, chTo, swapSum0+oldAb, chFrom); err != nil {
		return err
	}

	if err := waitUserBalance(ctx, u, chFrom, chFrom, oldB-swapSum0); err != nil {
		return err
	}

	return nil
}

// doMultiSwap makes multiswap of assets 'assetsGroups' from 'chFrom' to 'chTo'.
func doMultiSwap(ctx context.Context, u *wallet.User, chFrom, chTo string, assets types.MultiSwapAssets, assetsGroups map[string]string) error {
	oldB, oldAB := make(map[string]uint64), make(map[string]uint64) // map[group]amount

	for _, a := range assets.Assets {
		b, err := getIndustrialBalance(ctx, u, chFrom, chFrom, assetsGroups[a.Group])
		if err != nil {
			return err
		}
		oldB[a.Group] = b

		ab, err := getAllowedBalance(ctx, u, chTo, chTo, a.Group)
		if err != nil {
			return err
		}
		oldAB[a.Group] = ab
	}

	swapKey, swapID, err := multiSwapBegin(ctx, u, assets, chFrom, chTo)
	if err != nil {
		return err
	}

	if err := waitMultiSwap(ctx, u, chTo, chTo, swapID); err != nil {
		return err
	}

	if err := multiSwapDone(u, chTo, chTo, swapID, swapKey); err != nil {
		return err
	}

	for _, a := range assets.Assets {
		amount, err := strconv.Atoi(a.Amount)
		if err != nil {
			return errors.WithStack(err)
		}

		// check destination ch allowed balance
		if err := waitUserAllowedBalance(ctx, u,
			chTo, chTo, uint64(amount)+oldAB[a.Group], a.Group); err != nil {
			return err
		}

		// check source ch real balance

		if err := waitIndustrialUserBalance(ctx, u, chFrom, chFrom, assetsGroups[a.Group], oldB[a.Group]-uint64(amount)); err != nil {
			return err
		}
	}

	return nil
}

// buy transfers tokens from allowed to real balance.
func buy(ctx context.Context, u *wallet.User, token, currency string, amount uint64) error {
	oldCcB, err := ntesting.GetBalance(ctx, u, token, token)
	if err != nil {
		return err
	}

	oldAb, err := getAllowedBalance(ctx, u, token, token, currency)
	if err != nil {
		return err
	}

	if err := buyToken(ctx, u, token, currency, amount); err != nil {
		return err
	}

	if err := waitUserBalance(ctx, u, token, token, oldCcB+amount); err != nil {
		return err
	}

	return waitUserAllowedBalance(ctx, u,
		token, token, oldAb-amount, currency)
}

// buyToken is a helper for 'buy' func.
func buyToken(_ context.Context, u *wallet.User, token, currency string, amount uint64) error {
	_, err := u.SignedInvoke(nil, token, token, "buyToken", strconv.Itoa(int(amount)), strings.ToUpper(currency))
	if err != nil {
		return errors.WithStack(err)
	}
	return err
}

// swapBegin creates first part of the swap.
func swapBegin(_ context.Context, u *wallet.User, amount uint64, fromCh, toCh string) (string, string, error) {
	swapKey := "123"
	swapHash := sha3.Sum256([]byte(swapKey))
	nonce := service.GetNonceInt64()
	res, err := u.SignedInvoke(&nonce,
		fromCh, fromCh, "swapBegin",
		strings.ToUpper(fromCh), strings.ToUpper(toCh), strconv.Itoa(int(amount)), hex.EncodeToString(swapHash[:]))
	if err != nil {
		return "", "", errors.WithStack(err)
	}
	return swapKey, res.Event.TxID, nil
}

// swapBegin creates fisrt part of the multiswap.
func multiSwapBegin(_ context.Context, u *wallet.User, assets types.MultiSwapAssets, fromCh, toCh string) (string, string, error) {
	swapKey := "123"
	swapHash := sha3.Sum256([]byte(swapKey))

	b, err := json.Marshal(assets)
	if err != nil {
		return "", "", err
	}

	res, err := u.SignedInvoke(nil,
		fromCh, fromCh, "multiSwapBegin",
		strings.ToUpper(fromCh), string(b), strings.ToUpper(toCh), hex.EncodeToString(swapHash[:]))
	if err != nil {
		return "", "", errors.WithStack(err)
	}
	return swapKey, res.Event.TxID, nil
}

// swapDone publishes swap key in specified channel.
func swapDone(u *wallet.User, ch, cc, swapID, swapKey string) error {
	_, err := u.Invoke(ch, cc, "swapDone", swapID, swapKey)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// swapDone publishes multiswap key in specified channel.
func multiSwapDone(u *wallet.User, ch, cc, swapID, swapKey string) error {
	_, err := u.Invoke(ch, cc, "multiSwapDone", swapID, swapKey)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// waitSwap waits for swap in specified channel.
// Blocking func.
func waitSwap(ctx context.Context, u *wallet.User, ch, cc, swapID string) error {
	for {
		if ctx.Err() != nil {
			return errors.WithStack(ctx.Err())
		}
		_, err := u.Query(ch, cc, "swapGet", swapID)
		if err == nil {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
}

// waitMultiSwap waits for multiswap in specified channel.
// Blocking func.
func waitMultiSwap(ctx context.Context, u *wallet.User, ch, cc, swapID string) error {
	for {
		if ctx.Err() != nil {
			return errors.WithStack(ctx.Err())
		}
		_, err := u.Query(ch, cc, "multiSwapGet", swapID)
		if err == nil {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
}

// getAllowedBalance returns allowed balance of user.
func getAllowedBalance(_ context.Context, user *wallet.User, channelName, chainCodeName, currency string) (uint64, error) {
	tokenBalanceResponse, err := user.Query(channelName, chainCodeName, "allowedBalanceOf", user.Addr(), strings.ToUpper(currency))
	if err != nil {
		return 0, err
	}
	return ntesting.ParseBalance(tokenBalanceResponse)
}

// getIndustrialBalance returns balance of user in industrial token.
func getIndustrialBalance(_ context.Context, user *wallet.User, channelName, chainCodeName, group string) (uint64, error) {
	tokenBalanceResponse, err := user.Query(channelName, chainCodeName, "industrialBalanceOf", user.Addr())
	if err != nil {
		return 0, errors.WithStack(err)
	}

	var result map[string]string
	if err = json.Unmarshal(tokenBalanceResponse, &result); err != nil {
		return 0, errors.WithStack(err)
	}

	v, ok := result[group]
	if !ok {
		return 0, errors.Errorf("group not found %s", group)
	}
	vv, err := strconv.Atoi(v)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	return uint64(vv), nil
}

// waitUserBalance waits balance of user in specified channel.
// Blocking func.
func waitUserBalance(ctx context.Context, u *wallet.User, ch, cc string, expectedBalance uint64) error {
	return waitBalance(ctx, expectedBalance, func() (uint64, error) {
		return ntesting.GetBalance(ctx, u, ch, cc)
	})
}

// waitUserBalance waits balance of user in specified industrial channel.
// Blocking func.
func waitIndustrialUserBalance(ctx context.Context, u *wallet.User, ch, cc string, group string, expectedBalance uint64) error {
	return waitBalance(ctx, expectedBalance, func() (uint64, error) {
		return getIndustrialBalance(ctx, u, ch, cc, group)
	})
}

// waitUserBalance waits allowed balance of user in specified channel.
// Blocking func.
func waitUserAllowedBalance(ctx context.Context, u *wallet.User, ch, cc string, expectedBalance uint64, token string) error {
	return waitBalance(ctx, expectedBalance, func() (uint64, error) {
		return getAllowedBalance(ctx, u, ch, cc, token)
	})
}

// waitUserBalance is a helper func that waits balance of user in specified channel using provided callback 'getBalance' for queries.
// Blocking func.
func waitBalance(ctx context.Context, expectedBalance uint64, getBalance func() (uint64, error)) error {
	log := glog.FromContext(ctx)
	for {
		if ctx.Err() != nil {
			return errors.WithStack(ctx.Err())
		}
		b, err := getBalance()
		if err != nil {
			if hlf.IsEndorsementMismatchErr(err) {
				log.Infof("endorser mismatch %s", err)
				continue
			}
			return err
		}
		log.Infof("balance is %v", b)
		if b == expectedBalance {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
}

// setRate sets exchange rate.
func setRate(_ context.Context, u *wallet.User, token, currency string, rate uint64) error {
	isSet, err := isRateSet(u, token, strings.ToUpper(currency), rate)
	if err != nil {
		return err
	}
	if isSet {
		return nil
	}
	_, err = u.SignedInvoke(nil, token, token, "setRate", "buyToken", strings.ToUpper(currency), strconv.Itoa(int(rate)))
	if err != nil {
		return errors.WithStack(err)
	}
	_, err = u.SignedInvoke(nil, token, token, "setRate", "buyBack", strings.ToUpper(currency), strconv.Itoa(int(rate)))
	if err != nil {
		return errors.WithStack(err)
	}

	for {
		ok, err := isRateSet(u, token, strings.ToUpper(currency), rate)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
}

func isRateSet(u *wallet.User, token, currency string, rateValue uint64) (bool, error) {
	var meta struct {
		Rates []struct {
			DealType string   `json:"deal_type"`
			Currency string   `json:"currency"`
			Rate     *big.Int `json:"rate"`
		} `json:"rates"`
	}

	resp, err := u.Query(token, token, "metadata")
	if err != nil {
		return false, errors.WithStack(err)
	}

	if err := json.Unmarshal(resp, &meta); err != nil {
		return false, errors.WithStack(err)
	}

	var buyTokenFound, buyBackFound bool
	for _, rate := range meta.Rates {
		if rate.DealType == "buyToken" &&
			rate.Currency == currency &&
			rate.Rate.Uint64() == rateValue {
			buyTokenFound = true
		}
		if rate.DealType == "buyBack" &&
			rate.Currency == currency &&
			rate.Rate.Uint64() == rateValue {
			buyBackFound = true
		}

		if buyBackFound && buyTokenFound {
			return true, nil
		}
	}

	return false, nil
}

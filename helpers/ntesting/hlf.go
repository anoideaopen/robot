package ntesting

import (
	"context"
	"strconv"
	"strings"

	"github.com/anoideaopen/glog"
	"github.com/anoideaopen/robot/hlf/sdkwrapper/wallet"
	"github.com/pkg/errors"
)

// GetBalance returns balance of user.
func GetBalance(_ context.Context, user *wallet.User, channelName, chainCodeName string) (uint64, error) {
	tokenBalanceResponse, err := user.QueryWithRetryIfEndorsementMismatch(channelName, chainCodeName, "balanceOf", user.Addr())
	if err != nil {
		return 0, errors.WithStack(err)
	}
	return ParseBalance(tokenBalanceResponse)
}

// ParseBalance parses chaincode response for balance query.
func ParseBalance(balanceResponse []byte) (uint64, error) {
	tokenBalanceStr := strings.ReplaceAll(string(balanceResponse), "\"", "")
	b, err := strconv.ParseUint(tokenBalanceStr, 10, 64)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	return b, nil
}

func GetFiatOwner(_ context.Context, ciData CiTestData) (*wallet.User, error) {
	u, err := wallet.NewUser(
		ciData.HlfProfile.MspID,
		ciData.HlfCert,
		ciData.HlfSk,
		ciData.HlfProfilePath,
		"fiatOwner",
		ciData.HlfFiatOwnerKey)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return u, nil
}

func GetCcOwner(_ context.Context, ciData CiTestData) (*wallet.User, error) {
	u, err := wallet.NewUser(
		ciData.HlfProfile.MspID,
		ciData.HlfCert,
		ciData.HlfSk,
		ciData.HlfProfilePath,
		"ccOwner",
		ciData.HlfCcOwnerKey)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return u, nil
}

func GetIndustrialOwner(_ context.Context, ciData CiTestData) (*wallet.User, error) {
	u, err := wallet.NewUser(
		ciData.HlfProfile.MspID,
		ciData.HlfCert,
		ciData.HlfSk,
		ciData.HlfProfilePath,
		"industrialOwner",
		ciData.HlfIndustrialOwnerKey)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return u, nil
}

// CreateTestUser creates new user in ACL.
// It takes context, CiTestData and user ID.
// Returns new user and not nil err if user isn't created.
func CreateTestUser(_ context.Context, ciData CiTestData, userID string) (*wallet.User, error) {
	u, err := wallet.NewUser(
		ciData.HlfProfile.MspID,
		ciData.HlfCert,
		ciData.HlfSk,
		ciData.HlfProfilePath,
		userID,
		"")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return u, nil
}

// EmitFiat emits 'amount' fiats to the user 'u' using passed 'fiatOwner' in channel "ch" and chaincode "cc".
// Returns tx ID string and err (not nil) if emit operation failed.
// WARN calling this function several times during the formation of one block can lead to conflicts on some stands.
func EmitFiat(ctx context.Context, fiatOwner, u *wallet.User, amount uint64, ch, cc string) (string, error) {
	l := glog.FromContext(ctx)
	resp, err := fiatOwner.SignedInvoke(nil, ch, cc, "emit", u.Addr(), strconv.Itoa(int(amount)))
	if err != nil {
		return "", err
	}
	l.Infof("emitted for %s on %s-%s %v, txid: %s, bn: %v",
		u.ID,
		ch, cc,
		amount, resp.Event.TxID, resp.Event.BlockNumber)
	return resp.Event.TxID, nil
}

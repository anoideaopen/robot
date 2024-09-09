package swap

import (
	"encoding/json"
	"github.com/anoideaopen/foundation/core/types"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/hyperledger/fabric/integration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	ccCCUpper   = "CC"
	ccFiatUpper = "FIAT"

	fnEmit             = "emit"
	fnBalanceOf        = "balanceOf"
	fnAllowedBalanceOf = "allowedBalanceOf"
	fnSwapBegin        = "swapBegin"
	fnSwapDone         = "swapDone"

	emitAmount      = "1000"
	zeroAmount      = "0"
	defaultSwapHash = "7d4e3eec80026719639ed4dba68916eb94c7a49a053e05c8f9578fe4e5a3d7ea"
	defaultSwapKey  = "12345"
)

var _ = Describe("Robot swap tests", func() {
	var (
		ts client.TestSuite
	)

	BeforeEach(func() {
		ts = client.NewTestSuite(components)
	})
	AfterEach(func() {
		ts.ShutdownNetwork()
	})

	var (
		channels = []string{cmn.ChannelAcl, cmn.ChannelCC, cmn.ChannelFiat}
		user     *client.UserFoundation
	)
	BeforeEach(func() {
		By("start redis")
		ts.StartRedis()
	})
	BeforeEach(func() {
		ts.InitNetwork(channels, integration.GatewayBasePort)
		ts.DeployChaincodes()
	})
	BeforeEach(func() {
		By("start robot")
		ts.StartRobot()
	})
	AfterEach(func() {
		By("stop robot")
		ts.StopRobot()
		By("stop redis")
		ts.StopRedis()
	})

	BeforeEach(func() {
		By("add admin to acl")
		ts.AddAdminToACL()

		By("add user to acl")
		var err error
		user, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
		Expect(err).NotTo(HaveOccurred())

		ts.AddUser(user)

		By("emit tokens")
		ts.TxInvokeWithSign(
			cmn.ChannelFiat,
			cmn.ChannelFiat,
			ts.Admin(),
			fnEmit,
			"",
			client.NewNonceByTime().Get(),
			user.AddressBase58Check,
			emitAmount,
		).CheckErrorIsNil()

		By("emit check")
		ts.Query(
			cmn.ChannelFiat,
			cmn.ChannelFiat,
			fnBalanceOf,
			user.AddressBase58Check,
		).CheckBalance(emitAmount)
	})

	It("swap test", func() {
		var (
		//buyAmount       = "1"
		//rate            = "100000000"
		)

		By("swap from fiat to cc")
		By("swap begin")
		swapBeginTxID := ts.TxInvokeWithSign(
			cmn.ChannelFiat,
			cmn.ChannelFiat,
			user,
			fnSwapBegin,
			"",
			client.NewNonceByTime().Get(),
			ccFiatUpper,
			ccCCUpper,
			emitAmount,
			defaultSwapHash,
		).TxID()
		Expect(swapBeginTxID).ToNot(BeEmpty())

		By("swap get")
		ts.SwapGet(cmn.ChannelCC, cmn.ChannelCC, client.SfnSwapGet, swapBeginTxID)

		By("check balance 1")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnBalanceOf, user.AddressBase58Check).
			CheckBalance(zeroAmount)

		By("check allowed balance 1")
		ts.Query(cmn.ChannelCC, cmn.ChannelCC, fnAllowedBalanceOf, user.AddressBase58Check, ccFiatUpper).
			CheckBalance(zeroAmount)

		By("swap done")
		ts.NBTxInvoke(cmn.ChannelCC, cmn.ChannelCC, fnSwapDone, swapBeginTxID, defaultSwapKey).
			CheckErrorIsNil()

		By("check balance 2")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnBalanceOf, user.AddressBase58Check).
			CheckBalance(zeroAmount)

		By("check allowed balance 2")
		ts.Query(cmn.ChannelCC, cmn.ChannelCC, fnAllowedBalanceOf, user.AddressBase58Check, ccFiatUpper).
			CheckBalance(emitAmount)
	})

	It("Multiswap test", func() {
		By("multiswap begin")
		assets, err := json.Marshal(types.MultiSwapAssets{
			Assets: []*types.MultiSwapAsset{
				{
					Group:  ccFiatUpper,
					Amount: emitAmount,
				},
			},
		})
		Expect(err).NotTo(HaveOccurred())
		swapBeginTxID := ts.TxInvokeWithSign(
			cmn.ChannelFiat,
			cmn.ChannelFiat,
			user,
			"multiSwapBegin",
			"",
			client.NewNonceByTime().Get(),
			"FIAT",
			string(assets),
			"CC",
			defaultSwapHash,
		).TxID()
		Expect(swapBeginTxID).ToNot(BeEmpty())

		By("multiswap get 1")
		ts.SwapGet(cmn.ChannelCC, cmn.ChannelCC, client.SfnMultiSwapGet, swapBeginTxID)

		By("check balance 1")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnBalanceOf, user.AddressBase58Check).
			CheckBalance(zeroAmount)

		By("check allowed balance 1")
		ts.Query(cmn.ChannelCC, cmn.ChannelCC, fnAllowedBalanceOf, user.AddressBase58Check, ccFiatUpper).
			CheckBalance(zeroAmount)

		By("multiswap done")
		ts.NBTxInvoke(cmn.ChannelCC, cmn.ChannelCC, "multiSwapDone", swapBeginTxID, defaultSwapKey).
			CheckErrorIsNil()

		By("check balance 2")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnBalanceOf, user.AddressBase58Check).
			CheckBalance(zeroAmount)

		By("check allowed balance 2")
		ts.Query(cmn.ChannelCC, cmn.ChannelCC, fnAllowedBalanceOf, user.AddressBase58Check, ccFiatUpper).
			CheckBalance(emitAmount)
	})
})

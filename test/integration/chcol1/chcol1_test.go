package chcol1

import (
	"context"

	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/anoideaopen/robot/hlf"
	"github.com/anoideaopen/robot/test/unit/common"
	"github.com/hyperledger/fabric/integration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	userName                = "backend"
	channelWithoutChaincode = "empty"
)

var _ = Describe("Robot channel collector tests 1", func() {
	var (
		channels = []string{cmn.ChannelACL, cmn.ChannelFiat, channelWithoutChaincode}
		ts       *client.FoundationTestSuite

		networkFound *cmn.NetworkFoundation
	)

	BeforeEach(func() {
		ts = client.NewTestSuite(components)
	})
	AfterEach(func() {
		ts.ShutdownNetwork()
	})

	BeforeEach(func() {
		By("start redis")
		ts.StartRedis()
	})
	BeforeEach(func() {
		ts.InitNetwork(channels, integration.GossipBasePort)
		ts.DeployChaincodes()
		By("start robot")
		ts.StartRobot()

		networkFound = ts.NetworkFound
	})
	AfterEach(func() {
		By("stop robot")
		ts.StopRobot()
		By("stop redis")
		ts.StopRedis()
	})

	It("Channel collector create test", func() {
		ctx := context.Background()

		By("Test 1 new channel collector creator")
		chCr := hlf.NewChCollectorCreator(
			cmn.ChannelFiat,
			networkFound.ConnectionPath("User1"),
			userName,
			"Org1",
			common.DefaultPrefixes,
			1)
		Expect(chCr).NotTo(BeNil())

		dataReady := make(chan struct{}, 1)

		By("Test 1 new channel collector")
		chColl, err := chCr(ctx, dataReady, cmn.ChannelFiat, 10)
		Expect(err).NotTo(HaveOccurred())
		Expect(chColl).NotTo(BeNil())

		By("Test 1 channel collector close")
		chColl.Close()
		close(dataReady)
	})

	It("Channel collector create without chaincode test", func() {
		var blockNum uint64 = 0
		ctx := context.Background()

		By("Test 2 new channel collector creator")
		chCr := hlf.NewChCollectorCreator(
			channelWithoutChaincode,
			networkFound.ConnectionPath("User1"),
			userName,
			"Org1",
			common.DefaultPrefixes,
			1,
		)
		Expect(chCr).NotTo(BeNil())

		dataReady := make(chan struct{}, 1)

		By("Test 2 new channel collector")
		chColl, err := chCr(ctx, dataReady, channelWithoutChaincode, blockNum)
		Expect(err).NotTo(HaveOccurred())
		Expect(chColl).NotTo(BeNil())

		By("Test 2 checking block number")
		blockData, ok := <-chColl.GetData()
		Expect(ok).To(BeTrue())
		Expect(blockData.BlockNum).To(Equal(blockNum))

		By("Test 2 channel collector close")
		chColl.Close()
		close(dataReady)
	})
})

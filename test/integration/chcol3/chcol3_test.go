package chcol3

import (
	"context"
	"github.com/anoideaopen/common-component/loggerhlp"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/anoideaopen/glog"
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

var _ = Describe("Robot hlf tests", func() {
	var (
		channels = []string{cmn.ChannelAcl, cmn.ChannelFiat, channelWithoutChaincode}
		ts       client.TestSuite

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
		ts.InitNetwork(channels, integration.KafkaBasePort)
		ts.DeployChaincodes()

		By("start robot")
		ts.StartRobot()

		networkFound = ts.NetworkFound()
	})
	AfterEach(func() {
		By("stop robot")
		ts.StopRobot()
		By("stop redis")
		ts.StopRedis()
	})

	It("Bag SDK subscribe events test", func() {
		var (
			blockNum1 uint64 = 3
			blockNum2 uint64 = 1
		)

		By("Test 4 new logger & context")
		log, err := loggerhlp.CreateLogger("std", "debug")
		Expect(err).NotTo(HaveOccurred())
		logCtx := glog.NewContext(context.Background(), log)

		By("Test 4 first channel collector creator")
		chCr1 := hlf.NewChCollectorCreator(
			cmn.ChannelFiat,
			networkFound.ConnectionPath("User1"),
			userName,
			"Org1",
			common.DefaultPrefixes,
			1,
		)
		Expect(chCr1).NotTo(BeNil())

		dataReady := make(chan struct{}, 1)

		By("Test 4 first channel collector")
		chColl1, err := chCr1(logCtx, dataReady, cmn.ChannelFiat, blockNum1)
		Expect(err).NotTo(HaveOccurred())
		Expect(chColl1).NotTo(BeNil())

		By("Test 4 getting data from first channel collector")
		data1, ok := <-chColl1.GetData()
		Expect(ok).To(BeTrue())
		Expect(data1.BlockNum).To(Equal(blockNum1))

		By("Test 4 first channel collector close")
		chColl1.Close()

		By("Test 4 second channel collector creator")
		chCr2 := hlf.NewChCollectorCreator(
			"dst-2",
			networkFound.ConnectionPath("User1"),
			userName,
			"Org1",
			common.DefaultPrefixes,
			1,
		)
		Expect(chCr1).NotTo(BeNil())

		dataReady2 := make(chan struct{}, 1)

		By("Test 4 second channel collector")
		chColl2, err := chCr2(logCtx, dataReady2, cmn.ChannelFiat, blockNum2)

		By("Test 4 getting data from second channel collector")
		data2, ok := <-chColl2.GetData()
		Expect(ok).To(BeTrue())
		Expect(data2.BlockNum).To(Equal(blockNum2))

		By("Test 4 second channel collector close")
		chColl2.Close()
	})
})

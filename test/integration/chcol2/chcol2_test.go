package chcol2

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

var _ = Describe("Robot channel collector 2 tests", func() {
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
		ts.InitNetwork(channels, integration.IdemixBasePort)
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

	It("Get data test", func() {
		By("Test 3 new logger & context")
		log, err := loggerhlp.CreateLogger("std", "debug")
		Expect(err).NotTo(HaveOccurred())
		logCtx := glog.NewContext(context.Background(), log)

		By("Test 3 new channel collector creator")
		chCr := hlf.NewChCollectorCreator(
			cmn.ChannelFiat,
			networkFound.ConnectionPath("User1"),
			userName,
			"Org1",
			common.DefaultPrefixes,
			1,
		)
		Expect(chCr).NotTo(BeNil())

		dataReady := make(chan struct{}, 1)

		By("Test 3 new channel collector")
		chColl, err := chCr(logCtx, dataReady, cmn.ChannelFiat, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(chColl).NotTo(BeNil())

		const closeAfterNBlocks = 1

		By("Test 3 getting data")
		for i := 0; ; i++ {
			select {
			case data, ok := <-chColl.GetData():
				if ok {
					log.Info("data from block:", data.BlockNum)
				}

				if !ok && i >= closeAfterNBlocks {
					log.Info("channel closed i:", i)
					return
				}

				Expect(ok).To(BeTrue())
				Expect(data).NotTo(BeNil())

				if !data.IsEmpty() {
					log.Info("data for block:", data.BlockNum,
						" lens:", len(data.Txs), len(data.Swaps), len(data.MultiSwaps), len(data.SwapsKeys), len(data.MultiSwapsKeys))
				}

			default:
				log.Debug("need to wait")
				<-dataReady
			}

			// after closeAfterNBlocks - init stop
			if i == closeAfterNBlocks {
				log.Info("schedule close")

				// don't wait
				go func() {
					chColl.Close()
					close(dataReady)
					log.Info("was closed")
				}()
			}
		}
	})
})

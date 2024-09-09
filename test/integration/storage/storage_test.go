package storage

import (
	"context"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/anoideaopen/robot/dto/stordto"
	"github.com/anoideaopen/robot/storage/redis"
	"github.com/hyperledger/fabric/integration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	dbPrefix = "my1"
	channel1 = "ch1"
	channel2 = "ch2"
)

var _ = Describe("Robot storage tests", func() {
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
		channels = []string{cmn.ChannelAcl, cmn.ChannelCC, cmn.ChannelFiat, cmn.ChannelIndustrial}
	)
	BeforeEach(func() {
		By("start redis")
		ts.StartRedis()
	})
	BeforeEach(func() {
		ts.InitNetwork(channels, integration.SBEBasePort)
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

	It("Save Load checkpoint test", func() {
		ctx := context.Background()

		redisDbAddress := ts.NetworkFound().Robot.RedisAddresses[0]

		stor1, err := redis.NewStorage(
			ctx,
			[]string{redisDbAddress},
			"",
			false,
			nil,
			dbPrefix,
			channel1,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(stor1).NotTo(BeNil())

		stor2, err := redis.NewStorage(
			ctx,
			[]string{redisDbAddress},
			"",
			false,
			nil,
			dbPrefix,
			channel2,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(stor2).NotTo(BeNil())

		chp1, err := stor1.SaveCheckPoints(ctx, &stordto.ChCheckPoint{
			Ver:                     1,
			SrcCollectFromBlockNums: map[string]uint64{"fiat": 123},
			MinExecBlockNum:         111,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(chp1).NotTo(BeNil())

		chp2, err := stor2.SaveCheckPoints(ctx, &stordto.ChCheckPoint{
			Ver:                     20,
			SrcCollectFromBlockNums: map[string]uint64{"fiat": 123},
			MinExecBlockNum:         222,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(chp2).NotTo(BeNil())

		chp1, ok, err := stor1.LoadCheckPoints(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
		Expect(chp1).NotTo(BeNil())

		chp2, ok, err = stor2.LoadCheckPoints(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
		Expect(chp2).NotTo(BeNil())

		Expect(chp1.Ver).NotTo(Equal(chp2.Ver))
		Expect(chp1.MinExecBlockNum).NotTo(Equal(chp2.MinExecBlockNum))
	})

	It("Different channels checkpoints", func() {
		ctx := context.Background()

		redisDbAddress := ts.NetworkFound().Robot.RedisAddresses[0]

		stor1, err := redis.NewStorage(
			ctx,
			[]string{redisDbAddress},
			"",
			false,
			nil,
			dbPrefix,
			channel1,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(stor1).NotTo(BeNil())

		stor2, err := redis.NewStorage(
			ctx,
			[]string{redisDbAddress},
			"",
			false,
			nil,
			dbPrefix,
			channel2,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(stor2).NotTo(BeNil())

		chp1, err := stor1.SaveCheckPoints(ctx, &stordto.ChCheckPoint{
			Ver:                     1,
			SrcCollectFromBlockNums: map[string]uint64{"fiat": 123},
			MinExecBlockNum:         111,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(chp1).NotTo(BeNil())

		chp2, err := stor2.SaveCheckPoints(ctx, &stordto.ChCheckPoint{
			Ver:                     20,
			SrcCollectFromBlockNums: map[string]uint64{"fiat": 123},
			MinExecBlockNum:         222,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(chp2).NotTo(BeNil())

		chp1, ok, err := stor1.LoadCheckPoints(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
		Expect(chp1).NotTo(BeNil())

		chp2, ok, err = stor2.LoadCheckPoints(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
		Expect(chp2).NotTo(BeNil())

		Expect(chp1.Ver).NotTo(Equal(chp2.Ver))
		Expect(chp1.MinExecBlockNum).NotTo(Equal(chp2.MinExecBlockNum))
	})
})

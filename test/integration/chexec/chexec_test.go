package chexec

import (
	"context"
	"encoding/hex"
	"errors"
	"github.com/anoideaopen/robot/hlf/hlfprofile"
	"github.com/hyperledger/fabric/integration"
	"time"

	"github.com/anoideaopen/common-component/loggerhlp"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/anoideaopen/glog"
	"github.com/anoideaopen/robot/dto/executordto"
	"github.com/anoideaopen/robot/helpers/ntesting"
	"github.com/anoideaopen/robot/hlf"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/common/selection/fabricselection"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	userName                = "backend"
	channelWithoutChaincode = "empty"
)

var _ = Describe("Robot channel executor tests", func() {
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
		channels = []string{cmn.ChannelAcl, cmn.ChannelFiat, channelWithoutChaincode}
	)
	BeforeEach(func() {
		By("start redis")
		ts.StartRedis()
	})
	BeforeEach(func() {
		ts.InitNetwork(channels, integration.NWOBasePort)
		ts.DeployChaincodes()
	})
	BeforeEach(func() {
		By("start robot")
		ts.StopRobot()
	})
	AfterEach(func() {
		By("stop robot")
		ts.StopRobot()
		By("stop redis")
		ts.StopRedis()
	})

	It("Channel executor without chaincode test", func() {
		ctx := context.Background()

		ccCr := hlf.NewChExecutorCreator(
			channelWithoutChaincode,
			ts.NetworkFound().ConnectionPath("User1"),
			userName,
			"Org1",
			hlf.ExecuteOptions{
				ExecuteTimeout: 0 * time.Second,
			},
		)
		Expect(ccCr).NotTo(BeNil())

		chExec, err := ccCr(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(chExec).NotTo(BeNil())

		_, err = chExec.Execute(ctx, &executordto.Batch{
			Txs: [][]byte{
				{1, 2, 3, 4},
			},
		}, 0)
		Expect(err).To(HaveOccurred())

		var dErr fabricselection.DiscoveryError
		Expect(errors.As(err, &dErr)).To(BeTrue())
		Expect(dErr.IsTransient()).To(BeTrue())

		chExec.Close()
	})

	// Test was copied from original channel executor integration test, but not working
	// ToDo - fix test
	PIt("Channel executor execute test", func() {
		log, err := loggerhlp.CreateLogger("std", "debug")
		Expect(err).NotTo(HaveOccurred())
		logCtx := glog.NewContext(context.Background(), log)

		By("send fake txs, check them committed")
		ciData := ts.CiData()
		ciData.HlfProfile, err = hlfprofile.ParseProfile(ciData.HlfProfilePath)
		Expect(err).NotTo(HaveOccurred())

		ccCr := hlf.NewChExecutorCreator(
			cmn.ChannelFiat,
			ts.NetworkFound().ConnectionPath("User1"),
			userName,
			"Org1",
			hlf.ExecuteOptions{
				ExecuteTimeout: 0 * time.Second,
			},
		)
		Expect(ccCr).NotTo(BeNil())

		chExec, err := ccCr(logCtx)
		Expect(err).NotTo(HaveOccurred())
		Expect(chExec).NotTo(BeNil())

		firstBlockN, err := chExec.Execute(context.Background(), &executordto.Batch{
			Txs:        [][]byte{[]byte("123")},
			Swaps:      nil,
			MultiSwaps: nil,
			Keys:       nil,
			MultiKeys:  nil,
		}, 1)
		Expect(err).NotTo(HaveOccurred())

		secondBlockN, err := chExec.Execute(context.Background(), &executordto.Batch{
			Txs:        [][]byte{[]byte("123")},
			Swaps:      nil,
			MultiSwaps: nil,
			Keys:       nil,
			MultiKeys:  nil,
		}, firstBlockN+1)
		Expect(err).NotTo(HaveOccurred())
		if secondBlockN <= firstBlockN {
			err = errors.New("second block number is less or equals to first block number")
		}
		Expect(err).NotTo(HaveOccurred())

		user1, err := ntesting.CreateTestUser(context.Background(), ciData, "user")
		Expect(err).NotTo(HaveOccurred())

		fiatOwner, err := ntesting.GetFiatOwner(context.Background(), ciData)
		Expect(err).NotTo(HaveOccurred())

		var preimages [][]byte
		for i := 0; i < 3; i++ {
			txID, err := ntesting.EmitFiat(context.Background(), fiatOwner, user1, 1, ciData.HlfFiatChannel, ciData.HlfFiatChannel)
			Expect(err).NotTo(HaveOccurred())

			idBytes, err := hex.DecodeString(txID)
			Expect(err).NotTo(HaveOccurred())

			preimages = append(preimages, idBytes)
		}

		chExec1, err := ccCr(logCtx)
		Expect(err).NotTo(HaveOccurred())
		Expect(chExec).NotTo(BeNil())

		_, err = chExec1.Execute(context.Background(), &executordto.Batch{
			Txs:        preimages,
			Swaps:      nil,
			MultiSwaps: nil,
			Keys:       nil,
			MultiKeys:  nil,
		}, 1)
		Expect(err).NotTo(HaveOccurred())
	})
})

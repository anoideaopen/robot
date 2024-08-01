package hlf

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/anoideaopen/common-component/loggerhlp"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/anoideaopen/foundation/test/integration/cmn/fabricnetwork"
	"github.com/anoideaopen/foundation/test/integration/cmn/runner"
	"github.com/anoideaopen/glog"
	"github.com/anoideaopen/robot/hlf"
	"github.com/anoideaopen/robot/test/unit/common"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/hyperledger/fabric/integration/nwo/fabricconfig"
	runnerFbk "github.com/hyperledger/fabric/integration/nwo/runner"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/tedsuo/ifrit"
	ginkgomon "github.com/tedsuo/ifrit/ginkgomon_v2"
)

const (
	userName = "backend"

	ccCCUpper    = "CC"
	ccFiatUpper  = "FIAT"
	ccEmptyUpper = "EMPTY"

	fnEmit             = "emit"
	fnBalanceOf        = "balanceOf"
	fnAllowedBalanceOf = "allowedBalanceOf"
	fnSwapBegin        = "swapBegin"
	fnSwapGet          = "swapGet"
	fnSwapDone         = "swapDone"
	fnSetRate          = "setRate"
	fnBuyToken         = "buyToken"

	emitAmount      = "1000"
	zeroAmount      = "0"
	defaultSwapHash = "7d4e3eec80026719639ed4dba68916eb94c7a49a053e05c8f9578fe4e5a3d7ea"
	defaultSwapKey  = "12345"

	channelWithoutChaincode = "empty"
)

var _ = Describe("Robot hlf tests", func() {
	var (
		testDir          string
		cli              *docker.Client
		network          *nwo.Network
		networkProcess   ifrit.Process
		ordererProcesses []ifrit.Process
		peerProcesses    ifrit.Process
	)

	BeforeEach(func() {
		networkProcess = nil
		ordererProcesses = nil
		peerProcesses = nil
		var err error
		testDir, err = os.MkdirTemp("", "foundation")
		Expect(err).NotTo(HaveOccurred())

		cli, err = docker.NewClientFromEnv()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if networkProcess != nil {
			networkProcess.Signal(syscall.SIGTERM)
			Eventually(networkProcess.Wait(), network.EventuallyTimeout).Should(Receive())
		}
		if peerProcesses != nil {
			peerProcesses.Signal(syscall.SIGTERM)
			Eventually(peerProcesses.Wait(), network.EventuallyTimeout).Should(Receive())
		}
		if network != nil {
			network.Cleanup()
		}
		for _, ordererInstance := range ordererProcesses {
			ordererInstance.Signal(syscall.SIGTERM)
			Eventually(ordererInstance.Wait(), network.EventuallyTimeout).Should(Receive())
		}
		err := os.RemoveAll(testDir)
		Expect(err).NotTo(HaveOccurred())
	})

	var (
		channels         = []string{cmn.ChannelAcl, cmn.ChannelCC, cmn.ChannelFiat, cmn.ChannelIndustrial, channelWithoutChaincode}
		ordererRunners   []*ginkgomon.Runner
		redisProcess     ifrit.Process
		redisDB          *runner.RedisDB
		networkFound     *cmn.NetworkFoundation
		robotProc        ifrit.Process
		skiBackend       string
		skiRobot         string
		peer             *nwo.Peer
		admin            *client.UserFoundation
		user             *client.UserFoundation
		feeSetter        *client.UserFoundation
		feeAddressSetter *client.UserFoundation
	)
	BeforeEach(func() {
		By("start redis")
		redisDB = &runner.RedisDB{}
		redisProcess = ifrit.Invoke(redisDB)
		Eventually(redisProcess.Ready(), runnerFbk.DefaultStartTimeout).Should(BeClosed())
		Consistently(redisProcess.Wait()).ShouldNot(Receive())
	})
	BeforeEach(func() {
		networkConfig := nwo.MultiNodeSmartBFT()
		networkConfig.Channels = nil

		pchs := make([]*nwo.PeerChannel, 0, cap(channels))
		for _, ch := range channels {
			pchs = append(pchs, &nwo.PeerChannel{
				Name:   ch,
				Anchor: true,
			})
		}
		for _, peer := range networkConfig.Peers {
			peer.Channels = pchs
		}

		network = nwo.New(networkConfig, testDir, cli, StartPort(), components)
		cwd, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		network.ExternalBuilders = append(network.ExternalBuilders,
			fabricconfig.ExternalBuilder{
				Path:                 filepath.Join(cwd, ".", "externalbuilders", "binary"),
				Name:                 "binary",
				PropagateEnvironment: []string{"GOPROXY"},
			},
		)

		networkFound = cmn.New(network, channels)
		networkFound.Robot.RedisAddresses = []string{redisDB.Address()}
		networkFound.ChannelTransfer.RedisAddresses = []string{redisDB.Address()}

		networkFound.GenerateConfigTree()
		networkFound.Bootstrap()

		for _, orderer := range network.Orderers {
			runner := network.OrdererRunner(orderer)
			runner.Command.Env = append(runner.Command.Env, "FABRIC_LOGGING_SPEC=orderer.consensus.smartbft=debug:grpc=debug")
			ordererRunners = append(ordererRunners, runner)
			proc := ifrit.Invoke(runner)
			ordererProcesses = append(ordererProcesses, proc)
			Eventually(proc.Ready(), network.EventuallyTimeout).Should(BeClosed())
		}

		peerGroupRunner, _ := fabricnetwork.PeerGroupRunners(network)
		peerProcesses = ifrit.Invoke(peerGroupRunner)
		Eventually(peerProcesses.Ready(), network.EventuallyTimeout).Should(BeClosed())

		By("Joining orderers to channels")
		for _, channel := range channels {
			fabricnetwork.JoinChannel(network, channel)
		}

		By("Waiting for followers to see the leader")
		Eventually(ordererRunners[1].Err(), network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))
		Eventually(ordererRunners[2].Err(), network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))
		Eventually(ordererRunners[3].Err(), network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))

		By("Joining peers to channels")
		for _, channel := range channels {
			network.JoinChannel(channel, network.Orderers[0], network.PeersWithChannel(channel)...)
		}

		peer = network.Peer("Org1", "peer0")

		pathToPrivateKeyBackend := network.PeerUserKey(peer, "User1")
		skiBackend, err = cmn.ReadSKI(pathToPrivateKeyBackend)
		Expect(err).NotTo(HaveOccurred())

		pathToPrivateKeyRobot := network.PeerUserKey(peer, "User2")
		skiRobot, err = cmn.ReadSKI(pathToPrivateKeyRobot)
		Expect(err).NotTo(HaveOccurred())

		admin, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
		Expect(err).NotTo(HaveOccurred())
		Expect(admin.PrivateKeyBytes).NotTo(Equal(nil))

		feeSetter, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
		Expect(err).NotTo(HaveOccurred())
		Expect(feeSetter.PrivateKeyBytes).NotTo(Equal(nil))

		feeAddressSetter, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
		Expect(err).NotTo(HaveOccurred())
		Expect(feeAddressSetter.PrivateKeyBytes).NotTo(Equal(nil))

		cmn.DeployACL(network, components, peer, testDir, skiBackend, admin.PublicKeyBase58, admin.KeyType)
		cmn.DeployCC(network, components, peer, testDir, skiRobot, admin.AddressBase58Check)
		cmn.DeployFiat(network, components, peer, testDir, skiRobot,
			admin.AddressBase58Check, feeSetter.AddressBase58Check, feeAddressSetter.AddressBase58Check)
		cmn.DeployIndustrial(network, components, peer, testDir, skiRobot,
			admin.AddressBase58Check, feeSetter.AddressBase58Check, feeAddressSetter.AddressBase58Check)
	})
	BeforeEach(func() {
		By("start robot")
		robotRunner := networkFound.RobotRunner()
		robotProc = ifrit.Invoke(robotRunner)
		Eventually(robotProc.Ready(), network.EventuallyTimeout).Should(BeClosed())
	})
	AfterEach(func() {
		By("stop robot")
		if robotProc != nil {
			robotProc.Signal(syscall.SIGTERM)
			Eventually(robotProc.Wait(), network.EventuallyTimeout).Should(Receive())
		}
	})

	BeforeEach(func() {
		By("add admin to acl")
		client.AddUser(network, peer, network.Orderers[0], admin)

		By("add user to acl")
		var err error
		user, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
		Expect(err).NotTo(HaveOccurred())

		client.AddUser(network, peer, network.Orderers[0], user)

		By("emit tokens")
		client.TxInvokeWithSign(network, peer, network.Orderers[0],
			cmn.ChannelFiat, cmn.ChannelFiat, admin,
			fnEmit, "", client.NewNonceByTime().Get(), nil, user.AddressBase58Check, emitAmount)

		By("emit check")
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(emitAmount), nil),
			fnBalanceOf, user.AddressBase58Check)
	})

	It("Channel collector create test", func() {
		ctx := context.Background()

		chCr := hlf.NewChCollectorCreator(
			cmn.ChannelFiat,
			networkFound.ConnectionPath("User1"),
			userName,
			"Org1",
			common.DefaultPrefixes,
			1)
		Expect(chCr).NotTo(BeNil())

		dataReady := make(chan struct{}, 1)

		chColl, err := chCr(ctx, dataReady, ccFiatUpper, 10)
		Expect(err).NotTo(HaveOccurred())
		Expect(chColl).NotTo(BeNil())

		chColl.Close()
	})

	It("Channel collector create without chaincode test", func() {
		ctx := context.Background()

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

		chColl, err := chCr(ctx, dataReady, channelWithoutChaincode, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(chColl).NotTo(BeNil())

		blockData, ok := <-chColl.GetData()
		Expect(ok).To(BeTrue())
		Expect(blockData.BlockNum).To(Equal(uint64(0)))

		chColl.Close()
	})

	It("Get data test", func() {
		log, err := loggerhlp.CreateLogger("std", "debug")
		Expect(err).NotTo(HaveOccurred())
		logCtx := glog.NewContext(context.Background(), log)

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

		chColl, err := chCr(logCtx, dataReady, cmn.ChannelFiat, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(chColl).NotTo(BeNil())

		const closeAfterNBlocks = 1

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
					log.Info("was closed")
				}()
			}
		}
	})

	It("Bag SDK subscribe events test", func() {
		log, err := loggerhlp.CreateLogger("std", "debug")
		Expect(err).NotTo(HaveOccurred())
		logCtx := glog.NewContext(context.Background(), log)

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

		chColl1, err := chCr1(logCtx, dataReady, cmn.ChannelFiat, 3)
		Expect(err).NotTo(HaveOccurred())
		Expect(chColl1).NotTo(BeNil())

		data1, ok := <-chColl1.GetData()
		Expect(ok).To(BeTrue())
		Expect(data1.BlockNum).To(Equal(uint64(3)))

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
		chColl2, err := chCr2(logCtx, dataReady2, cmn.ChannelFiat, 1)

		data2, ok := <-chColl2.GetData()
		Expect(ok).To(BeTrue())
		Expect(data2.BlockNum).To(Equal(int64(data2.BlockNum)))
	})

	/*
		It("Channel executor without chaincode test", func() {
			ctx := context.Background()

			ccCr := hlf.NewChExecutorCreator(
				channelWithoutChaincode,
				networkFound.ConnectionPath("User1"),
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
		})
	*/
})

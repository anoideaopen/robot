package chexec

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"time"

	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/anoideaopen/foundation/test/integration/cmn/fabricnetwork"
	"github.com/anoideaopen/foundation/test/integration/cmn/runner"
	"github.com/anoideaopen/robot/dto/executordto"
	"github.com/anoideaopen/robot/hlf"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/common/selection/fabricselection"
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
	userName                = "backend"
	channelWithoutChaincode = "empty"
)

var _ = Describe("Robot channel executor tests", func() {
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
		channels         = []string{cmn.ChannelAcl, cmn.ChannelFiat, channelWithoutChaincode}
		ordererRunners   []*ginkgomon.Runner
		redisProcess     ifrit.Process
		redisDB          *runner.RedisDB
		networkFound     *cmn.NetworkFoundation
		robotProc        ifrit.Process
		skiBackend       string
		peer             *nwo.Peer
		admin            *client.UserFoundation
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
		cmn.DeployFiat(network, components, peer, testDir, skiBackend,
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

		chExec.Close()
	})

	/*
		It("Channel executor execute test", func() {
			log, err := loggerhlp.CreateLogger("std", "debug")
			Expect(err).NotTo(HaveOccurred())
			logCtx := glog.NewContext(context.Background(), log)

			By("send fake txs, check them committed")

			ccCr := hlf.NewChExecutorCreator(
				cmn.ChannelFiat,
				networkFound.ConnectionPath("User1"),
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

			// require.NoError(t, user.BalanceShouldBe(3))

		})
	*/
})

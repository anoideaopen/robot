package unit

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"testing"

	"github.com/anoideaopen/common-component/testshlp"
	"github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/glog"
	"github.com/anoideaopen/robot/collectorbatch"
	"github.com/anoideaopen/robot/dto/collectordto"
	"github.com/anoideaopen/robot/dto/executordto"
	"github.com/anoideaopen/robot/dto/parserdto"
	"github.com/anoideaopen/robot/helpers/ntesting"
	"github.com/anoideaopen/robot/hlf/parser"
	cmn "github.com/anoideaopen/robot/test/unit/common"
	"github.com/stretchr/testify/require"
	pb "google.golang.org/protobuf/proto"
)

func NoTestBatchSize(t *testing.T) {
	_ = ntesting.CI(t)
	ctx, log := testshlp.CreateCtxLogger(t)

	allBlockData := getBlocks(t, log)

	// real BlocksCountLimit is 1..10
	for bsize := 1; bsize < 20; bsize++ {
		maxDiff := float64(0)
		stat := make([]int, 100)
		bCount := 0
		var batch *collectorbatch.CBatch

		for _, blockData := range allBlockData {
			if batch == nil {
				batch = collectorbatch.NewBatch(ctx, "fiat", collectorbatch.Limits{
					BlocksCountLimit: uint(bsize),
				}, calcBS)
			}

			if ok, _ := batch.AddIfInLimit(blockData.chName, blockData.data); ok {
				continue
			}
			// collect stat
			execBatch, _ := batch.GetBatchForExec()
			actualSize, _ := calcBS(execBatch)
			diff, _ := calcDiff(actualSize, execBatch.PredictSize)
			if maxDiff < diff {
				maxDiff = diff
			}

			stat[int(math.Ceil(diff))]++
			bCount++

			batch = nil
		}

		log.Infof("bSize: %v maxDiff: %v%% allBatches: %v distribution:", bsize, maxDiff, bCount)
		for k, v := range stat {
			if v == 0 {
				continue
			}
			log.Infof("%v-%v%% - %v%%", k-1, k, v*100.0/bCount)
		}
		log.Info("")
	}
}

func calcDiff(actual, predict uint) (diff, absDiff float64) {
	// Due to the fact that we know only predict value, we should calculate the difference from it
	absDiff = float64(actual) - float64(predict)
	diff = absDiff * 100 / float64(predict)
	return
}

func calcBS(b *executordto.Batch) (uint, error) {
	batch := &proto.Batch{
		TxIDs:          b.Txs,
		Swaps:          b.Swaps,
		Keys:           b.Keys,
		MultiSwapsKeys: b.MultiKeys,
		MultiSwaps:     b.MultiSwaps,
	}
	return uint(pb.Size(batch)), nil
}

type blockDataWrap struct {
	chName string
	data   *collectordto.BlockData
}

func getBlocks(t *testing.T, log glog.Logger) []blockDataWrap {
	dirs := []string{"fiat", "cc"}
	var res []blockDataWrap
	for _, chName := range dirs {
		files, err := os.ReadDir(fmt.Sprintf("test-data/%s", chName))
		require.NoError(t, err)
		prsr := parser.NewParser(log, "fiat", chName, parserdto.TxPrefixes{
			Tx:        "batchTransactions",
			Swap:      "swaps",
			MultiSwap: "multi_swap",
		})

		for _, file := range files {
			b := cmn.GetBlock(t, fmt.Sprintf("test-data/%s/%s", chName, file.Name()))
			bd, err := prsr.ExtractData(b)
			if err != nil {
				continue
			}

			if bd.IsEmpty() {
				continue
			}
			res = append(res, blockDataWrap{
				chName: chName,
				data:   bd,
			})
		}
	}

	// rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(res), func(i, j int) { res[i], res[j] = res[j], res[i] })

	return res
}

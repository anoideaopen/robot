package common

import (
	"os"
	"testing"

	"github.com/anoideaopen/robot/dto/parserdto"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// DefaultPrefixes is a struct of default test prefixes
var DefaultPrefixes = parserdto.TxPrefixes{
	Tx:        "batchTransactions",
	Swap:      "swaps",
	MultiSwap: "multi_swap",
}

// GetBlock returns block from specified path
func GetBlock(t *testing.T, pathToBlock string) *common.Block {
	file, err := os.ReadFile(pathToBlock)
	require.NoError(t, err)

	fabBlock := &common.Block{}
	err = proto.Unmarshal(file, fabBlock)
	require.NoError(t, err)

	return fabBlock
}

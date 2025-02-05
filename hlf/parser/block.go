package parser

import (
	"fmt"

	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"google.golang.org/protobuf/proto"
)

// prsBlock contains all the necessary information about the blockchain block
type prsBlock struct {
	data      [][]byte
	number    uint64
	txsFilter []uint8
	isConfig  bool
}

// fromFabricBlock converts common.Block to blocklib.prsBlock.
// Such conversion is necessary for further comfortable work with information from the block.
func fromFabricBlock(block *common.Block) (*prsBlock, error) {
	metadata := &common.Metadata{}

	if err := proto.Unmarshal(block.GetMetadata().GetMetadata()[common.BlockMetadataIndex_SIGNATURES], metadata); err != nil {
		return nil, fmt.Errorf("error unmarshaling metadata from block at index [%s]: %w", common.BlockMetadataIndex_SIGNATURES, err)
	}

	envelope := &common.Envelope{}
	if err := proto.Unmarshal(block.GetData().GetData()[0], envelope); err != nil {
		return nil, fmt.Errorf("unmarshal envelope error: %w", err)
	}

	payload := &common.Payload{}
	if err := proto.Unmarshal(envelope.GetPayload(), payload); err != nil {
		return nil, fmt.Errorf("unmarshal payload error: %w", err)
	}

	hdr := &common.ChannelHeader{}
	if err := proto.Unmarshal(payload.GetHeader().GetChannelHeader(), hdr); err != nil {
		return nil, fmt.Errorf("unmarshal channel header error: %w", err)
	}

	filter := block.GetMetadata().GetMetadata()[common.BlockMetadataIndex_TRANSACTIONS_FILTER]
	if filter == nil {
		filter = block.GetMetadata().GetMetadata()[1] // for HLF 1.x compatibility
	}

	return &prsBlock{
		data:      block.GetData().GetData(),
		number:    block.GetHeader().GetNumber(),
		txsFilter: filter,
		isConfig:  common.HeaderType(hdr.GetType()) == common.HeaderType_CONFIG || common.HeaderType(hdr.GetType()) == common.HeaderType_ORDERER_TRANSACTION, //nolint:staticcheck
	}, nil
}

func (b *prsBlock) txs() []prsTx {
	validationCode := int32(peer.TxValidationCode_INVALID_OTHER_REASON)

	txs := make([]prsTx, 0, len(b.data))
	for txNumber, data := range b.data {
		if b.isConfig {
			txs = append(txs, prsTx{
				data:           data,
				validationCode: int32(peer.TxValidationCode_VALID),
			})
			continue
		}

		for _, code := range peer.TxValidationCode_value {
			if b.txsFilter[txNumber] == uint8(code) {
				validationCode = code
				break
			}
		}
		txs = append(txs, prsTx{data: data, validationCode: validationCode})
	}
	return txs
}

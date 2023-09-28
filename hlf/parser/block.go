package parser

import (
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
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

	if err := proto.Unmarshal(block.Metadata.Metadata[common.BlockMetadataIndex_SIGNATURES], metadata); err != nil {
		return nil, errors.Wrapf(err, "error unmarshaling metadata from block at index [%s]", common.BlockMetadataIndex_SIGNATURES)
	}

	envelope := &common.Envelope{}
	if err := proto.Unmarshal(block.Data.Data[0], envelope); err != nil {
		return nil, errors.Wrap(err, "unmarshal envelope error")
	}

	payload := &common.Payload{}
	if err := proto.Unmarshal(envelope.Payload, payload); err != nil {
		return nil, errors.Wrap(err, "unmarshal payload error")
	}

	hdr := &common.ChannelHeader{}
	if err := proto.Unmarshal(payload.Header.ChannelHeader, hdr); err != nil {
		return nil, errors.Wrap(err, "unmarshal channel header error")
	}

	filter := block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER]
	if filter == nil {
		filter = block.Metadata.Metadata[1] // for HLF 1.x compatibility
	}

	return &prsBlock{
		data:      block.Data.Data,
		number:    block.Header.Number,
		txsFilter: filter,
		isConfig:  common.HeaderType(hdr.Type) == common.HeaderType_CONFIG || common.HeaderType(hdr.Type) == common.HeaderType_ORDERER_TRANSACTION,
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

package parser

import (
	"encoding/hex"
	"strings"

	"github.com/anoideaopen/common-component/errorshlp"
	"github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/glog"
	"github.com/anoideaopen/robot/dto/collectordto"
	"github.com/anoideaopen/robot/dto/parserdto"
	"github.com/anoideaopen/robot/helpers/nerrors"
	"github.com/anoideaopen/robot/logger"
	protoSer "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/pkg/errors"
)

const (
	minUnicodeRuneValue = 0
	swapKeyEvent        = "key"
	multiSwapKeyEvent   = "multi_swap_key"
)

type Parser struct {
	log        glog.Logger
	txPrefixes parserdto.TxPrefixes
	dstChName  string
	srcChName  string
	isOwnCh    bool
}

func NewParser(log glog.Logger, dstChName, srcChName string, txPrefixes parserdto.TxPrefixes) *Parser {
	log = log.With(logger.Labels{
		Component: logger.ComponentParser,
		DstChName: dstChName,
		SrcChName: srcChName,
	}.Fields()...)

	return &Parser{
		log:        log,
		dstChName:  dstChName,
		srcChName:  srcChName,
		isOwnCh:    dstChName == srcChName,
		txPrefixes: txPrefixes,
	}
}

func (p *Parser) ExtractData(block *common.Block) (*collectordto.BlockData, error) {
	b, err := fromFabricBlock(block)
	if err != nil {
		return nil, errorshlp.WrapWithDetails(err, nerrors.ErrTypeParsing, nerrors.ComponentParser)
	}

	res := &collectordto.BlockData{BlockNum: b.number}

	if b.isConfig {
		return res, nil
	}

	res.Txs,
		res.Swaps, res.MultiSwaps,
		res.SwapsKeys, res.MultiSwapsKeys,
		res.Size = p.extractTxsAndSwaps(b.txs())

	return res, nil
}

func (p *Parser) extractTxsAndSwaps(txs []prsTx) ( //nolint:funlen,gocognit
	batchTxs [][]byte,
	swaps []*proto.Swap,
	multiSwaps []*proto.MultiSwap,
	swapsKeys []*proto.SwapKey,
	multiSwapsKeys []*proto.SwapKey,
	totalSize uint,
) {
	for txNum, tx := range txs {
		// If the transaction is marked as invalid, skip it
		if !tx.isValid() {
			p.log.Debugf("skip invalid transaction (number %d in block)", txNum)
			continue
		}

		chdr, err := tx.channelHeader()
		if err != nil {
			p.log.Errorf("failed to get tx channel header: %+v", err)
			continue
		}

		if common.HeaderType(chdr.Type) != common.HeaderType_ENDORSER_TRANSACTION {
			continue
		}

		actions, err := tx.getActions()
		if err != nil {
			p.log.Errorf("failed to get actions from transaction: %+v", err)
			continue
		}

		for _, action := range actions {
			rwSets, err := action.rwSets()
			if err != nil {
				p.log.Errorf("failed to get rwsets from action: %+v", err)
				continue
			}

			if p.isOwnCh {
				txID, s, ok := p.extractBatchTxFromRwSets(rwSets, chdr.TxId)
				if ok {
					batchTxs = append(batchTxs, txID)
					totalSize += s
					break
				}
				continue
			}

			ss, mss, s := p.extractSwapsAndMultiSwaps(rwSets)
			swaps = append(swaps, ss...)
			multiSwaps = append(multiSwaps, mss...)
			totalSize += s

			if action.payload.Action == nil || action.payload.Action.ProposalResponsePayload == nil {
				p.log.Debug("no payload in chaincode action payload")
				continue
			}

			sk, msk, s, err := p.extractSwapOrMultiSwapKey(action)
			if err != nil {
				p.log.Errorf("extract swap or multi swap key error: %+v", err)
				continue
			}

			if sk != nil {
				swapsKeys = append(swapsKeys, sk)
			} else if msk != nil {
				multiSwapsKeys = append(multiSwapsKeys, msk)
			}
			totalSize += s
		}
	}
	return
}

func (p *Parser) extractBatchTxFromRwSets(rwSets []prsRwSet, txid string) ([]byte, uint, bool) {
	for _, rw := range rwSets {
		for _, write := range rw.kvRWSet.Writes {
			if write.IsDelete {
				continue
			}

			ixIDFromKey := p.extractTxBatchID(write.Key)
			if ixIDFromKey == "" {
				continue
			}
			if txid != ixIDFromKey {
				p.log.Errorf("preimage from rwset is not equals to ID of transaction: tx ID %s, preimage %s", txid, ixIDFromKey)
				continue
			}

			bytesID, err := hex.DecodeString(ixIDFromKey)
			if err != nil {
				p.log.Errorf("decode tx id error: %s", err)
				return nil, 0, false
			}
			return bytesID, uint(len(bytesID)), true
		}
	}

	return nil, 0, false
}

func (p *Parser) extractSwapOrMultiSwapKey(action prsAction) (
	swapKey *proto.SwapKey,
	multiSwapKey *proto.SwapKey,
	totalSize uint,
	resErr error,
) {
	ccEvent, err := action.chaincodeEvent()
	if err != nil {
		resErr = errors.Wrap(err, "failed to get chaincode event from action")
		return
	}

	if ccEvent.EventName != swapKeyEvent && ccEvent.EventName != multiSwapKeyEvent {
		return
	}

	p.log.Infof("received key event [%s] %s", string(ccEvent.Payload), ccEvent.TxId)

	args := strings.Split(string(ccEvent.Payload), "\t")
	const countPayloadParts = 3
	if len(args) < countPayloadParts {
		resErr = errors.Errorf("incorrect key event on channel %s with payload [%s]",
			ccEvent.ChaincodeId, string(ccEvent.Payload))
		return
	}
	toChannel, swapID, keyArg := args[0], args[1], args[2]
	if p.dstChName != "" && !strings.EqualFold(toChannel, p.dstChName) {
		return
	}

	bSwapID, err := hex.DecodeString(swapID)
	if err != nil {
		resErr = errors.Wrapf(err, "incorrect id, %s", swapID)
		return
	}

	key := &proto.SwapKey{
		Id:  bSwapID,
		Key: keyArg,
	}
	totalSize = uint(len(bSwapID) + len(keyArg))

	if ccEvent.EventName == swapKeyEvent {
		swapKey = key
		return
	}

	multiSwapKey = key
	return
}

func (p *Parser) extractSwapsAndMultiSwaps(rwSets []prsRwSet) (swaps []*proto.Swap, multiSwaps []*proto.MultiSwap, totalSize uint) {
	for _, rw := range rwSets {
		for _, write := range rw.kvRWSet.Writes {
			if write.IsDelete {
				continue
			}

			if _, ok := p.hasPrefix(write.Key, p.txPrefixes.Swap); ok {
				s := &proto.Swap{}
				if err := protoSer.Unmarshal(write.Value, s); err != nil {
					p.log.Errorf("unmarshal swap from write-value error: %s", err)
					continue
				}
				if p.dstChName == "" || strings.EqualFold(s.To, p.dstChName) {
					swaps = append(swaps, s)
					totalSize += uint(len(write.Value))
				}
				continue
			}

			if _, ok := p.hasPrefix(write.Key, p.txPrefixes.MultiSwap); ok {
				ms := &proto.MultiSwap{}
				if err := protoSer.Unmarshal(write.Value, ms); err != nil {
					p.log.Errorf("unmarshal multi swap from write-value error: %s", err)
					continue
				}
				if p.dstChName == "" || strings.EqualFold(ms.To, p.dstChName) {
					multiSwaps = append(multiSwaps, ms)
					totalSize += uint(len(write.Value))
				}
				continue
			}
		}
	}
	return
}

func (p *Parser) extractTxBatchID(compositeID string) string {
	pos, ok := p.hasPrefix(compositeID, p.txPrefixes.Tx)
	if !ok {
		return ""
	}
	return compositeID[pos : len(compositeID)-1]
}

func (p *Parser) hasPrefix(compositeID, prefix string) (int, bool) {
	const countZeroRunes = 2
	if (len(compositeID) < len(prefix)+countZeroRunes) ||
		compositeID[0] != minUnicodeRuneValue ||
		compositeID[len(prefix)+1] != minUnicodeRuneValue ||
		compositeID[1:len(prefix)+1] != prefix {
		return 0, false
	}

	return len(prefix) + countZeroRunes, true
}

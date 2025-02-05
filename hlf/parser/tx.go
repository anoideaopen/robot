package parser

import (
	"fmt"

	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"google.golang.org/protobuf/proto"
)

type prsTx struct {
	data           []byte
	validationCode int32
}

// isValid checks if transaction with specified number (txNumber int) in block (block *common.Block) is valid or not and returns corresponding bool value.
func (tx *prsTx) isValid() bool {
	return tx.validationCode == int32(peer.TxValidationCode_VALID)
}

// envelope returns pointer to common.Envelope.
// common.envelope contains payload with a signature.
func (tx *prsTx) envelope() (*common.Envelope, error) {
	envelope := &common.Envelope{}
	if err := proto.Unmarshal(tx.data, envelope); err != nil {
		return nil, fmt.Errorf("unmarshal envelope error: %w", err)
	}
	return envelope, nil
}

// payload returns pointer to common.payload.
// common.payload is the message contents (and header to allow for signing).
func (tx *prsTx) payload() (*common.Payload, error) {
	envelope, err := tx.envelope()
	if err != nil {
		return nil, err
	}

	payload := &common.Payload{}
	if err = proto.Unmarshal(envelope.GetPayload(), payload); err != nil {
		return nil, fmt.Errorf("unmarshal payload error: %w", err)
	}
	return payload, nil
}

// channelHeader returns pointer to common.ChannelHeader of the transaction.
func (tx *prsTx) channelHeader() (*common.ChannelHeader, error) {
	payload, err := tx.payload()
	if err != nil {
		return nil, err
	}
	chdr := &common.ChannelHeader{}
	if err = proto.Unmarshal(payload.GetHeader().GetChannelHeader(), chdr); err != nil {
		return nil, fmt.Errorf("unmarshal channel header error: %w", err)
	}
	return chdr, nil
}

// peerTransaction returns pointer to peer.Transaction.
// The transaction to be sent to the ordering service. A transaction contains one or more TransactionAction.
// Each TransactionAction binds a proposal to potentially multiple actions.
// The transaction is atomic meaning that either all actions in the transaction will be committed or none will.
// Note that while a Transaction might include more than one Header, the Header.creator field must be the same in each.
// A single client is free to issue a number of independent Proposal, each with their header (Header) and request payload (ChaincodeProposalPayload).
// Each proposal is independently endorsed generating an action (proposalResponsePayload) with one signature per Endorser.
// Any number of independent proposals (and their action) might be included in a transaction to ensure that they are treated atomically.
func (tx *prsTx) peerTransaction() (*peer.Transaction, error) {
	payload, err := tx.payload()
	if err != nil {
		return nil, err
	}
	transaction := &peer.Transaction{}
	if err = proto.Unmarshal(payload.GetData(), transaction); err != nil {
		return nil, fmt.Errorf("unmarshal transaction error: %w", err)
	}
	return transaction, nil
}

// getActions returns slice of the prsAction structs
func (tx *prsTx) getActions() ([]prsAction, error) {
	transaction, err := tx.peerTransaction()
	if err != nil {
		return nil, err
	}
	actions := make([]prsAction, 0, len(transaction.GetActions()))
	for _, act := range transaction.GetActions() {
		ccActionPayload := &peer.ChaincodeActionPayload{}
		if err = proto.Unmarshal(act.GetPayload(), ccActionPayload); err != nil {
			return nil, fmt.Errorf("unmarshal chaincode action payload error: %w", err)
		}
		actions = append(actions, prsAction{payload: ccActionPayload})
	}
	return actions, nil
}

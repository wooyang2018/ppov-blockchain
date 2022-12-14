// Copyright (C) 2021 Aung Maw
// Licensed under the GNU General Public License v3.0

package consensus

import (
	"github.com/wooyang2018/ppov-blockchain/core"
	"github.com/wooyang2018/ppov-blockchain/emitter"
	"github.com/wooyang2018/ppov-blockchain/storage"
	"github.com/wooyang2018/ppov-blockchain/txpool"
)

type TxPool interface {
	SubmitTx(tx *core.Transaction) error
	PopTxsFromQueue(max int) []*core.Transaction
	SetTxsPending(hashes [][]byte)
	GetTxsToExecute(hashes [][]byte) ([]*core.Transaction, [][]byte)
	RemoveTxs(hashes [][]byte)
	PutTxsToQueue(hashes [][]byte)
	SyncTxs(peer *core.PublicKey, hashes [][]byte) error
	StoreTxs(txs *core.TxList) error
	GetTx(hash []byte) *core.Transaction
	GetTxStatus(hash []byte) txpool.TxStatus
	GetStatus() txpool.Status
}

type Storage interface {
	GetMerkleRoot() []byte
	Commit(data *storage.CommitData) error
	GetBlock(hash []byte) (*core.Block, error)
	GetLastBlock() (*core.Block, error)
	GetLastQC() (*core.QuorumCert, error)
	GetBlockHeight() uint64
	HasTx(hash []byte) bool
}

type MsgService interface {
	BroadcastProposal(blk *core.Block) error
	BroadcastBatch(batch *core.Batch) error
	BroadcastNewView(qc *core.QuorumCert) error
	SendVote(pubKey *core.PublicKey, vote *core.Vote) error
	SendBatchVote(pubKey *core.PublicKey, vote *core.BatchVote) error
	RequestBlock(pubKey *core.PublicKey, hash []byte) (*core.Block, error)
	RequestBlockByHeight(pubKey *core.PublicKey, height uint64) (*core.Block, error)
	SendNewView(pubKey *core.PublicKey, qc *core.QuorumCert) error
	SubscribeBatch(buffer int) *emitter.Subscription
	SubscribeProposal(buffer int) *emitter.Subscription
	SubscribeVote(buffer int) *emitter.Subscription
	SubscribeBatchVote(buffer int) *emitter.Subscription
	SubscribeNewView(buffer int) *emitter.Subscription
}

type Execution interface {
	Execute(blk *core.Block, txs []*core.Transaction) (*core.BlockCommit, []*core.TxCommit)
}

type Resources struct {
	Signer    core.Signer
	VldStore  core.ValidatorStore
	Storage   Storage
	MsgSvc    MsgService
	TxPool    TxPool
	Execution Execution
}

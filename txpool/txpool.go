// Copyright (C) 2021 Aung Maw
// Copyright (C) 2023 Wooyang2018
// Licensed under the GNU General Public License v3.0

package txpool

import (
	"bytes"
	"encoding/base64"
	"errors"

	"github.com/wooyang2018/ppov-blockchain/core"
	"github.com/wooyang2018/ppov-blockchain/emitter"
	"github.com/wooyang2018/ppov-blockchain/logger"
)

type Status struct {
	Total   int `json:"total"`   //Total = len(txStore.txItems)
	Pending int `json:"pending"` //Pending = Total-Queue
	Queue   int `json:"queue"`   //Queue = len(txStore.txq)
}

type Storage interface {
	HasTx(hash []byte) bool
}

type Execution interface {
	VerifyTx(tx *core.Transaction) error
}

type MsgService interface {
	SubscribeTxList(buffer int) *emitter.Subscription
	BroadcastTxList(txList *core.TxList) error
	RequestTxList(pubKey *core.PublicKey, hashes [][]byte) (*core.TxList, error)
}

type TxStatus uint8

const (
	TxStatusNotFound TxStatus = iota
	TxStatusQueue
	TxStatusPending
	TxStatusCommitted
)

type TxPool struct {
	storage     Storage      //存储服务
	execution   Execution    //交易执行服务
	msgSvc      MsgService   //通信服务
	store       *txStore     //交易缓存
	broadcaster *broadcaster //交易广播器
	broadcastTx bool         //是否广播交易
}

func New(storage Storage, execution Execution, msgSvc MsgService, broadcastTx bool) *TxPool {
	pool := &TxPool{
		storage:     storage,
		execution:   execution,
		msgSvc:      msgSvc,
		store:       newTxStore(),
		broadcastTx: broadcastTx,
	}
	if pool.broadcastTx {
		pool.broadcaster = newBroadcaster(msgSvc)
		go pool.broadcaster.run() //运行交易广播器
		go pool.subscribeTxs()
	}
	return pool
}

func (pool *TxPool) SubmitTx(tx *core.Transaction) error {
	return pool.submitTx(tx)
}

func (pool *TxPool) SyncTxs(peer *core.PublicKey, hashes [][]byte) error {
	return pool.syncTxs(peer, hashes)
}

func (pool *TxPool) StoreTxs(txs *core.TxList) error {
	return pool.storeTxs(txs, false)
}

func (pool *TxPool) StorePendingTxs(txs *core.TxList) error {
	return pool.storeTxs(txs, true)
}

func (pool *TxPool) PopTxsFromQueue(max int) []*core.Transaction {
	return pool.store.popTxsFromQueue(max)
}

func (pool *TxPool) GetTxsFromQueue(max int) []*core.Transaction {
	return pool.store.getTxsFromQueue(max)
}

func (pool *TxPool) PutTxsToQueue(hashes [][]byte) {
	pool.store.putTxsToQueue(hashes)
}

func (pool *TxPool) SetTxsPending(hashes [][]byte) {
	pool.store.setTxsPending(hashes)
}

func (pool *TxPool) GetTxsToExecute(hashes [][]byte) ([]*core.Transaction, [][]byte) {
	return pool.getTxsToExecute(hashes)
}

func (pool *TxPool) RemoveTxs(hashes [][]byte) {
	pool.store.removeTxs(hashes)
}

func (pool *TxPool) GetTx(hash []byte) *core.Transaction {
	return pool.store.getTx(hash)
}

func (pool *TxPool) GetTxStatus(hash []byte) TxStatus {
	return pool.getTxStatus(hash)
}

func (pool *TxPool) GetStatus() Status {
	return pool.store.getStatus()
}

func (pool *TxPool) submitTx(tx *core.Transaction) error {
	pool.PutTxsToQueue([][]byte{tx.Hash()})
	if err := pool.addNewTx(tx, false); err != nil {
		return err
	}
	if pool.broadcastTx {
		pool.broadcaster.queue <- tx
	}
	return nil
}

func (pool *TxPool) subscribeTxs() {
	sub := pool.msgSvc.SubscribeTxList(100)
	for e := range sub.Events() {
		txList := e.(*core.TxList)
		if err := pool.addTxList(txList, false); err != nil {
			logger.I().Warnf("add tx list failed %+v", err)
		}
	}
}

func (pool *TxPool) addTxList(txList *core.TxList, pending bool) error {
	jobCh := make(chan *core.Transaction, len(*txList))
	defer close(jobCh)
	out := make(chan error, len(*txList))
	defer close(out)

	for i := 0; i < 50; i++ {
		go func(jobCh <-chan *core.Transaction, out chan<- error, pending bool) {
			for tx := range jobCh {
				out <- pool.addNewTx(tx, pending)
			}
		}(jobCh, out, pending)
	}
	for _, tx := range *txList {
		jobCh <- tx
	}
	for i := 0; i < len(*txList); i++ {
		err := <-out
		if err != nil {
			return err
		}
	}
	return nil
}

func (pool *TxPool) addNewTx(tx *core.Transaction, pending bool) error {
	if err := tx.Validate(); err != nil {
		return err
	}
	if pool.storage.HasTx(tx.Hash()) {
		return nil
	}
	if err := pool.execution.VerifyTx(tx); err != nil {
		return err
	}
	pool.store.addNewTx(tx, pending)
	return nil
}

func (pool *TxPool) syncTxs(peer *core.PublicKey, hashes [][]byte) error {
	missing := make([][]byte, 0)
	for _, hash := range hashes {
		if !pool.storage.HasTx(hash) && pool.store.getTx(hash) == nil {
			missing = append(missing, hash)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	txList, err := pool.requestTxList(peer, missing)
	if err != nil {
		return err
	}
	return pool.addTxList(txList, false) //TODO 同步的交易可能是Pending状态
}

func (pool *TxPool) requestTxList(peer *core.PublicKey, hashes [][]byte) (*core.TxList, error) {
	txList, err := pool.msgSvc.RequestTxList(peer, hashes)
	if err != nil {
		return nil, err
	}
	for i, tx := range *txList {
		if !bytes.Equal(hashes[i], tx.Hash()) {
			return nil, errors.New("invalid txlist response")
		}
	}
	return txList, nil
}

func (pool *TxPool) storeTxs(txs *core.TxList, pending bool) error {
	missing := make([]*core.Transaction, 0)
	for _, tx := range *txs {
		if !pool.storage.HasTx(tx.Hash()) && pool.store.getTx(tx.Hash()) == nil {
			missing = append(missing, tx)
			if !pending && pool.broadcastTx {
				pool.broadcaster.queue <- tx
			}
		}
	}
	if len(missing) == 0 {
		return nil
	}
	if !pending {
		logger.I().Debugw("store txs into txpool", "txs", len(missing), "queue", pool.store.txq.Len())
	}
	txList := core.TxList(missing)
	return pool.addTxList(&txList, pending)
}

func (pool *TxPool) getTxsToExecute(hashes [][]byte) ([]*core.Transaction, [][]byte) {
	txs := make([]*core.Transaction, 0, len(hashes))
	executedTxs := make([][]byte, 0)
	for _, hash := range hashes {
		if pool.storage.HasTx(hash) {
			executedTxs = append(executedTxs, hash)
		} else {
			tx := pool.store.getTx(hash)
			if tx != nil {
				txs = append(txs, tx)
			} else {
				// tx not found in local node
				// all txs from accepted blocks should be sync
				logger.I().Fatalw("missing tx to execute",
					"tx", base64.StdEncoding.EncodeToString(tx.Hash()))
			}
		}
	}
	pool.store.setTxsPending(hashes)
	return txs, executedTxs
}

func (pool *TxPool) getTxStatus(hash []byte) TxStatus {
	status := pool.store.getTxStatus(hash)
	if status != TxStatusNotFound {
		return status
	}
	if pool.storage.HasTx(hash) {
		return TxStatusCommitted
	}
	return TxStatusNotFound
}

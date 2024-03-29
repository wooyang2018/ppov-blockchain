// Copyright (C) 2021 Aung Maw
// Copyright (C) 2023 Wooyang2018
// Licensed under the GNU General Public License v3.0

package txpool

import (
	"container/heap"
	"sync"
	"time"

	"github.com/wooyang2018/ppov-blockchain/core"
)

type txItem struct {
	tx           *core.Transaction
	receivedTime int64
	index        int //txItem在txQueue中的索引
}

func newTxItem(tx *core.Transaction) *txItem {
	return &txItem{
		tx:           tx,
		receivedTime: time.Now().UnixNano(),
		index:        -1,
	}
}

// inQueue 根据index判断交易是否在队列中
func (item *txItem) inQueue() bool {
	return item.index != -1
}

type txQueue []*txItem

var _ heap.Interface = (*txQueue)(nil)

func newTxQueue() *txQueue {
	txq := make(txQueue, 0)
	return &txq
}

func (txq txQueue) Len() int {
	return len(txq)
}

func (txq txQueue) Less(i, j int) bool {
	return txq[i].receivedTime < txq[j].receivedTime
}

func (txq txQueue) Swap(i, j int) {
	txq[i], txq[j] = txq[j], txq[i]
	txq[i].index = i
	txq[j].index = j
}

func (txq *txQueue) Push(x interface{}) {
	n := len(*txq)
	item := x.(*txItem)
	item.index = n
	*txq = append(*txq, item)
}

func (txq *txQueue) Pop() interface{} {
	old := *txq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	item.index = -1
	*txq = old[0 : n-1]
	return item
}

type txStore struct {
	txq     *txQueue           //交易队列/堆,receivedTime顺序，涉及排队的交易
	txItems map[string]*txItem //tx.Hash ==> txItem，涉及全部交易

	mtx sync.RWMutex
}

func newTxStore() *txStore {
	return &txStore{
		txq:     newTxQueue(),
		txItems: make(map[string]*txItem),
	}
}

func (store *txStore) addNewTx(tx *core.Transaction, pending bool) {
	store.mtx.Lock()
	defer store.mtx.Unlock()

	if store.txItems[string(tx.Hash())] != nil {
		return
	}
	item := newTxItem(tx)
	if !pending {
		heap.Push(store.txq, item)
	}
	store.txItems[string(tx.Hash())] = item
}

func (store *txStore) popTxsFromQueue(max int) []*core.Transaction {
	store.mtx.Lock()
	defer store.mtx.Unlock()

	count := min(store.txq.Len(), max)
	if count == 0 {
		return nil
	}
	ret := make([]*core.Transaction, count)
	for i := range ret {
		item := heap.Pop(store.txq).(*txItem)
		ret[i] = item.tx
	}
	return ret
}

func (store *txStore) getTxsFromQueue(max int) []*core.Transaction {
	store.mtx.Lock()
	defer store.mtx.Unlock()

	count := min(store.txq.Len(), max)
	if count == 0 {
		return nil
	}
	ret := make([]*core.Transaction, count)
	for i := range ret {
		item := (*store.txq)[i]
		ret[i] = item.tx
	}
	return ret
}

func (store *txStore) putTxsToQueue(hashes [][]byte) {
	store.mtx.Lock()
	defer store.mtx.Unlock()

	for _, hash := range hashes {
		if item, found := store.txItems[string(hash)]; found {
			if !item.inQueue() {
				heap.Push(store.txq, item)
			}
		}
	}
}

func (store *txStore) setTxsPending(hashes [][]byte) {
	store.mtx.Lock()
	defer store.mtx.Unlock()

	for _, hash := range hashes {
		if item, found := store.txItems[string(hash)]; found {
			if item.inQueue() {
				heap.Remove(store.txq, item.index)
			}
		}
	}
}

func (store *txStore) removeTxs(hashes [][]byte) {
	store.mtx.Lock()
	defer store.mtx.Unlock()

	for _, hash := range hashes {
		if item, found := store.txItems[string(hash)]; found {
			if item.inQueue() {
				heap.Remove(store.txq, item.index)
			}
			delete(store.txItems, string(hash))
		}
	}
}

func (store *txStore) getTx(hash []byte) *core.Transaction {
	store.mtx.RLock()
	defer store.mtx.RUnlock()

	item := store.txItems[string(hash)]
	if item == nil {
		return nil
	}
	return item.tx
}

func (store *txStore) getTxStatus(hash []byte) TxStatus {
	store.mtx.RLock()
	defer store.mtx.RUnlock()

	item := store.txItems[string(hash)]
	if item == nil {
		return TxStatusNotFound
	}
	if item.inQueue() {
		return TxStatusQueue
	}
	return TxStatusPending
}

func (store *txStore) getStatus() (status Status) {
	store.mtx.RLock()
	defer store.mtx.RUnlock()

	status.Total = len(store.txItems)
	status.Queue = store.txq.Len()
	status.Pending = status.Total - status.Queue
	return status
}

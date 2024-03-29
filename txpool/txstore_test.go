// Copyright (C) 2021 Aung Maw
// Copyright (C) 2023 Wooyang2018
// Licensed under the GNU General Public License v3.0

package txpool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/wooyang2018/ppov-blockchain/core"
)

func TestTxStore_addNewTx(t *testing.T) {
	assert := assert.New(t)

	tx := core.NewTransaction().Sign(core.GenerateKey(nil))
	store := newTxStore()
	store.addNewTx(tx, false)

	assert.Equal(1, store.getStatus().Total)
	assert.Equal(1, store.getStatus().Queue)
	assert.Equal(0, store.getStatus().Pending)

	txItem := store.txItems[string(tx.Hash())]

	assert.Equal(0, txItem.index)

	// add the same tx again and should not accept
	store.addNewTx(tx, false)

	assert.Nil(store.getTx([]byte("notexist")))
	assert.NotNil(store.getTx(tx.Hash()))
	assert.Equal(1, store.getStatus().Total)
	assert.Equal(1, store.getStatus().Queue)
	assert.Equal(0, store.getStatus().Pending)

	txItem1 := store.txItems[string(tx.Hash())]

	assert.Equal(txItem, txItem1)
	assert.Equal(txItem.receivedTime, txItem1.receivedTime)
}

func TestTxStore_popTxsFromQueue(t *testing.T) {
	assert := assert.New(t)

	priv := core.GenerateKey(nil)
	tx1 := core.NewTransaction().SetNonce(4).Sign(priv)
	tx2 := core.NewTransaction().SetNonce(3).Sign(priv)
	tx3 := core.NewTransaction().SetNonce(6).Sign(priv)
	tx4 := core.NewTransaction().SetNonce(2).Sign(priv)

	store := newTxStore()

	store.addNewTx(tx1, false)
	time.Sleep(1 * time.Microsecond)
	store.addNewTx(tx2, false)
	time.Sleep(1 * time.Microsecond)
	store.addNewTx(tx3, false)
	time.Sleep(1 * time.Microsecond)
	store.addNewTx(tx4, false)

	txs := store.popTxsFromQueue(2)

	assert.Equal(2, len(txs))
	assert.Equal(tx1.Hash(), txs[0].Hash())
	assert.Equal(tx2.Hash(), txs[1].Hash())

	assert.False(store.txItems[string(tx1.Hash())].inQueue())
	assert.False(store.txItems[string(tx2.Hash())].inQueue())

	assert.Equal(4, store.getStatus().Total)
	assert.Equal(2, store.getStatus().Queue)
	assert.Equal(2, store.getStatus().Pending)

	txs = store.popTxsFromQueue(3)

	assert.False(store.txItems[string(tx3.Hash())].inQueue())
	assert.False(store.txItems[string(tx4.Hash())].inQueue())

	assert.Equal(2, len(txs))
	assert.Equal(tx3.Hash(), txs[0].Hash())
	assert.Equal(tx4.Hash(), txs[1].Hash())

	assert.Equal(4, store.getStatus().Total)
	assert.Equal(0, store.getStatus().Queue)
	assert.Equal(4, store.getStatus().Pending)

	txs = store.popTxsFromQueue(2)
	assert.Nil(txs)
}

func TestTxStore_putTxsToQueue(t *testing.T) {
	assert := assert.New(t)

	priv := core.GenerateKey(nil)
	tx1 := core.NewTransaction().SetNonce(4).Sign(priv)
	tx2 := core.NewTransaction().SetNonce(3).Sign(priv)
	tx3 := core.NewTransaction().SetNonce(6).Sign(priv)
	tx4 := core.NewTransaction().SetNonce(2).Sign(priv)

	store := newTxStore()

	store.addNewTx(tx1, false)
	time.Sleep(1 * time.Microsecond)
	store.addNewTx(tx2, false)
	time.Sleep(1 * time.Microsecond)
	store.addNewTx(tx3, false)
	time.Sleep(1 * time.Microsecond)
	store.addNewTx(tx4, false)

	store.popTxsFromQueue(3)

	store.putTxsToQueue([][]byte{tx2.Hash(), tx3.Hash()})

	assert.Equal(3, store.getStatus().Queue)

	txs := store.popTxsFromQueue(2)

	assert.Equal(tx2.Hash(), txs[0].Hash())
	assert.Equal(tx3.Hash(), txs[1].Hash())

	store.putTxsToQueue([][]byte{tx1.Hash()})

	assert.Equal(2, store.getStatus().Queue)

	txs = store.popTxsFromQueue(2)

	assert.Equal(tx1.Hash(), txs[0].Hash())
	assert.Equal(tx4.Hash(), txs[1].Hash())
}

func TestTxStore_setTxsPending(t *testing.T) {
	assert := assert.New(t)

	priv := core.GenerateKey(nil)
	tx1 := core.NewTransaction().SetNonce(4).Sign(priv)
	tx2 := core.NewTransaction().SetNonce(3).Sign(priv)
	tx3 := core.NewTransaction().SetNonce(6).Sign(priv)
	tx4 := core.NewTransaction().SetNonce(2).Sign(priv)

	store := newTxStore()

	store.addNewTx(tx1, false)
	time.Sleep(1 * time.Microsecond)
	store.addNewTx(tx2, false)
	time.Sleep(1 * time.Microsecond)
	store.addNewTx(tx3, false)
	time.Sleep(1 * time.Microsecond)
	store.addNewTx(tx4, false)

	store.setTxsPending([][]byte{tx2.Hash(), tx4.Hash()})

	assert.Equal(2, store.getStatus().Pending)
	assert.Equal(2, store.getStatus().Queue)

	assert.False(store.txItems[string(tx2.Hash())].inQueue())
	assert.False(store.txItems[string(tx4.Hash())].inQueue())

	txs := store.popTxsFromQueue(3)

	assert.Equal(2, len(txs))
	assert.Equal(tx1.Hash(), txs[0].Hash())
	assert.Equal(tx3.Hash(), txs[1].Hash())
}

func TestTxStore_removeTxs(t *testing.T) {
	assert := assert.New(t)

	priv := core.GenerateKey(nil)
	tx1 := core.NewTransaction().SetNonce(4).Sign(priv)
	tx2 := core.NewTransaction().SetNonce(3).Sign(priv)
	tx3 := core.NewTransaction().SetNonce(6).Sign(priv)
	tx4 := core.NewTransaction().SetNonce(2).Sign(priv)

	store := newTxStore()

	store.addNewTx(tx1, false)
	time.Sleep(1 * time.Microsecond)
	store.addNewTx(tx2, false)
	time.Sleep(1 * time.Microsecond)
	store.addNewTx(tx3, false)
	time.Sleep(1 * time.Microsecond)
	store.addNewTx(tx4, false)

	store.popTxsFromQueue(2)

	store.removeTxs([][]byte{tx2.Hash(), tx4.Hash()})

	assert.Equal(2, store.getStatus().Total)
	assert.Equal(1, store.getStatus().Queue)
	assert.Equal(1, store.getStatus().Pending)

	txs := store.popTxsFromQueue(3)

	assert.Equal(1, len(txs))
	assert.Equal(tx3.Hash(), txs[0].Hash())
}

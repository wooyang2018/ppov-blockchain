// Copyright (C) 2021 Aung Maw
// Copyright (C) 2023 Wooyang2018
// Licensed under the GNU General Public License v3.0

package storage

import (
	"crypto"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	_ "golang.org/x/crypto/sha3"

	"github.com/wooyang2018/ppov-blockchain/core"
	"github.com/wooyang2018/ppov-blockchain/merkle"
)

const hashFunc = crypto.SHA3_256

func TestStateStore_loadPrevValuesAndTreeIndexes(t *testing.T) {
	assert := assert.New(t)

	dir, _ := os.MkdirTemp("", "db")
	rawDB, _ := NewLevelDB(dir)
	db := &levelDB{rawDB}
	ss := &stateStore{db, hashFunc, 20}

	updfns := make([]updateFunc, 3)
	updfns[0] = ss.setState([]byte{1}, []byte{100})
	updfns[1] = ss.setState([]byte{2}, []byte{200})
	updfns[2] = ss.setTreeIndex([]byte{1}, big.NewInt(9).Bytes())
	updateLevelDB(db, updfns)

	scList := []*core.StateChange{
		core.NewStateChange().SetKey([]byte{1}),
		core.NewStateChange().SetKey([]byte{2}),
	}

	ss.loadPrevTreeIndexes(scList)
	ss.loadPrevValues(scList)

	assert.Equal([]byte{100}, scList[0].PrevValue())
	assert.Equal([]byte{200}, scList[1].PrevValue())
	assert.Equal(big.NewInt(9).Bytes(), scList[0].PrevTreeIndex())
	assert.Nil(scList[1].PrevTreeIndex())
}
func TestStateStore_updateState(t *testing.T) {
	assert := assert.New(t)

	dir, _ := os.MkdirTemp("", "db")
	rawDB, _ := NewLevelDB(dir)
	db := &levelDB{rawDB}
	ss := &stateStore{db, hashFunc, 20}

	upd := core.NewStateChange().
		SetKey([]byte{1}).
		SetValue([]byte{2}).
		SetTreeIndex([]byte{1})

	assert.Nil(ss.getStateNotFoundNil(upd.Key()))

	updateLevelDB(db, ss.commitStateChange(upd))

	assert.Equal(upd.Value(), ss.getStateNotFoundNil(upd.Key()))

	idx, err := ss.getMerkleIndex(upd.Key())
	assert.NoError(err)
	assert.Equal(upd.TreeIndex(), idx)
}

func TestStateStore_computeUpdatedTreeNodes(t *testing.T) {
	assert := assert.New(t)

	scList := []*core.StateChange{
		core.NewStateChange().
			SetKey([]byte{1}).SetValue([]byte{10}).SetTreeIndex([]byte{9}),
		core.NewStateChange().
			SetKey([]byte{2}).SetValue([]byte{20}).SetTreeIndex([]byte{12}),
	}

	ss := &stateStore{
		hashFunc:        hashFunc,
		concurrentLimit: 20,
	}
	nodes := ss.computeUpdatedTreeNodes(scList)

	p0 := merkle.NewPosition(0, big.NewInt(9))
	p1 := merkle.NewPosition(0, big.NewInt(12))

	assert.Equal(p0.Bytes(), nodes[0].Position.Bytes())
	assert.Equal(p1.Bytes(), nodes[1].Position.Bytes())

	d0 := ss.sumStateValue([]byte{10})
	d1 := ss.sumStateValue([]byte{20})

	assert.Equal(d0, nodes[0].Data)
	assert.Equal(d1, nodes[1].Data)
}

func TestStateStore_setNewTreeIndexes(t *testing.T) {
	assert := assert.New(t)

	leafCount := big.NewInt(12)

	scList := []*core.StateChange{
		core.NewStateChange().
			SetKey([]byte{1}).SetValue([]byte{10}).
			SetPrevTreeIndex(big.NewInt(9).Bytes()),
		core.NewStateChange().
			SetKey([]byte{2}).SetValue([]byte{20}),
	}
	ss := &stateStore{
		hashFunc: hashFunc,
	}
	newLeafCount := ss.setNewTreeIndexes(scList, leafCount)

	assert.Equal(big.NewInt(13).Bytes(), newLeafCount.Bytes())
	assert.Equal(scList[0].PrevTreeIndex(), scList[0].TreeIndex())
	assert.Nil(scList[1].PrevTreeIndex())
	assert.Equal(big.NewInt(12).Bytes(), scList[1].TreeIndex())
}

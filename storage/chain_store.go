// Copyright (C) 2021 Aung Maw
// Copyright (C) 2023 Wooyang2018
// Licensed under the GNU General Public License v3.0

package storage

import (
	"bytes"
	"encoding/binary"

	"github.com/wooyang2018/ppov-blockchain/core"
)

type chainStore struct {
	getter getter
}

func (cs *chainStore) getLastBlock() (*core.Block, error) {
	height, err := cs.getBlockHeight()
	if err != nil {
		return nil, err
	}
	return cs.getBlockByHeight(height)
}

// getBlockHeight 获取最新区块高度
func (cs *chainStore) getBlockHeight() (uint64, error) {
	b, err := cs.getter.Get([]byte{colBlockHeight})
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(b), nil
}

func (cs *chainStore) getBlockByHeight(height uint64) (*core.Block, error) {
	hash, err := cs.getBlockHashByHeight(height)
	if err != nil {
		return nil, err
	}
	return cs.getBlock(hash)
}

// getBlockHashByHeight 根据高度获取区块Hash
func (cs *chainStore) getBlockHashByHeight(height uint64) ([]byte, error) {
	return cs.getter.Get(concatBytes([]byte{colBlockHashByHeight}, uint64BEBytes(height)))
}

// getBlock 根据Hash获取区块
func (cs *chainStore) getBlock(hash []byte) (*core.Block, error) {
	b, err := cs.getter.Get(concatBytes([]byte{colBlockByHash}, hash))
	if err != nil {
		return nil, err
	}
	blk := core.NewBlock()
	if err := blk.Unmarshal(b); err != nil {
		return nil, err
	}
	return blk, nil
}

// getLastQC 获取上一个已提交区块的QC
func (cs *chainStore) getLastQC() (*core.QuorumCert, error) {
	b, err := cs.getter.Get([]byte{colLastQC})
	if err != nil {
		return nil, err
	}
	qc := core.NewQuorumCert()
	if err := qc.Unmarshal(b); err != nil {
		return nil, err
	}
	return qc, nil
}

// getBlockCommit 通过Hash获取区块提交记录
func (cs *chainStore) getBlockCommit(hash []byte) (*core.BlockCommit, error) {
	b, err := cs.getter.Get(concatBytes([]byte{colBlockCommitByHash}, hash))
	if err != nil {
		return nil, err
	}
	bcm := core.NewBlockCommit()
	if err := bcm.Unmarshal(b); err != nil {
		return nil, err
	}
	return bcm, nil
}

// getTx 根据Hash获取交易
func (cs *chainStore) getTx(hash []byte) (*core.Transaction, error) {
	b, err := cs.getter.Get(concatBytes([]byte{colTxByHash}, hash))
	if err != nil {
		return nil, err
	}
	tx := core.NewTransaction()
	if err := tx.Unmarshal(b); err != nil {
		return nil, err
	}
	return tx, nil
}

func (cs *chainStore) hasTx(hash []byte) bool {
	return cs.getter.HasKey(concatBytes([]byte{colTxByHash}, hash))
}

// getTxCommit 根据Hash获取交易提交记录
func (cs *chainStore) getTxCommit(hash []byte) (*core.TxCommit, error) {
	val, err := cs.getter.Get(concatBytes([]byte{colTxCommitByHash}, hash))
	if err != nil {
		return nil, err
	}
	txc := core.NewTxCommit()
	if err := txc.Unmarshal(val); err != nil {
		return nil, err
	}
	return txc, nil
}

// setBlockHeight 获取设置最新Block高度的函数
func (cs *chainStore) setBlockHeight(height uint64) updateFunc {
	return func(setter setter) error {
		return setter.Set([]byte{colBlockHeight}, uint64BEBytes(height))
	}
}

func (cs *chainStore) setBlock(blk *core.Block) []updateFunc {
	ret := make([]updateFunc, 0)
	ret = append(ret, cs.setBlockByHash(blk))
	ret = append(ret, cs.setBlockHashByHeight(blk))
	return ret
}

func (cs *chainStore) setLastQC(qc *core.QuorumCert) updateFunc {
	return func(setter setter) error {
		if qc == nil { // some blocks may not have qc (hotstuff nature)
			return nil
		}
		val, err := qc.Marshal()
		if err != nil {
			return err
		}
		return setter.Set([]byte{colLastQC}, val)
	}
}

func (cs *chainStore) setBlockByHash(blk *core.Block) updateFunc {
	return func(setter setter) error {
		val, err := blk.Marshal()
		if err != nil {
			return err
		}
		return setter.Set(concatBytes([]byte{colBlockByHash}, blk.Hash()), val)
	}
}

func (cs *chainStore) setBlockHashByHeight(blk *core.Block) updateFunc {
	return func(setter setter) error {
		return setter.Set(
			concatBytes([]byte{colBlockHashByHeight}, uint64BEBytes(blk.Height())),
			blk.Hash(),
		)
	}
}

func (cs *chainStore) setBlockCommit(bcm *core.BlockCommit) updateFunc {
	return func(setter setter) error {
		val, err := bcm.Marshal()
		if err != nil {
			return err
		}
		return setter.Set(
			concatBytes([]byte{colBlockCommitByHash}, bcm.Hash()), val,
		)
	}
}

func (cs *chainStore) setTxs(txs []*core.Transaction) []updateFunc {
	ret := make([]updateFunc, len(txs))
	for i, tx := range txs {
		ret[i] = cs.setTx(tx)
	}
	return ret
}

func (cs *chainStore) setTxCommits(txCommits []*core.TxCommit) []updateFunc {
	ret := make([]updateFunc, len(txCommits))
	for i, txc := range txCommits {
		ret[i] = cs.setTxCommit(txc)
	}
	return ret
}

func (cs *chainStore) setTx(tx *core.Transaction) updateFunc {
	return func(setter setter) error {
		val, err := tx.Marshal()
		if err != nil {
			return err
		}
		return setter.Set(
			concatBytes([]byte{colTxByHash}, tx.Hash()), val,
		)
	}
}

func (cs *chainStore) setTxCommit(txc *core.TxCommit) updateFunc {
	return func(setter setter) error {
		val, err := txc.Marshal()
		if err != nil {
			return err
		}
		return setter.Set(
			concatBytes([]byte{colTxCommitByHash}, txc.Hash()), val,
		)
	}
}

func uint64BEBytes(val uint64) []byte {
	buf := bytes.NewBuffer(nil)
	binary.Write(buf, binary.BigEndian, val)
	return buf.Bytes()
}

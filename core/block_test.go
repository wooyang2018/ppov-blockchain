// Copyright (C) 2021 Aung Maw
// Copyright (C) 2023 Wooyang2018
// Licensed under the GNU General Public License v3.0

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/wooyang2018/ppov-blockchain/pb"
)

func TestBlock(t *testing.T) {
	assertt := assert.New(t)

	privKey := GenerateKey(nil)

	qc := NewQuorumCert().Build([]*Vote{
		{data: &pb.Vote{
			BlockHash: []byte{0},
			Signature: privKey.Sign([]byte{0}).data,
		}},
	})

	batchHeader := NewBatch().Header().SetTransactions([][]byte{{1}}).Sign(privKey)
	sign := privKey.Sign(batchHeader.Hash())
	batchQC := NewBatchQuorumCert().Build(batchHeader.Hash(), []*Signature{sign})
	batchHeader.SetBatchQuorumCert(batchQC)
	blk := NewBlock().
		SetHeight(4).
		SetParentHash([]byte{1}).
		SetExecHeight(0).
		SetQuorumCert(qc).
		SetMerkleRoot([]byte{1}).
		SetBatchHeaders([]*BatchHeader{batchHeader}, true).
		Sign(privKey)

	assertt.Equal(uint64(4), blk.Height())
	assertt.Equal([]byte{1}, blk.ParentHash())
	assertt.Equal(privKey.PublicKey(), blk.Proposer())
	assertt.Equal(privKey.PublicKey().Bytes(), blk.data.Proposer)
	assertt.Equal(uint64(0), blk.ExecHeight())
	assertt.Equal(qc, blk.QuorumCert())
	assertt.Equal([]byte{1}, blk.MerkleRoot())
	assertt.Equal([][]byte{{1}}, blk.Transactions())

	vs := new(MockValidatorStore)
	vs.On("VoterCount").Return(1)
	vs.On("MajorityValidatorCount").Return(1)
	vs.On("MajorityVoterCount").Return(1)
	vs.On("IsVoter", privKey.PublicKey()).Return(true)
	vs.On("IsVoter", mock.Anything).Return(false)
	vs.On("IsWorker", privKey.PublicKey()).Return(true)
	vs.On("IsWorker", mock.Anything).Return(false)

	bOk, err := blk.Marshal()
	assertt.NoError(err)

	blk.data.Signature = []byte("invalid sig")
	bInvalidSig, _ := blk.Marshal()

	privKey1 := GenerateKey(nil)
	bInvalidValidator, _ := blk.Sign(privKey1).Marshal()

	bNilQC, _ := blk.
		SetQuorumCert(NewQuorumCert()).
		Sign(privKey).
		Marshal()

	blk.data.Hash = []byte("invalid hash")
	bInvalidHash, _ := blk.Marshal()

	// test validate
	tests := []struct {
		name    string
		b       []byte
		wantErr bool
	}{
		{"valid", bOk, false},
		{"invalid sig", bInvalidSig, true},
		{"invalid validator", bInvalidValidator, true},
		{"nil qc", bNilQC, true},
		{"invalid", bInvalidHash, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			blk := NewBlock()
			err := blk.Unmarshal(tt.b)
			assert.NoError(err)

			err = blk.Validate(vs)

			if tt.wantErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}

func TestBlock_Vote(t *testing.T) {
	assert := assert.New(t)
	privKey := GenerateKey(nil)
	blk := NewBlock().Sign(privKey)
	vote := blk.Vote(privKey)
	assert.Equal(blk.Hash(), vote.BlockHash())

	vs := new(MockValidatorStore)
	vs.On("IsVoter", privKey.PublicKey()).Return(true)

	err := vote.Validate(vs)
	assert.NoError(err)
	vs.AssertExpectations(t)
}

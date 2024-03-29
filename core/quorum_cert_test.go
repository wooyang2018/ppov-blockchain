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

func TestQuorumCert(t *testing.T) {
	privKeys := make([]*PrivateKey, 5)

	vs := new(MockValidatorStore)
	vs.On("VoterCount").Return(4)
	vs.On("WorkerCount").Return(3)
	vs.On("MajorityValidatorCount").Return(3)

	for i := range privKeys {
		privKeys[i] = GenerateKey(nil)
		if i != 4 {
			vs.On("IsVoter", privKeys[i].pubKey).Return(true)
		}
		if i != 3 && i != 4 {
			vs.On("IsWorker", privKeys[i].pubKey).Return(true)
		}
	}
	vs.On("IsVoter", mock.Anything).Return(false)
	vs.On("IsWorker", mock.Anything).Return(false)

	blockHash := []byte{1}
	votes := make([]*Vote, len(privKeys))
	for i, priv := range privKeys {
		vote := NewVote()
		vote.setData(&pb.Vote{
			BlockHash: blockHash,
			Signature: priv.Sign(blockHash).data,
		})
		votes[i] = vote
	}

	nilSigVote := NewVote()
	nilSigVote.setData(&pb.Vote{
		BlockHash: blockHash,
		Signature: nil,
	})

	invalidSigVote := NewVote()
	invalidSigVote.setData(&pb.Vote{
		BlockHash: blockHash,
		Signature: privKeys[4].Sign([]byte("wrong data")).data,
	})

	qc := NewQuorumCert().Build([]*Vote{votes[2], votes[1], votes[0]})
	qcValid, err := qc.Marshal()
	assert.NoError(t, err)

	qc = NewQuorumCert().Build([]*Vote{votes[2], votes[1], votes[0], votes[3]})
	qcValidFull, _ := qc.Marshal()

	qc = NewQuorumCert().Build([]*Vote{votes[1], votes[0]})
	qcNotEnoughSig, _ := qc.Marshal()

	qc = NewQuorumCert().Build([]*Vote{votes[2], votes[3], nilSigVote, votes[0]})
	qcNilSig, _ := qc.Marshal()

	qc = NewQuorumCert().Build([]*Vote{votes[2], votes[3], votes[0], votes[2]})
	qcDuplicateKey, _ := qc.Marshal()

	qc = NewQuorumCert().Build([]*Vote{votes[1], votes[3], votes[0], votes[4]})
	qcInvalidValidator, _ := qc.Marshal()

	qc = NewQuorumCert().Build([]*Vote{votes[2], votes[1], votes[0], invalidSigVote})
	qcInvalidSig, _ := qc.Marshal()

	// test validate
	tests := []struct {
		name         string
		b            []byte
		unmarshalErr bool
		validateErr  bool
	}{
		{"valid", qcValid, false, false},
		{"valid full", qcValidFull, false, false},
		{"nil qc", nil, false, true},
		{"not enough sig", qcNotEnoughSig, false, true},
		{"nil sig", qcNilSig, true, true},
		{"duplicate key", qcDuplicateKey, false, true},
		{"invalid validator", qcInvalidValidator, false, true},
		{"invalid sig", qcInvalidSig, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			qc := NewQuorumCert()
			err := qc.Unmarshal(tt.b)
			if tt.unmarshalErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			err = qc.Validate(vs)
			if tt.validateErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}

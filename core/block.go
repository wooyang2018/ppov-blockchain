// Copyright (C) 2021 Aung Maw
// Copyright (C) 2023 Wooyang2018
// Licensed under the GNU General Public License v3.0

package core

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"

	"golang.org/x/crypto/sha3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/wooyang2018/ppov-blockchain/pb"
)

// errors
var (
	ErrInvalidBlockHash = errors.New("invalid block hash")
	ErrNilBlock         = errors.New("nil block")
)

// Block type
type Block struct {
	data       *pb.Block
	proposer   *PublicKey
	quorumCert *QuorumCert
	headers    []*BatchHeader
}

var _ json.Marshaler = (*Block)(nil)
var _ json.Unmarshaler = (*Block)(nil)

func NewBlock() *Block {
	return &Block{
		data: new(pb.Block),
	}
}

// Sum returns sha3 sum of block
func (blk *Block) Sum() []byte {
	h := sha3.New256()
	binary.Write(h, binary.BigEndian, blk.data.Height)
	h.Write(blk.data.ParentHash)
	h.Write(blk.data.Proposer)
	if blk.data.QuorumCert != nil {
		h.Write(blk.data.QuorumCert.BlockHash) // qc reference block hash
	}
	binary.Write(h, binary.BigEndian, blk.data.ExecHeight)
	h.Write(blk.data.MerkleRoot)
	binary.Write(h, binary.BigEndian, blk.data.Timestamp)
	for _, header := range blk.data.BatchHeaders {
		h.Write(header.Hash)
	}
	return h.Sum(nil)
}

// Validate block
func (blk *Block) Validate(vs ValidatorStore) error {
	if blk.data == nil {
		return ErrNilBlock
	}
	if !blk.IsGenesis() { // skip quorum cert validation for genesis block
		if err := blk.quorumCert.Validate(vs); err != nil {
			return err
		}
		for _, header := range blk.BatchHeaders() {
			if err := header.Validate(vs); err != nil {
				return err
			}
		}
	}
	if !bytes.Equal(blk.Sum(), blk.Hash()) {
		return ErrInvalidBlockHash
	}
	sig, err := newSignature(&pb.Signature{
		PubKey: blk.data.Proposer,
		Value:  blk.data.Signature,
	})
	if !vs.IsWorker(sig.PublicKey()) {
		return ErrInvalidValidator
	}
	if err != nil {
		return err
	}
	if !sig.Verify(blk.data.Hash) {
		return ErrInvalidSig
	}
	return nil
}

// Vote creates a vote for block
func (blk *Block) Vote(signer Signer) *Vote {
	vote := NewVote()
	vote.setData(&pb.Vote{
		BlockHash: blk.data.Hash,
		Signature: signer.Sign(blk.data.Hash).data,
	})
	return vote
}

func (blk *Block) ProposerVote() *Vote {
	vote := NewVote()
	vote.setData(&pb.Vote{
		BlockHash: blk.data.Hash,
		Signature: &pb.Signature{
			PubKey: blk.data.Proposer,
			Value:  blk.data.Signature,
		},
	})
	return vote
}

func (blk *Block) setData(data *pb.Block) error {
	blk.data = data
	if !blk.IsGenesis() { // every block contains qc except for genesis
		blk.quorumCert = NewQuorumCert()
		if err := blk.quorumCert.setData(data.QuorumCert); err != nil {
			return err
		}
		blk.headers = make([]*BatchHeader, len(data.BatchHeaders))
		for index := range blk.headers {
			blk.headers[index] = NewBatchHeader()
			if err := blk.headers[index].setData(data.BatchHeaders[index]); err != nil {
				return err
			}
		}
	}
	proposer, err := NewPublicKey(blk.data.Proposer)
	if err != nil {
		return err
	}
	blk.proposer = proposer
	return nil
}

func (blk *Block) SetHeight(val uint64) *Block {
	blk.data.Height = val
	return blk
}

func (blk *Block) SetParentHash(val []byte) *Block {
	blk.data.ParentHash = val
	return blk
}

func (blk *Block) SetQuorumCert(val *QuorumCert) *Block {
	blk.quorumCert = val
	blk.data.QuorumCert = val.data
	return blk
}

func (blk *Block) SetExecHeight(val uint64) *Block {
	blk.data.ExecHeight = val
	return blk
}

func (blk *Block) SetMerkleRoot(val []byte) *Block {
	blk.data.MerkleRoot = val
	return blk
}

func (blk *Block) SetTimestamp(val int64) *Block {
	blk.data.Timestamp = val
	return blk
}

func (blk *Block) SetBatchHeaders(val []*BatchHeader, isSetTx bool) *Block {
	blk.data.BatchHeaders = make([]*pb.BatchHeader, len(val))
	blk.headers = make([]*BatchHeader, len(val))
	for index := range val {
		blk.headers[index] = val[index]
		blk.data.BatchHeaders[index] = val[index].data
	}
	if isSetTx {
		txSet := make(map[string]struct{})
		txs := make([][]byte, 0)
		for _, batch := range val {
			for _, hash := range batch.Transactions() {
				if _, ok := txSet[string(hash)]; !ok {
					txSet[string(hash)] = struct{}{} //集合去重
					txs = append(txs, hash)
				}
			}
		}
		blk.data.Transactions = txs
	}
	return blk
}

func (blk *Block) SetTransactions(val [][]byte) *Block {
	blk.data.Transactions = val
	return blk
}

func (blk *Block) Sign(signer Signer) *Block {
	blk.proposer = signer.PublicKey()
	blk.data.Proposer = signer.PublicKey().key
	blk.data.Hash = blk.Sum()
	blk.data.Signature = signer.Sign(blk.data.Hash).data.Value
	return blk
}

func (blk *Block) Hash() []byte                 { return blk.data.Hash }
func (blk *Block) Height() uint64               { return blk.data.Height }
func (blk *Block) ParentHash() []byte           { return blk.data.ParentHash }
func (blk *Block) Proposer() *PublicKey         { return blk.proposer }
func (blk *Block) QuorumCert() *QuorumCert      { return blk.quorumCert }
func (blk *Block) ExecHeight() uint64           { return blk.data.ExecHeight }
func (blk *Block) MerkleRoot() []byte           { return blk.data.MerkleRoot }
func (blk *Block) Timestamp() int64             { return blk.data.Timestamp }
func (blk *Block) IsGenesis() bool              { return blk.Height() == 0 }
func (blk *Block) BatchHeaders() []*BatchHeader { return blk.headers }
func (blk *Block) Transactions() [][]byte       { return blk.data.Transactions }

// Marshal encodes blk as bytes
func (blk *Block) Marshal() ([]byte, error) {
	return proto.Marshal(blk.data)
}

// Unmarshal decodes block from bytes
func (blk *Block) Unmarshal(b []byte) error {
	data := new(pb.Block)
	if err := proto.Unmarshal(b, data); err != nil {
		return err
	}
	return blk.setData(data)
}

func (blk *Block) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(blk.data)
}

func (blk *Block) UnmarshalJSON(b []byte) error {
	data := new(pb.Block)
	if err := protojson.Unmarshal(b, data); err != nil {
		return err
	}
	return blk.setData(data)
}

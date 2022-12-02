package core

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"

	"golang.org/x/crypto/sha3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/wooyang2018/ppov-blockchain/logger"
	"github.com/wooyang2018/ppov-blockchain/pb"
)

// errors
var (
	ErrInvalidBatchHeaderHash = errors.New("invalid batch header hash")
	ErrNilBatchHeader         = errors.New("nil batch header")
)

type BatchHeader struct {
	data            *pb.BatchHeader
	proposer        *PublicKey
	batchQuorumCert *BatchQuorumCert
}

var _ json.Marshaler = (*BatchHeader)(nil)
var _ json.Unmarshaler = (*BatchHeader)(nil)

func NewBatchHeader() *BatchHeader {
	return &BatchHeader{
		data: new(pb.BatchHeader),
	}
}

// Sum returns sha3 sum of batch
func (b *BatchHeader) Sum() []byte {
	h := sha3.New256()
	h.Write(b.data.Proposer)
	binary.Write(h, binary.BigEndian, b.data.Timestamp)
	for _, txHash := range b.data.Transactions {
		h.Write(txHash)
	}
	return h.Sum(nil)
}

// Validate batch header
func (b *BatchHeader) Validate(vs ValidatorStore) error {
	if b.data == nil {
		return ErrNilBatchHeader
	}
	if b.batchQuorumCert != nil {
		if err := b.batchQuorumCert.Validate(vs); err != nil {
			return err
		}
	}
	if !bytes.Equal(b.Sum(), b.Hash()) {
		return ErrInvalidBatchHeaderHash
	}
	sig, err := newSignature(&pb.Signature{
		PubKey: b.data.Proposer,
		Value:  b.data.Signature,
	})
	if err != nil {
		return err
	}
	if !vs.IsWorker(sig.PublicKey()) {
		return ErrInvalidValidator
	}
	if !sig.Verify(b.data.Hash) {
		return ErrInvalidSig
	}
	return nil
}

func (b *BatchHeader) setData(data *pb.BatchHeader) error {
	b.data = data
	if data.BatchQuorumCert != nil && data.BatchQuorumCert.Signatures != nil {
		b.batchQuorumCert = NewBatchQuorumCert()
		if err := b.batchQuorumCert.setData(data.BatchQuorumCert); err != nil {
			return err
		}
	}
	proposer, err := NewPublicKey(b.data.Proposer)
	if err != nil {
		return err
	}
	b.proposer = proposer
	return nil
}

func (b *BatchHeader) SetBatchQuorumCert(val *BatchQuorumCert) *BatchHeader {
	if string(b.Hash()) == string(val.BatchHash()) {
		b.batchQuorumCert = val
		b.data.BatchQuorumCert = val.data
	} else {
		logger.I().Error("set batch qc failed for unmatched hash")
	}
	return b
}

func (b *BatchHeader) SetTimestamp(val int64) *BatchHeader {
	b.data.Timestamp = val
	return b
}

func (b *BatchHeader) SetTransactions(val [][]byte) *BatchHeader {
	b.data.Transactions = val
	return b
}

// Sign BatchHeader的签名函数仅供测试使用
func (b *BatchHeader) Sign(signer Signer) *BatchHeader {
	b.proposer = signer.PublicKey()
	b.data.Proposer = signer.PublicKey().key
	b.data.Hash = b.Sum()
	b.data.Signature = signer.Sign(b.data.Hash).data.Value
	return b
}

func (b *BatchHeader) Hash() []byte                      { return b.data.Hash }
func (b *BatchHeader) Proposer() *PublicKey              { return b.proposer }
func (b *BatchHeader) BatchQuorumCert() *BatchQuorumCert { return b.batchQuorumCert }
func (b *BatchHeader) Timestamp() int64                  { return b.data.Timestamp }
func (b *BatchHeader) Transactions() [][]byte            { return b.data.Transactions }

func (b *BatchHeader) Marshal() ([]byte, error) {
	return proto.Marshal(b.data)
}

func (b *BatchHeader) Unmarshal(bytes []byte) error {
	data := new(pb.BatchHeader)
	if err := proto.Unmarshal(bytes, data); err != nil {
		return err
	}
	return b.setData(data)
}

func (b *BatchHeader) UnmarshalJSON(bytes []byte) error {
	data := new(pb.BatchHeader)
	if err := protojson.Unmarshal(bytes, data); err != nil {
		return err
	}
	return b.setData(data)
}

func (b *BatchHeader) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(b.data)
}

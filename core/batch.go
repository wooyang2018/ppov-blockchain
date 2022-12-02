package core

import (
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/wooyang2018/ppov-blockchain/pb"
)

type Batch struct {
	data   *pb.Batch
	txList *TxList
	header *BatchHeader
}

var _ json.Marshaler = (*Batch)(nil)
var _ json.Unmarshaler = (*Batch)(nil)

func NewBatch() *Batch {
	b := new(Batch)
	b.data = new(pb.Batch)
	b.header = NewBatchHeader()
	b.data.Header = new(pb.BatchHeader)
	return b
}

func (b *Batch) setData(data *pb.Batch) error {
	if data.Header == nil {
		return ErrNilBatchHeader
	}
	b.data = data
	b.header = NewBatchHeader()
	if err := b.header.setData(data.Header); err != nil {
		return err
	}
	txs := make([]*Transaction, len(data.TxList), len(data.TxList))
	for i, v := range data.TxList {
		tx := NewTransaction()
		if err := tx.setData(v); err != nil {
			return err
		}
		txs[i] = tx
	}
	b.txList = (*TxList)(&txs)
	return nil
}

func (b *Batch) SetTransactions(val []*Transaction) *Batch {
	hashes := make([][]byte, len(val), len(val))
	data := make([]*pb.Transaction, len(val), len(val))
	for i, v := range val {
		hashes[i] = v.Hash()
		data[i] = v.data
	}
	b.txList = (*TxList)(&val)
	b.header.SetTransactions(hashes)
	b.data.Header.Transactions = hashes
	b.data.TxList = data
	return b
}

func (b *Batch) Sign(signer Signer) *Batch {
	b.header.proposer = signer.PublicKey()
	b.header.data.Proposer = signer.PublicKey().key
	b.data.Header.Proposer = b.header.data.Proposer
	b.header.data.Hash = b.header.Sum()
	b.data.Header.Hash = b.header.data.Hash
	b.header.data.Signature = signer.Sign(b.header.data.Hash).data.Value
	b.data.Header.Signature = b.header.data.Signature
	return b
}

func (b *Batch) SetTimestamp(val int64) *Batch {
	b.data.Header.Timestamp = val
	b.header.SetTimestamp(val)
	return b
}

func (b *Batch) Header() *BatchHeader { return b.header }
func (b *Batch) TxList() *TxList      { return b.txList }

func (b *Batch) Marshal() ([]byte, error) {
	return proto.Marshal(b.data)
}

func (b *Batch) Unmarshal(bytes []byte) error {
	data := new(pb.Batch)
	if err := proto.Unmarshal(bytes, data); err != nil {
		return err
	}
	return b.setData(data)
}

func (b *Batch) UnmarshalJSON(bytes []byte) error {
	data := new(pb.Batch)
	if err := protojson.Unmarshal(bytes, data); err != nil {
		return err
	}
	return b.setData(data)
}

func (b *Batch) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(b.data)
}

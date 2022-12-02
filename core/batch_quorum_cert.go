package core

import (
	"errors"

	"google.golang.org/protobuf/proto"

	"github.com/wooyang2018/ppov-blockchain/pb"
)

// errors
var (
	ErrNilBatchQC        = errors.New("nil batch qc")
	ErrNotEnoughBatchSig = errors.New("not enough batch signatures in qc")
	ErrDuplicateBatchSig = errors.New("duplicate batch signature in qc")
	ErrInvalidBatchSig   = errors.New("invalid batch signature")
	ErrInvalidBatchVoter = errors.New("not a invalid batch voter")
)

// BatchQuorumCert type
type BatchQuorumCert struct {
	data *pb.BatchQuorumCert
	sigs sigList
}

func NewBatchQuorumCert() *BatchQuorumCert {
	return &BatchQuorumCert{
		data: new(pb.BatchQuorumCert),
	}
}

func (qc *BatchQuorumCert) Validate(vs ValidatorStore) error {
	if qc.data == nil {
		return ErrNilBatchQC
	}
	if len(qc.sigs) < vs.MajorityVoterCount() {
		return ErrNotEnoughBatchSig
	}
	if qc.sigs.hasDuplicate() {
		return ErrDuplicateBatchSig
	}
	if qc.sigs.hasInvalidVoter(vs) {
		return ErrInvalidBatchVoter
	}
	if qc.sigs.hasInvalidSig(qc.data.BatchHash) {
		return ErrInvalidBatchSig
	}
	return nil
}

func (qc *BatchQuorumCert) setData(data *pb.BatchQuorumCert) error {
	if data == nil {
		return ErrNilBatchQC
	}
	qc.data = data
	sigs, err := newSigList(qc.data.Signatures)
	if err != nil {
		return err
	}
	qc.sigs = sigs
	return nil
}

func (qc *BatchQuorumCert) Build(hash []byte, signs []*Signature) *BatchQuorumCert {
	qc.data.Signatures = make([]*pb.Signature, len(signs))
	qc.sigs = make(sigList, len(signs))
	qc.data.BatchHash = hash
	for i, sign := range signs {
		qc.data.Signatures[i] = sign.data
		qc.sigs[i] = sign
	}
	return qc
}

func (qc *BatchQuorumCert) BatchHash() []byte        { return qc.data.BatchHash }
func (qc *BatchQuorumCert) Signatures() []*Signature { return qc.sigs }

// Marshal encodes quorum cert as bytes
func (qc *BatchQuorumCert) Marshal() ([]byte, error) {
	return proto.Marshal(qc.data)
}

// Unmarshal decodes quorum cert from bytes
func (qc *BatchQuorumCert) Unmarshal(b []byte) error {
	data := new(pb.BatchQuorumCert)
	if err := proto.Unmarshal(b, data); err != nil {
		return err
	}
	return qc.setData(data)
}

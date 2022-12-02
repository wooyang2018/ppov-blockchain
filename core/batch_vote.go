package core

import (
	"errors"

	"google.golang.org/protobuf/proto"

	"github.com/wooyang2018/ppov-blockchain/pb"
)

// errors
var (
	ErrNilBatchVote    = errors.New("nil batch vote")
	ErrDifferentSigner = errors.New("batch vote with different signers")
)

// BatchVote type
type BatchVote struct {
	data    *pb.BatchVote
	voter   *PublicKey
	headers []*BatchHeader
	sigs    sigList
}

func NewBatchVote() *BatchVote {
	return &BatchVote{
		data: new(pb.BatchVote),
	}
}

// Validate batch vote
func (vote *BatchVote) Validate(vs ValidatorStore) error {
	if vote.data == nil {
		return ErrNilBatchVote
	}
	for i, s := range vote.data.Signatures {
		sig, err := newSignature(s)
		if err != nil {
			return err
		}
		if !vs.IsVoter(sig.PublicKey()) {
			return ErrInvalidBatchVoter
		}
		if !sig.Verify(vote.data.BatchHeaders[i].Hash) {
			return ErrInvalidSig
		}
	}
	return nil
}

func (vote *BatchVote) setData(data *pb.BatchVote) error {
	if data == nil {
		return ErrNilBatchVote
	}
	vote.data = data
	length := len(vote.data.Signatures)
	vote.sigs = make(sigList, 0, length)
	vote.headers = make([]*BatchHeader, 0, length)
	for i := 0; i < length; i++ {
		sig, err := newSignature(vote.data.Signatures[i])
		if err != nil {
			return err
		}
		vote.sigs = append(vote.sigs, sig)

		header := NewBatchHeader()
		if err := header.setData(vote.data.BatchHeaders[i]); err != nil {
			return err
		}
		vote.headers = append(vote.headers, header)

		if vote.voter == nil {
			vote.voter = sig.pubKey
		} else {
			if vote.voter.String() != sig.PublicKey().String() {
				return ErrDifferentSigner
			}
		}

	}
	return nil
}

// Build generate a vote for a bunch of batches
func (vote *BatchVote) Build(headers []*BatchHeader, signer Signer) *BatchVote {
	data := new(pb.BatchVote)
	length := len(headers)
	data.BatchHeaders = make([]*pb.BatchHeader, 0, length)
	data.Signatures = make([]*pb.Signature, 0, length)
	for i := 0; i < length; i++ {
		data.BatchHeaders = append(data.BatchHeaders, headers[i].data)
		data.Signatures = append(data.Signatures, signer.Sign(headers[i].data.Hash).data)
	}
	vote.setData(data)
	return vote
}

func (vote *BatchVote) BatchHeaders() []*BatchHeader { return vote.headers }
func (vote *BatchVote) Voter() *PublicKey            { return vote.voter }
func (vote *BatchVote) Signatures() []*Signature     { return vote.sigs }

// Marshal encodes batch vote as bytes
func (vote *BatchVote) Marshal() ([]byte, error) {
	return proto.Marshal(vote.data)
}

// Unmarshal decodes batch vote from bytes
func (vote *BatchVote) Unmarshal(b []byte) error {
	data := new(pb.BatchVote)
	if err := proto.Unmarshal(b, data); err != nil {
		return err
	}
	return vote.setData(data)
}

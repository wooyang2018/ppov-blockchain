// Copyright (C) 2021 Aung Maw
// Copyright (C) 2023 Wooyang2018
// Licensed under the GNU General Public License v3.0

// package hotstuff implements hotstuff core algorithm from https://arxiv.org/abs/1803.05069

package hotstuff

import (
	"os"
	"time"

	"github.com/wooyang2018/ppov-blockchain/logger"
)

// Hotstuff consensus engine
type Hotstuff struct {
	*state
	driver Driver
	tester *tester
}

func New(driver Driver, file *os.File, b0 Block, q0 QC) *Hotstuff {
	return &Hotstuff{
		state:  newState(b0, q0),
		driver: driver,
		tester: newTester(file),
	}
}

// OnPropose is called to propose a new block
func (hs *Hotstuff) OnPropose() Block {
	bLeaf := hs.GetBLeaf()
	bNew := hs.driver.CreateLeaf(bLeaf, hs.GetQCHigh(), bLeaf.Height()+1)
	if bNew == nil {
		return nil
	}
	hs.setBLeaf(bNew)
	hs.startProposal(bNew)
	hs.driver.BroadcastProposal(bNew)
	return bNew
}

// OnReceiveVote is called when received a vote
func (hs *Hotstuff) OnReceiveVote(v Vote) {
	err := hs.addVote(v)
	if err != nil {
		return
	}
	logger.I().Debugw("hotstuff received vote", "height", v.Block().Height())
	if hs.GetVoteCount() >= hs.driver.MajorityValidatorCount() {
		votes := hs.GetVotes()
		hs.endProposal()
		hs.UpdateQCHigh(hs.driver.CreateQC(votes))
	}
}

// OnReceiveProposal is called when a new proposal is received
func (hs *Hotstuff) OnReceiveProposal(bNew Block) {
	if hs.CanVote(bNew) {
		hs.driver.VoteBlock(bNew)
		hs.setBVote(bNew)
	}
	hs.Update(bNew)
}

// CanVote returns true if the hotstuff instance can vote the given block
func (hs *Hotstuff) CanVote(bNew Block) bool {
	if CmpBlockHeight(bNew, hs.GetBVote()) == 1 {
		return hs.CheckSafetyRule(bNew) || hs.CheckLivenessRule(bNew)
	}
	return false
}

// CheckSafetyRule returns true if the given block extends from b_Lock
func (hs *Hotstuff) CheckSafetyRule(bNew Block) bool {
	bLock := hs.GetBLock()
	for b := bNew; CmpBlockHeight(b, bLock) != -1; b = b.Parent() {
		if bLock.Equal(b) {
			return true
		}
	}
	return false
}

// CheckLivenessRule returns true if the qc referenced block of the given block is higher than b_Lock
func (hs *Hotstuff) CheckLivenessRule(bNew Block) bool {
	return CmpBlockHeight(bNew.Justify().Block(), hs.GetBLock()) == 1
}

// Update perform two/three chain consensus phases
func (hs *Hotstuff) Update(bNew Block) {
	if TwoPhaseFlag {
		_, b, b1 := GetJustifyBlocks(bNew)
		hs.UpdateQCHigh(bNew.Justify()) // prepare phase for b1
		if CmpBlockHeight(b1, hs.GetBLock()) == 1 {
			hs.setBLock(b1)        // commit phase for b1
			if IsTwoChain(b, b1) { // decide phase for b
				t1 := time.Now().UnixNano()
				hs.onCommit(b)
				hs.setBExec(b)
				t2 := time.Now().UnixNano()
				hs.tester.saveItem(b.Height(), b.Timestamp(), t1, t2, len(b.Transactions()))
			}
		}
	} else {
		b, b1, b2 := GetJustifyBlocks(bNew)
		hs.UpdateQCHigh(bNew.Justify()) // prepare phase for b2
		if CmpBlockHeight(b1, hs.GetBLock()) == 1 {
			hs.setBLock(b1)              // commit phase for b1
			if IsThreeChain(b, b1, b2) { // decide phase for b
				t1 := time.Now().UnixNano()
				hs.onCommit(b)
				hs.setBExec(b)
				t2 := time.Now().UnixNano()
				hs.tester.saveItem(b.Height(), b.Timestamp(), t1, t2, len(b.Transactions()))
			}
		}
	}
}

func (hs *Hotstuff) onCommit(b Block) {
	if CmpBlockHeight(b, hs.GetBExec()) == 1 {
		// commit parent blocks recursively
		hs.onCommit(b.Parent())
		hs.driver.Commit(b)
	} else if !hs.GetBExec().Equal(b) {
		logger.I().Warnf("hotstuff safety breached b-recurrsive: %+v\n bexec: %d",
			b, hs.GetBExec().Height())
	}
}

// UpdateQCHigh replaces qcHigh if the block of given qc is higher than the qcHigh block
func (hs *Hotstuff) UpdateQCHigh(qc QC) {
	if CmpBlockHeight(qc.Block(), hs.GetQCHigh().Block()) == 1 {
		logger.I().Debugw("hotstuff updated high qc", "height", qc.Block().Height())
		hs.setQCHigh(qc)
		hs.setBLeaf(qc.Block())
		hs.qcHighEmitter.Emit(qc)
	}
}

// GetJustifyBlocks returns justify referenced blocks
func GetJustifyBlocks(bNew Block) (b, b1, b2 Block) {
	if b2 = bNew.Justify().Block(); b2 == nil {
		return b, b1, b2
	}
	if b1 = b2.Justify().Block(); b1 == nil {
		return b, b1, b2
	}
	b = b1.Justify().Block()
	return b, b1, b2
}

func IsTwoChain(b1, b2 Block) bool {
	if b1 == nil || b2 == nil {
		return false
	}
	return b1.Equal(b2.Parent())
}

// IsThreeChain checks whether the blocks satisfy three chain rule: b<--b1<--p2
func IsThreeChain(b, b1, b2 Block) bool {
	if b == nil || b1 == nil || b2 == nil {
		return false
	}
	return b1.Equal(b2.Parent()) && b.Equal(b1.Parent())
}

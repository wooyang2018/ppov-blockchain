// Copyright (C) 2021 Aung Maw
// Licensed under the GNU General Public License v3.0

package consensus

import (
	"time"

	"github.com/wooyang2018/ppov-blockchain/core"
	"github.com/wooyang2018/ppov-blockchain/hotstuff"
	"github.com/wooyang2018/ppov-blockchain/logger"
)

type Consensus struct {
	resources *Resources
	config    Config

	startTime int64

	state     *state
	hsDriver  *hsDriver
	hotstuff  *hotstuff.Hotstuff
	validator *validator
	pacemaker *pacemaker
	rotator   *rotator

	voterState  *voterState
	leaderState *leaderState
}

func New(resources *Resources, config Config) *Consensus {
	cons := &Consensus{
		resources: resources,
		config:    config,
	}
	return cons
}

func (cons *Consensus) Start() {
	cons.start()
}

func (cons *Consensus) Stop() {
	cons.stop()
}

func (cons *Consensus) GetStatus() Status {
	return cons.getStatus()
}

func (cons *Consensus) GetBlock(hash []byte) *core.Block {
	return cons.state.getBlock(hash)
}

func (cons *Consensus) start() {
	cons.startTime = time.Now().UnixNano()
	b0, q0 := cons.getInitialBlockAndQC()
	cons.setupState(b0)
	cons.setupPPovState()
	cons.setupHsDriver()
	cons.setupHotstuff(b0, q0)
	cons.setupValidator()
	cons.setupPacemaker()
	cons.setupRotator()

	cons.validator.start()
	cons.pacemaker.start()
	cons.rotator.start()
}

func (cons *Consensus) stop() {
	if cons.pacemaker == nil {
		return
	}
	cons.pacemaker.stop()
	cons.rotator.stop()
	cons.validator.stop()
}

func (cons *Consensus) setupState(b0 *core.Block) {
	cons.state = newState(cons.resources)
	cons.state.setBlock(b0)
	cons.state.setLeaderIndex(cons.resources.VldStore.GetWorkerIndex(b0.Proposer()))
}

func (cons *Consensus) getInitialBlockAndQC() (*core.Block, *core.QuorumCert) {
	b0, err := cons.resources.Storage.GetLastBlock()
	if err == nil {
		q0, err := cons.resources.Storage.GetLastQC()
		if err != nil {
			logger.I().Fatalf("cannot get last qc %d", b0.Height())
		}
		return b0, q0
	}
	// chain not started, create genesis block
	genesis := &genesis{
		resources: cons.resources,
		chainID:   cons.config.ChainID,
	}
	return genesis.run()
}

func (cons *Consensus) setupHsDriver() {
	cons.hsDriver = &hsDriver{
		resources:    cons.resources,
		config:       cons.config,
		checkTxDelay: 10 * time.Millisecond,
		state:        cons.state,
		leaderState:  cons.leaderState,
	}
}

func (cons *Consensus) setupHotstuff(b0 *core.Block, q0 *core.QuorumCert) {
	cons.hotstuff = hotstuff.New(
		cons.hsDriver,
		newHsBlock(b0, cons.state),
		newHsQC(q0, cons.state),
	)
}

func (cons *Consensus) setupPPovState() {
	cons.voterState = newVoterState()
	cons.voterState.setVoteBatchLimit(cons.config.VoteBatchLimit)

	cons.leaderState = newLeaderState()
	cons.leaderState.setBatchSignLimit(cons.resources.VldStore.MajorityVoterCount())
	cons.leaderState.setBatchWaitTime(cons.config.BatchWaitTime)
	cons.leaderState.setBlockBatchLimit(cons.config.BlockBatchLimit)
}

func (cons *Consensus) setupValidator() {
	cons.validator = &validator{
		resources:   cons.resources,
		state:       cons.state,
		leaderState: cons.leaderState,
		voterState:  cons.voterState,
		hotstuff:    cons.hotstuff,
	}
}

func (cons *Consensus) setupPacemaker() {
	cons.pacemaker = &pacemaker{
		resources:  cons.resources,
		config:     cons.config,
		state:      cons.state,
		voterState: cons.voterState,
		hotstuff:   cons.hotstuff,
	}
}

func (cons *Consensus) setupRotator() {
	cons.rotator = &rotator{
		resources: cons.resources,
		config:    cons.config,
		state:     cons.state,
		hotstuff:  cons.hotstuff,
	}
}

func (cons *Consensus) getStatus() (status Status) {
	if cons.pacemaker == nil {
		return status
	}
	status.StartTime = cons.startTime
	status.CommittedTxCount = cons.state.getCommittedTxCount()
	status.BlockPoolSize = cons.state.getBlockPoolSize()
	status.QCPoolSize = cons.state.getQCPoolSize()
	status.LeaderIndex = cons.state.getLeaderIndex()
	status.ViewStart = cons.rotator.getViewStart()
	status.PendingViewChange = cons.rotator.getPendingViewChange()

	status.BVote = cons.hotstuff.GetBVote().Height()
	status.BLeaf = cons.hotstuff.GetBLeaf().Height()
	status.BLock = cons.hotstuff.GetBLock().Height()
	status.BExec = cons.hotstuff.GetBExec().Height()
	status.QCHigh = qcRefHeight(cons.hotstuff.GetQCHigh())
	return status
}

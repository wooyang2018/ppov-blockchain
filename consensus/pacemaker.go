// Copyright (C) 2021 Aung Maw
// Licensed under the GNU General Public License v3.0

package consensus

import (
	"time"

	"github.com/wooyang2018/ppov-blockchain/core"
	"github.com/wooyang2018/ppov-blockchain/hotstuff"
	"github.com/wooyang2018/ppov-blockchain/logger"
)

type pacemaker struct {
	resources *Resources
	config    Config

	state      *state
	voterState *voterState

	hotstuff *hotstuff.Hotstuff

	stopCh chan struct{}
}

func (pm *pacemaker) start() {
	if pm.stopCh != nil {
		return
	}
	pm.stopCh = make(chan struct{})
	go pm.batchRun()
	go pm.run()
	logger.I().Info("started pacemaker")
}

func (pm *pacemaker) stop() {
	if pm.stopCh == nil {
		return // not started yet
	}
	select {
	case <-pm.stopCh: // already stopped
		return
	default:
	}
	close(pm.stopCh)
	logger.I().Info("stopped pacemaker")
	pm.stopCh = nil
}

func (pm *pacemaker) batchRun() {
	subQC := pm.hotstuff.SubscribeNewQCHigh()
	defer subQC.Unsubscribe()

	for {
		pm.newBatch()
		bacthT := pm.nextBatchTimeout()

		select {
		case <-pm.stopCh:
			return

		case <-bacthT.C:
		case <-subQC.Events():
		}
		bacthT.Stop()
	}
}

func (pm *pacemaker) run() {
	subQC := pm.hotstuff.SubscribeNewQCHigh()
	defer subQC.Unsubscribe()

	for {
		blkDelay := time.After(pm.config.BlockDelay)
		pm.newBlock()
		beatT := pm.nextProposeTimeout()

		select {
		case <-pm.stopCh:
			return

		// either beatdelay timeout or I'm able to create qc
		case <-beatT.C:
		case <-subQC.Events():
		}
		beatT.Stop()

		select {
		case <-pm.stopCh:
			return
		case <-blkDelay:
		}
	}
}

func (pm *pacemaker) nextProposeTimeout() *time.Timer {
	proposeWait := pm.config.ProposeTimeout
	return time.NewTimer(proposeWait)
}

func (pm *pacemaker) nextBatchTimeout() *time.Timer {
	batchWait := pm.config.BatchTimeout
	if pm.resources.TxPool.GetStatus().Total == 0 {
		batchWait += pm.config.TxWaitTime
	}
	return time.NewTimer(batchWait)
}

func (pm *pacemaker) newBlock() {
	pm.state.mtxUpdate.Lock()
	defer pm.state.mtxUpdate.Unlock()

	select {
	case <-pm.stopCh:
		return
	default:
	}

	if !pm.state.isThisNodeLeader() {
		return
	}

	blk := pm.hotstuff.OnPropose()
	logger.I().Debugw("proposed block", "height", blk.Height(), "qc", qcRefHeight(blk.Justify()))
	vote := blk.(*hsBlock).block.ProposerVote()
	pm.hotstuff.OnReceiveVote(newHsVote(vote, pm.state))
	pm.hotstuff.Update(blk)
}

func (pm *pacemaker) newBatch() {
	pm.state.mtxUpdate.Lock()
	defer pm.state.mtxUpdate.Unlock()

	select {
	case <-pm.stopCh:
		return
	default:
	}

	if !pm.state.isThisNodeWorker() {
		return
	}

	signer := pm.resources.Signer
	batch := core.NewBatch().
		SetTransactions(pm.resources.TxPool.PopTxsFromQueue(pm.config.BatchTxLimit)).
		SetTimestamp(time.Now().UnixNano()).
		Sign(signer)
	pm.voterState.addBatch(batch)
	pm.resources.MsgSvc.BroadcastBatch(batch)

	widx := pm.resources.VldStore.GetWorkerIndex(signer.PublicKey())
	logger.I().Debugw("generated batch", "txs", len(batch.Transactions()), "worker", widx)
}

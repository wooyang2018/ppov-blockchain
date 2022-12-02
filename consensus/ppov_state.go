package consensus

import (
	"sync"
	"time"

	"github.com/wooyang2018/ppov-blockchain/core"
)

type voterState struct {
	batchQ         []*core.BatchHeader //待投票的Batch队列
	voteBatchLimit int
	mtxState       sync.RWMutex
}

func newVoterState() *voterState {
	return &voterState{
		batchQ: make([]*core.BatchHeader, 0),
	}
}

func (v *voterState) setVoteBatchLimit(voteBatchLimit int) *voterState {
	v.mtxState.Lock()
	defer v.mtxState.Unlock()
	v.voteBatchLimit = voteBatchLimit
	return v
}

func (v *voterState) addBatch(batch *core.BatchHeader) {
	v.mtxState.Lock()
	defer v.mtxState.Unlock()
	v.batchQ = append(v.batchQ, batch)
}

func (v *voterState) hasEnoughBatch() bool {
	v.mtxState.RLock()
	defer v.mtxState.RUnlock()
	return len(v.batchQ) >= v.voteBatchLimit
}

// popBatch 从队列头部弹出num个Batch
func (v *voterState) popBatch() []*core.BatchHeader {
	v.mtxState.Lock()
	defer v.mtxState.Unlock()

	num := v.voteBatchLimit
	res := make([]*core.BatchHeader, 0, num)
	for _, batch := range v.batchQ {
		if num <= 0 {
			break
		}
		num--
		res = append(res, batch)
	}
	v.batchQ = v.batchQ[len(res):]
	return res
}

type leaderState struct {
	batchMap    map[string]*core.BatchHeader //batch hash -> batch
	batchSigns  map[string][]*core.Signature //batch hash -> signature list
	batchStopCh map[string]chan struct{}

	batchReadyQ []*core.BatchHeader //就绪Batch队列

	batchWaitTime   time.Duration //Batch超时时间
	blockBatchLimit int
	batchSignLimit  int

	mtxState sync.RWMutex //TODO 锁粒度优化
}

func newLeaderState() *leaderState {
	return &leaderState{
		batchMap:    make(map[string]*core.BatchHeader),
		batchSigns:  make(map[string][]*core.Signature),
		batchStopCh: make(map[string]chan struct{}),
		batchReadyQ: make([]*core.BatchHeader, 0),
	}
}

func (l *leaderState) setBatchSignLimit(batchSignLimit int) *leaderState {
	l.mtxState.Lock()
	defer l.mtxState.Unlock()
	l.batchSignLimit = batchSignLimit
	return l
}

func (l *leaderState) setBlockBatchLimit(blockBatchLimit int) *leaderState {
	l.mtxState.Lock()
	defer l.mtxState.Unlock()
	l.blockBatchLimit = blockBatchLimit
	return l
}

func (l *leaderState) setBatchWaitTime(batchWaitTime time.Duration) *leaderState {
	l.mtxState.Lock()
	defer l.mtxState.Unlock()
	l.batchWaitTime = batchWaitTime
	return l
}

func (l *leaderState) addBatchVote(vote *core.BatchVote) {
	l.mtxState.Lock()
	defer l.mtxState.Unlock()
	for index, sig := range vote.Signatures() {
		batch := vote.BatchHeaders()[index]
		hash := string(batch.Hash())
		//如果第一次收到对该Batch的投票
		if _, ok := l.batchMap[hash]; !ok {
			l.batchMap[hash] = batch
			l.batchSigns[hash] = make([]*core.Signature, 0)
			l.batchStopCh[hash] = make(chan struct{})
			go l.waitCleanBatch(hash, time.NewTimer(l.batchWaitTime), l.batchStopCh[hash])
		}
		l.batchSigns[hash] = append(l.batchSigns[hash], sig)
		if len(l.batchSigns[hash]) >= l.batchSignLimit {
			batchQC := core.NewBatchQuorumCert().Build(batch.Hash(), l.batchSigns[hash])
			batch.SetBatchQuorumCert(batchQC)
			l.batchReadyQ = append(l.batchReadyQ, batch)
			close(l.batchStopCh[hash])
			delete(l.batchMap, hash)
			delete(l.batchSigns, hash)
			delete(l.batchStopCh, hash)
		}
	}
}

// popReadyBatch 从就绪队列的头部弹出num个Batch
func (l *leaderState) popReadyBatch() []*core.BatchHeader {
	l.mtxState.Lock()
	defer l.mtxState.Unlock()

	num := l.blockBatchLimit
	res := make([]*core.BatchHeader, 0, num)
	for _, batch := range l.batchReadyQ {
		if num <= 0 { //TODO 改写法
			break
		}
		num--
		res = append(res, batch)
	}
	l.batchReadyQ = l.batchReadyQ[len(res):]
	return res
}

func (l *leaderState) waitCleanBatch(hash string, timer *time.Timer, stopCh chan struct{}) {
	select {
	case <-timer.C:
		l.mtxState.Lock()
		delete(l.batchMap, hash)
		delete(l.batchSigns, hash)
		delete(l.batchStopCh, hash)
		l.mtxState.Unlock()
	case <-stopCh:
	}
	timer.Stop()
}

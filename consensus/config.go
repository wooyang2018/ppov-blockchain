// Copyright (C) 2021 Aung Maw
// Licensed under the GNU General Public License v3.0

package consensus

import "time"

type Config struct {
	ChainID int64

	// maximum tx count in a batch
	BatchTxLimit int

	// maximum batch count in a block
	BlockBatchLimit int

	// maximum batch count in batch vote
	VoteBatchLimit int

	// block creation delay if no transactions in the pool
	TxWaitTime time.Duration

	// Leader等待对某个Batch的投票的最长时间
	BatchWaitTime time.Duration

	// for leader, delay to propose next block if she cannot create qc
	ProposeTimeout time.Duration

	BatchTimeout time.Duration

	// minimum delay between each block (i.e, it can define maximum block rate)
	BlockDelay time.Duration

	// view duration for a leader
	ViewWidth time.Duration

	// leader must create next qc within this duration
	LeaderTimeout time.Duration
}

var DefaultConfig = Config{
	BatchTxLimit:    200,
	BlockBatchLimit: 4,
	VoteBatchLimit:  4,
	TxWaitTime:      1 * time.Second,
	BatchWaitTime:   3 * time.Second,
	ProposeTimeout:  500 * time.Millisecond,
	BatchTimeout:    400 * time.Millisecond,
	BlockDelay:      40 * time.Millisecond, // maximum block rate = 25 blk per sec
	ViewWidth:       90 * time.Second,
	LeaderTimeout:   30 * time.Second,
}

// Copyright (C) 2021 Aung Maw
// Copyright (C) 2023 Wooyang2018
// Licensed under the GNU General Public License v3.0

package consensus

import "time"

const ExecuteTxFlag = true   //set to false when benchmark test
const PreserveTxFlag = false //set to true when benchmark test
const VoteBatchFlag = true   //set to false to prevent voting on batch

type Config struct {
	ChainID int64

	// maximum tx count in a batch
	BatchTxLimit int

	// maximum batch count in a block
	BlockBatchLimit int

	//batch count in a batch vote
	VoteBatchLimit int

	// block creation delay if no transactions in the pool
	TxWaitTime time.Duration

	// maximum delay the leader waits for voting on a batch
	BatchWaitTime time.Duration

	// duration to wait to propose next block if leader cannot create qc
	ProposeTimeout time.Duration

	// duration to wait to propose next batch if leader cannot create qc
	BatchTimeout time.Duration

	// minimum delay between each block (i.e, it can define maximum block rate)
	BlockDelay time.Duration

	// view duration for a leader
	ViewWidth time.Duration

	// leader must create next qc within this duration
	LeaderTimeout time.Duration

	// path to save the benchmark log of the consensus algorithm (it will not be saved if blank)
	BenchmarkPath string
}

var DefaultConfig = Config{
	BatchTxLimit:    5000,
	BlockBatchLimit: -1, // set to -1 to adapt to the number of worker nodes
	VoteBatchLimit:  -1, // set to -1 to adapt to the number of worker nodes
	TxWaitTime:      1 * time.Second,
	BatchWaitTime:   3 * time.Second,
	ProposeTimeout:  1500 * time.Millisecond,
	BatchTimeout:    1500 * time.Millisecond,
	BlockDelay:      500 * time.Millisecond,
	ViewWidth:       60 * time.Second,
	LeaderTimeout:   20 * time.Second,
	BenchmarkPath:   "",
}

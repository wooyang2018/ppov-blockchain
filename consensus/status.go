// Copyright (C) 2021 Aung Maw
// Copyright (C) 2023 Wooyang2018
// Licensed under the GNU General Public License v3.0

package consensus

type Status struct {
	StartTime int64

	// committed tx count since node is up
	CommittedTxCount int
	BlockPoolSize    int
	QCPoolSize       int

	// start timestamp of current view
	ViewStart int64

	// set to true when current view timeout
	// set to false once the view leader created the first qc
	PendingViewChange bool
	LeaderIndex       int

	// hotstuff state (block heights)
	BVote  uint64
	BLock  uint64
	BExec  uint64
	BLeaf  uint64
	QCHigh uint64
}

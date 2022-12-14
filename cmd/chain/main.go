// Copyright (C) 2021 Aung Maw
// Licensed under the GNU General Public License v3.0

package main

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/wooyang2018/ppov-blockchain/node"
)

const (
	FlagDebug   = "debug"
	FlagDataDir = "dataDir"

	FlagPort    = "port"
	FlagAPIPort = "apiPort"

	FlagBroadcastTx = "broadcast-tx"

	// storage
	FlagMerkleBranchFactor = "storage-merkle-branch-factor"

	// execution
	FlagTxExecTimeout       = "execution-tx-exec-timeout"
	FlagExecConcurrentLimit = "execution-concurrent-limit"

	// consensus
	FlagChainID        = "chainID"
	FlagBlockTxLimit   = "consensus-block-tx-limit"
	FlagTxWaitTime     = "consensus-tx-wait-time"
	FlagProposeTimeout = "consensus-propose-timeout"
	FlagBlockDelay     = "consensus-block-delay"
	FlagViewWidth      = "consensus-view-width"
	FlagLeaderTimeout  = "consensus-leader-timeout"
)

var nodeConfig = node.DefaultConfig

var rootCmd = &cobra.Command{
	Use:   "chain",
	Short: "ppov blockchain",
	Run: func(cmd *cobra.Command, args []string) {
		node.Run(nodeConfig)
	},
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&nodeConfig.Debug,
		FlagDebug, false, "debug mode")

	rootCmd.PersistentFlags().StringVarP(&nodeConfig.Datadir,
		FlagDataDir, "d", "", "blockchain data directory")
	rootCmd.MarkPersistentFlagRequired(FlagDataDir)

	rootCmd.Flags().IntVarP(&nodeConfig.Port,
		FlagPort, "p", nodeConfig.Port, "p2p port")

	rootCmd.Flags().IntVarP(&nodeConfig.APIPort,
		FlagAPIPort, "P", nodeConfig.APIPort, "node api port")

	rootCmd.Flags().BoolVar(&nodeConfig.BroadcastTx,
		FlagBroadcastTx, false, "whether to broadcast transaction")

	rootCmd.Flags().Uint8Var(&nodeConfig.StorageConfig.MerkleBranchFactor,
		FlagMerkleBranchFactor, nodeConfig.StorageConfig.MerkleBranchFactor,
		"merkle tree branching factor")

	rootCmd.Flags().DurationVar(&nodeConfig.ExecutionConfig.TxExecTimeout,
		FlagTxExecTimeout, nodeConfig.ExecutionConfig.TxExecTimeout,
		"tx execution timeout")

	rootCmd.Flags().IntVar(&nodeConfig.ExecutionConfig.ConcurrentLimit,
		FlagExecConcurrentLimit, nodeConfig.ExecutionConfig.ConcurrentLimit,
		"concurrent tx execution limit")

	rootCmd.Flags().Int64Var(&nodeConfig.ConsensusConfig.ChainID,
		FlagChainID, nodeConfig.ConsensusConfig.ChainID,
		"chainid is used to create genesis block")

	rootCmd.Flags().IntVar(&nodeConfig.ConsensusConfig.BatchTxLimit,
		FlagBlockTxLimit, nodeConfig.ConsensusConfig.BatchTxLimit,
		"maximum tx count in a block")

	rootCmd.Flags().DurationVar(&nodeConfig.ConsensusConfig.TxWaitTime,
		FlagTxWaitTime, nodeConfig.ConsensusConfig.TxWaitTime,
		"block creation delay if no transactions in the pool")

	rootCmd.Flags().DurationVar(&nodeConfig.ConsensusConfig.ProposeTimeout,
		FlagProposeTimeout, nodeConfig.ConsensusConfig.ProposeTimeout,
		"duration to wait to propose next block if leader cannot create qc")

	rootCmd.Flags().DurationVar(&nodeConfig.ConsensusConfig.BlockDelay,
		FlagBlockDelay, nodeConfig.ConsensusConfig.BlockDelay,
		"minimum delay between blocks")

	rootCmd.Flags().DurationVar(&nodeConfig.ConsensusConfig.ViewWidth,
		FlagViewWidth, nodeConfig.ConsensusConfig.ViewWidth,
		"view duration for a leader")

	rootCmd.Flags().DurationVar(&nodeConfig.ConsensusConfig.LeaderTimeout,
		FlagLeaderTimeout, nodeConfig.ConsensusConfig.LeaderTimeout,
		"leader must create next qc in this duration")
}

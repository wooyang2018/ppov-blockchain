// Copyright (C) 2021 Aung Maw
// Copyright (C) 2023 Wooyang2018
// Licensed under the GNU General Public License v3.0

package testutil

import (
	"github.com/wooyang2018/ppov-blockchain/core"
	"github.com/wooyang2018/ppov-blockchain/tests/cluster"
)

type LoadClient interface {
	SetupOnCluster(cls *cluster.Cluster) error
	SubmitTx() (int, *core.Transaction, error)
	BatchSubmitTx(num int) (int, *core.TxList, error)
	SubmitTxAndWait() (int, error)
}

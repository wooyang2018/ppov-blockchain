// Copyright (C) 2021 Aung Maw
// Copyright (C) 2023 Wooyang2018
// Licensed under the GNU General Public License v3.0

package experiments

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/wooyang2018/ppov-blockchain/tests/cluster"
	"github.com/wooyang2018/ppov-blockchain/tests/health"
)

type NetworkPacketLoss struct {
	Percent float32
}

func (expm *NetworkPacketLoss) Name() string {
	return fmt.Sprintf("network_packet_loss_%.2f", expm.Percent)
}

func (expm *NetworkPacketLoss) Run(cls *cluster.Cluster) error {
	effects := make([]string, cls.NodeCount())
	for i := 0; i < cls.NodeCount(); i++ {
		percent := expm.Percent + rand.Float32()
		if err := cls.GetNode(i).EffectLoss(percent); err != nil {
			fmt.Println(err)
		}
		effects[i] = fmt.Sprintf("%.2f%%", percent)
	}
	defer cls.RemoveEffects()

	fmt.Printf("Added packet loss %v\n", effects)
	cluster.Sleep(20 * time.Second)
	if err := health.CheckMajorityNodes(cls); err != nil {
		return err
	}

	cls.RemoveEffects()
	fmt.Println("Removed effects")
	cluster.Sleep(10 * time.Second)
	return nil
}

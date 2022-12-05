// Copyright (C) 2021 Aung Maw
// Licensed under the GNU General Public License v3.0

package cluster

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/multiformats/go-multiaddr"

	"github.com/wooyang2018/ppov-blockchain/core"
	"github.com/wooyang2018/ppov-blockchain/node"
)

const WorkerProportion float32 = 1
const VoterProportion float32 = 0.8

func ReadRemoteHosts(hostsPath string, nodeCount int) ([]string, error) {
	raw, err := os.ReadFile(hostsPath)
	if err != nil {
		return nil, err
	}
	hosts := strings.Split(string(raw), "\n")
	if len(hosts) < nodeCount {
		return nil, fmt.Errorf("not enough hosts, %d | %d",
			len(hosts), nodeCount)
	}
	return hosts[:nodeCount], nil
}

func WriteNodeKey(datadir string, key *core.PrivateKey) error {
	f, err := os.Create(path.Join(datadir, node.NodekeyFile))
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(key.Bytes())
	return err
}

func WriteGenesisFile(datadir string, genesis *node.Genesis) error {
	f, err := os.Create(path.Join(datadir, node.GenesisFile))
	if err != nil {
		return err
	}
	defer f.Close()

	e := json.NewEncoder(f)
	e.SetIndent("", "  ")
	return e.Encode(genesis)
}

func WritePeersFile(datadir string, peers []node.Peer) error {
	f, err := os.Create(path.Join(datadir, node.PeersFile))
	if err != nil {
		return err
	}
	defer f.Close()
	e := json.NewEncoder(f)
	e.SetIndent("", "  ")
	return e.Encode(peers)
}

func MakeRandomKeys(count int) []*core.PrivateKey {
	keys := make([]*core.PrivateKey, count)
	for i := 0; i < count; i++ {
		keys[i] = core.GenerateKey(nil)
	}
	return keys
}

func MakePeers(keys []*core.PrivateKey, addrs []multiaddr.Multiaddr) []node.Peer {
	vlds := make([]node.Peer, len(addrs))
	// create validator infos (pubkey + addr)
	for i, addr := range addrs {
		vlds[i] = node.Peer{
			PubKey: keys[i].PublicKey().Bytes(),
			Addr:   addr.String(),
		}
	}
	return vlds
}

func SetupTemplateDir(dir string, keys []*core.PrivateKey, vlds []node.Peer) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := os.Mkdir(dir, 0755); err != nil {
		return err
	}
	genesis := &node.Genesis{
		Workers: make([]string, 0, 0),
		Voters:  make([]string, 0, 0),
		Weights: make([]int, 0, 0),
	}

	workers := PickUniqueRandoms(len(keys), int(float32(len(keys))*WorkerProportion))
	fmt.Printf("Setup workers: %v\n", workers)
	for _, v := range workers {
		genesis.Workers = append(genesis.Workers, keys[v].PublicKey().String())
		genesis.Weights = append(genesis.Weights, 1)
	}

	// Ensure that the node is either a Worker or a Voter
	var voters []int
	unselectedIndexes := GetUnselectedIndexes(len(keys), workers)
	if len(unselectedIndexes) <= int(float32(len(keys))*VoterProportion) {
		voters = append(voters, unselectedIndexes...)
		indexes := PickUniqueRandoms(len(workers), int(float32(len(keys))*VoterProportion)-len(unselectedIndexes))
		for _, v := range indexes {
			voters = append(voters, workers[v])
		}
	} else {
		indexes := PickUniqueRandoms(len(unselectedIndexes), int(float32(len(keys))*VoterProportion))
		for _, v := range indexes {
			voters = append(voters, unselectedIndexes[v])
		}
	}
	fmt.Printf("Setup voters: %v\n", voters)
	for _, v := range voters {
		genesis.Voters = append(genesis.Voters, keys[v].PublicKey().String())
	}

	for i, key := range keys {
		dir := path.Join(dir, strconv.Itoa(i))
		os.Mkdir(dir, 0755)
		if err := WriteNodeKey(dir, key); err != nil {
			return err
		}
		if err := WriteGenesisFile(dir, genesis); err != nil {
			return err
		}
		if err := WritePeersFile(dir, vlds); err != nil {
			return err
		}
	}
	return nil
}

func RunCommand(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	fmt.Printf(" $ %s\n", strings.Join(cmd.Args, " "))
	return cmd.Run()
}

func AddPPoVFlags(cmd *exec.Cmd, config *node.Config) {
	cmd.Args = append(cmd.Args, "-d", config.Datadir)
	cmd.Args = append(cmd.Args, "-p", strconv.Itoa(config.Port))
	cmd.Args = append(cmd.Args, "-P", strconv.Itoa(config.APIPort))
	if config.Debug {
		cmd.Args = append(cmd.Args, "--debug")
	}
	if config.BroadcastTx {
		cmd.Args = append(cmd.Args, "--broadcast-tx")
	}

	cmd.Args = append(cmd.Args, "--storage-merkle-branch-factor",
		strconv.Itoa(int(config.StorageConfig.MerkleBranchFactor)))

	cmd.Args = append(cmd.Args, "--execution-tx-exec-timeout",
		config.ExecutionConfig.TxExecTimeout.String(),
	)
	cmd.Args = append(cmd.Args, "--execution-concurrent-limit",
		strconv.Itoa(config.ExecutionConfig.ConcurrentLimit))

	cmd.Args = append(cmd.Args, "--chainID",
		strconv.Itoa(int(config.ConsensusConfig.ChainID)))

	cmd.Args = append(cmd.Args, "--consensus-block-tx-limit",
		strconv.Itoa(config.ConsensusConfig.BatchTxLimit))

	cmd.Args = append(cmd.Args, "--consensus-tx-wait-time",
		config.ConsensusConfig.TxWaitTime.String())

	cmd.Args = append(cmd.Args, "--consensus-propose-timeout",
		config.ConsensusConfig.ProposeTimeout.String())

	cmd.Args = append(cmd.Args, "--consensus-block-delay",
		config.ConsensusConfig.BlockDelay.String())

	cmd.Args = append(cmd.Args, "--consensus-view-width",
		config.ConsensusConfig.ViewWidth.String())

	cmd.Args = append(cmd.Args, "--consensus-leader-timeout",
		config.ConsensusConfig.LeaderTimeout.String())
}

func PickUniqueRandoms(total, count int) []int {
	rand.Seed(time.Now().Unix())
	unique := make(map[int]struct{}, count)
	for len(unique) < count {
		unique[rand.Intn(total)] = struct{}{}
	}
	ret := make([]int, 0, count)
	for v := range unique {
		ret = append(ret, v)
	}
	return ret
}

func GetUnselectedIndexes(total int, selected []int) []int {
	smap := make(map[int]struct{}, len(selected))
	for _, idx := range selected {
		smap[idx] = struct{}{}
	}
	ret := make([]int, 0, total-len(selected))
	for i := 0; i < total; i++ {
		if _, found := smap[i]; !found {
			ret = append(ret, i)
		}
	}
	return ret
}

// Sleep print duration and call time.Sleep
func Sleep(d time.Duration) {
	fmt.Printf("Wait for %s\n", d)
	time.Sleep(d)
}

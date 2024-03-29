// Copyright (C) 2021 Aung Maw
// Copyright (C) 2023 Wooyang2018
// Licensed under the GNU General Public License v3.0

package execution

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/wooyang2018/ppov-blockchain/chaincode/ppovcoin"
	"github.com/wooyang2018/ppov-blockchain/core"
)

func TestTxExecuter(t *testing.T) {
	assert := assert.New(t)

	priv := core.GenerateKey(nil)
	depInput := &DeploymentInput{
		CodeInfo: CodeInfo{
			DriverType: DriverTypeNative,
			CodeID:     []byte(NativeCodeIDPPoVCoin),
		},
	}
	b, _ := json.Marshal(depInput)
	txDep := core.NewTransaction().SetInput(b).Sign(priv)

	blk := core.NewBlock().SetHeight(10).Sign(priv)

	trk := newStateTracker(newMapStateStore(), nil)
	reg := newCodeRegistry()
	texe := txExecutor{
		codeRegistry: reg,
		timeout:      1 * time.Second,
		txTrk:        trk,
		blk:          blk,
		tx:           txDep,
	}
	txc := texe.execute()

	assert.NotEqual("", txc.Error(), "code driver not registered")

	reg.registerDriver(DriverTypeNative, newNativeCodeDriver())
	txc = texe.execute()

	assert.Equal("", txc.Error())
	assert.Equal(blk.Hash(), txc.BlockHash())
	assert.Equal(blk.Height(), txc.BlockHeight())

	// codeinfo must be saved by key (transaction hash)
	cinfo, err := reg.getCodeInfo(txDep.Hash(), trk.spawn(codeRegistryAddr))

	assert.NoError(err)
	assert.Equal(*cinfo, depInput.CodeInfo)

	cc, err := reg.getInstance(txDep.Hash(), trk.spawn(codeRegistryAddr))

	assert.NoError(err)
	assert.NotNil(cc)

	ccInput := &ppovcoin.Input{
		Method: "minter",
	}
	b, _ = json.Marshal(ccInput)
	minter, err := cc.Query(&callContextTx{
		input:        b,
		stateTracker: trk.spawn(txDep.Hash()),
	})

	assert.NoError(err)
	assert.Equal(priv.PublicKey().Bytes(), minter, "deployer must be set as minter")

	ccInput.Method = "mint"
	ccInput.Dest = priv.PublicKey().Bytes()
	ccInput.Value = 100
	b, _ = json.Marshal(ccInput)

	txInvoke := core.NewTransaction().SetCodeAddr(txDep.Hash()).SetInput(b).Sign(priv)

	texe.tx = txInvoke
	txc = texe.execute()

	assert.Equal("", txc.Error())

	ccInput.Method = "balance"
	ccInput.Value = 0
	b, _ = json.Marshal(ccInput)

	b, err = cc.Query(&callContextTx{
		input:        b,
		stateTracker: trk.spawn(txDep.Hash()),
	})

	var balance int64
	json.Unmarshal(b, &balance)

	assert.NoError(err)
	assert.EqualValues(100, balance)
}

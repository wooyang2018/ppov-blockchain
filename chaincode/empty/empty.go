// Copyright (C) 2023 Wooyang2018
// Licensed under the GNU General Public License v3.0

package empty

import (
	"github.com/wooyang2018/ppov-blockchain/execution/chaincode"
)

// Empty chaincode
type Empty struct{}

var _ chaincode.Chaincode = (*Empty)(nil)

func (c *Empty) Init(ctx chaincode.CallContext) error {
	return nil
}

func (c *Empty) Invoke(ctx chaincode.CallContext) error {
	return nil
}

func (c *Empty) Query(ctx chaincode.CallContext) ([]byte, error) {
	return nil, nil
}

// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"com.tuntun.rangers/node/src/common"
	"fmt"
	"os"
	"sort"
	"testing"
)

func TestSort(t *testing.T) {
	defer func() {
		os.RemoveAll("1.ini")
	}()
	common.Init(0, "1.ini", "dev")
	common.SetBlockHeight(10000000)

	txs := make(Transactions, 0)

	tx := &Transaction{
		Source: "0x01",
		Nonce:  2,
		Data:   "a",
	}
	tx.Hash = tx.GenHash()
	txs = append(txs, tx)

	tx = &Transaction{
		Source: "0x03",
		Nonce:  2,
		Data:   "a",
	}
	tx.Hash = tx.GenHash()
	txs = append(txs, tx)

	tx = &Transaction{
		Source: "0x03",
		Nonce:  2,
		Data:   "aa",
	}
	tx.Hash = tx.GenHash()
	txs = append(txs, tx)

	tx = &Transaction{
		Source: "0x02",
		Nonce:  2,
		Data:   "a",
	}
	tx.Hash = tx.GenHash()
	txs = append(txs, tx)

	tx = &Transaction{
		Source: "0x03",
		Nonce:  1,
		Data:   "aa",
	}
	tx.Hash = tx.GenHash()
	txs = append(txs, tx)

	sort.Sort(txs)
	for _, tx := range txs {
		fmt.Printf("%s-%d-%s-%s\n", tx.Source, tx.Nonce, tx.Data, tx.Hash.String())
	}

	// check sort
	tx = txs[0]
	if tx.Nonce != 1 || tx.Source != "0x03" {
		t.Fatal("fail to sort1")
	}

	tx = txs[1]
	if tx.Nonce != 2 || tx.Source != "0x03" || tx.Data != "aa" {
		t.Fatal("fail to sort2")
	}

}

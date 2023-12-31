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

package cli

import (
	"fmt"
)

var (
	ErrPassword    = fmt.Errorf("password error")
	ErrUnlocked    = fmt.Errorf("please unlock the account first")
	ErrUnConnected = fmt.Errorf("please connect to one node first")
)

type txRawData struct {
	//from string
	Target    string `json:"target"`
	Value     uint64 `json:"value"`
	TxType    int    `json:"tx_type"`
	Nonce     uint64 `json:"nonce"`
	Data      string `json:"data"`
	Sign      string `json:"sign"`
	ExtraData string `json:"extra_data"`
}

func opError(err error) *Result {
	ret, _ := failResult(err.Error())
	return ret
}

func opSuccess(data interface{}) *Result {
	ret, _ := successResult(data)
	return ret
}

type MinerInfo struct {
	PK          string
	VrfPK       string
	ID          string
	Stake       uint64
	NType       byte
	ApplyHeight uint64
	AbortHeight uint64
}

type accountOp interface {
	NewAccount(password string, miner bool) *Result

	AccountList() *Result

	Lock(addr string) *Result

	UnLock(addr string, password string) *Result

	AccountInfo() *Result

	DeleteAccount() *Result

	Close()
}

type chainOp interface {
	Connect(ip string, port int) error

	Endpoint() string

	SendRaw(tx *txRawData) *Result

	Balance(addr string) *Result

	MinerInfo(addr string) *Result

	BlockHeight() *Result

	GroupHeight() *Result

	ApplyMiner(mtype int, stake uint64, gas, gasprice uint64) *Result

	AbortMiner(mtype int, gas, gasprice uint64) *Result

	RefundMiner(mtype int, gas, gasprice uint64) *Result

	TxInfo(hash string) *Result

	BlockByHash(hash string) *Result

	BlockByHeight(h uint64) *Result

	ViewContract(addr string) *Result

	TxReceipt(hash string) *Result
}

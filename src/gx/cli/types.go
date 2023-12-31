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
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/middleware/types"
	"math/big"
)

type Result struct {
	Message string      `json:"message"`
	Status  int         `json:"status"`
	Data    interface{} `json:"data"`
}

func (r *Result) IsSuccess() bool {
	return r.Status == 0
}

type ErrorResult struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type RPCReqObj struct {
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Jsonrpc string        `json:"jsonrpc"`
	ID      uint          `json:"id"`
}

type RPCResObj struct {
	Jsonrpc string       `json:"jsonrpc"`
	ID      uint         `json:"id"`
	Result  *Result      `json:"result,omitempty"`
	Error   *ErrorResult `json:"error,omitempty"`
}

type Transactions struct {
	Hash      string `json:"hash"`
	Source    string `json:"source"`
	Target    string `json:"target"`
	Value     string `json:"value"`
	Height    uint64 `json:"height"`
	BlockHash string `json:"block_hash"`
}

type PubKeyInfo struct {
	PubKey string `json:"pub_key"`
	ID     string `json:"id"`
}

type ConnInfo struct {
	Id      string `json:"id"`
	Ip      string `json:"ip"`
	TcpPort string `json:"tcp_port"`
}

type GroupStat struct {
	Dismissed bool  `json:"dismissed"`
	VCount    int32 `json:"v_count"`
}

type ProposerStat struct {
	Stake      uint64  `json:"stake"`
	StakeRatio float64 `json:"stake_ratio"`
	PCount     int32   `json:"p_count"`
}

type CastStat struct {
	Group    map[string]GroupStat    `json:"group"`
	Proposer map[string]ProposerStat `json:"proposer"`
}

type MortGage struct {
	Stake       uint64 `json:"stake"`
	ApplyHeight uint64 `json:"apply_height"`
	AbortHeight uint64 `json:"abort_height"`
	Type        string `json:"type"`
}

func NewMortGageFromMiner(miner *types.Miner) *MortGage {
	t := "proposal"
	if miner.Type == common.MinerTypeValidator {
		t = "validator"
	}
	mg := &MortGage{
		Stake:       miner.Stake,
		ApplyHeight: miner.ApplyHeight,
		Type:        t,
	}
	return mg
}

type NodeInfo struct {
	ID           string     `json:"id"`
	Balance      string     `json:"balance"`
	Status       string     `json:"status"`
	WGroupNum    int        `json:"w_group_num"`
	AGroupNum    int        `json:"a_group_num"`
	NType        string     `json:"n_type"`
	TxPoolNum    int        `json:"tx_pool_num"`
	BlockHeight  uint64     `json:"block_height"`
	GroupHeight  uint64     `json:"group_height"`
	MortGages    []MortGage `json:"mort_gages"`
	VrfThreshold float64    `json:"vrf_threshold"`
}

type PageObjects struct {
	Total uint64        `json:"count"`
	Data  []interface{} `json:"data"`
}

type Block struct {
	Version     uint64        `json:"version"`
	Height      uint64        `json:"height"`
	Hash        common.Hash   `json:"hash"`
	PreHash     common.Hash   `json:"preHash"`
	CurTime     string        `json:"curTime"`
	PreTime     string        `json:"preTime"`
	Castor      groupsig.ID   `json:"proposer"`
	GroupID     groupsig.ID   `json:"groupId"`
	Signature   string        `json:"sigature"`
	Prove       *big.Int      `json:"prove"`
	TotalQN     uint64        `json:"totalQn"`
	Qn          uint64        `json:"qn"`
	Txs         []common.Hash `json:"txs"`
	EvictedTxs  []common.Hash `json:"wrongTxs"`
	TxNum       uint64        `json:"txCount"`
	StateRoot   common.Hash   `json:"stateRoot"`
	TxRoot      common.Hash   `json:"txRoot"`
	ReceiptRoot common.Hash   `json:"receiptRoot"`
	Random      string        `json:"random"`
}

type BlockDetail struct {
	Block
	Trans []Transaction `json:"txDetails"`
}

type BlockReceipt struct {
	Receipts        []*types.Receipt `json:"receipts"`
	EvictedReceipts []*types.Receipt `json:"evictedReceipts"`
}

type ExplorerBlockDetail struct {
	BlockDetail
	Receipts        []*types.Receipt `json:"receipts"`
	EvictedReceipts []*types.Receipt `json:"evictedReceipts"`
}

type Group struct {
	Height        uint64      `json:"height"`
	Id            groupsig.ID `json:"id"`
	PreId         groupsig.ID `json:"pre_id"`
	ParentId      groupsig.ID `json:"parent_id"`
	BeginHeight   uint64      `json:"begin_height"`
	DismissHeight uint64      `json:"dismiss_height"`
	Members       []string    `json:"members"`
}

type Transaction struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   int32  `json:"type"`

	Signature string `json:"signature"`

	SubTransactions string `json:"subTransactions"`

	Hash common.Hash `json:"hash"`

	Data      string `json:"data"`
	ExtraData string `json:"extraData"`
}

type Dashboard struct {
	BlockHeight uint64     `json:"block_height"`
	GroupHeight uint64     `json:"group_height"`
	WorkGNum    int        `json:"work_g_num"`
	NodeInfo    *NodeInfo  `json:"node_info"`
	Conns       []ConnInfo `json:"conns"`
	Miner       string     `json:"miner"`
	Addr        string     `json:"addr"`
}

type ExplorerAccount struct {
	Balance   *big.Int               `json:"balance"`
	Nonce     uint64                 `json:"nonce"`
	Type      uint32                 `json:"type"`
	CodeHash  string                 `json:"code_hash"`
	Code      string                 `json:"code"`
	StateData map[string]interface{} `json:"state_data"`
}

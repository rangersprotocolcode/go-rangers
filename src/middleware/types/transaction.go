// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"bytes"
	"strconv"

	"com.tuntun.rocket/node/src/common"
)

const (
	TransactionTypeBonus       = 1
	TransactionTypeMinerApply  = 2
	TransactionTypeMinerAbort  = 3
	TransactionTypeMinerRefund = 4
	TransactionTypeMinerAdd    = 5

	//以下交易类型会被外部使用 禁止更改
	TransactionTypeOperatorBalance   = 99
	TransactionTypeOperatorEvent     = 100 // 调用状态机/转账
	TransactionTypeGetCoin           = 101 // 查询主链币
	TransactionTypeGetAllCoin        = 102 // 查询所有主链币
	TransactionTypeFT                = 103 // 查询特定FT
	TransactionTypeAllFT             = 104 // 查询所有FT
	TransactionTypeNFT               = 105 // 根据setId、id查询特定NFT
	TransactionTypeNFTListByAddress  = 106 // 查询账户下所有NFT
	TransactionTypeNFTSet            = 107 // 查询NFTSet信息
	TransactionTypeStateMachineNonce = 108 // 调用状态机nonce(预留接口）
	TransactionTypeFTSet             = 113 // 根据ftId, 查询ftSet信息
	TransactionTypeNFTCount          = 114 // 查询用户Rocket上的指定NFT的拥有数量
	TransactionTypeNFTList           = 115 // 查询用户Rocket上的指定NFT的拥有数量
	TransactionTypeNFTGtZero         = 118 // 查询指定用户Rocket上的余额大于0的非同质化代币列表

	TransactionTypeWithdraw = 109

	TransactionTypePublishFT      = 110 // 用户发FTSet
	TransactionTypePublishNFTSet  = 111 // 用户发NFTSet
	TransactionTypeShuttleNFT     = 112 // 用户穿梭NFT
	TransactionTypeMintFT         = 116 // mintFT
	TransactionTypeMintNFT        = 117 // mintNFT
	TransactionTypeTransferBNT    = 127 // 状态机给用户转主链币
	TransactionTypeTransferFT     = 119 // 状态机给用户转FT
	TransactionTypeLockNFT        = 120 // 锁定NFT
	TransactionTypeUnLockNFT      = 121 // 解锁NFT
	TransactionTypeApproveNFT     = 122 // 授权NFT
	TransactionTypeRevokeNFT      = 123 // 回收NFT
	TransactionTypeTransferNFT    = 124 // 状态机给用户转NFT
	TransactionTypeUpdateNFT      = 125 // 更新NFT数据
	TransactionTypeBatchUpdateNFT = 126 // 批量更新NFT数据 deprecated
	TransactionTypeImportNFT      = 128 // 从外部导入NFT/NFTSet

	TransactionTypeLockResource   = 129 // 锁定 nft/ft/bnt
	TransactionTypeUnLockResource = 130 // 解锁 nft/ft/bnt
	TransactionTypeComboNFT       = 131 // 组合nft

	// 状态机通知客户端
	TransactionTypeNotify          = 301 // 通知某个用户
	TransactionTypeNotifyGroup     = 302 // 通知某个组
	TransactionTypeNotifyBroadcast = 303 // 通知所有人

	// 从rocket_connector来的消息
	TransactionTypeCoinDepositAck = 201 // 充值
	TransactionTypeFTDepositAck   = 202 // 充值
	TransactionTypeNFTDepositAck  = 203 // 充值

	// 状态机管理
	TransactionTypeAddStateMachine = 901 // 新增状态机
	TransactionTypeUpdateStorage   = 902 // 刷新状态机存储
	TransactionTypeStartSTM        = 903 // 重启状态机
	TransactionTypeStopSTM         = 904 // 关闭状态机
	TransactionTypeUpgradeSTM      = 905 // 更新状态机（停机->删除本地镜像->下载新镜像->启动）
	TransactionTypeQuitSTM         = 906 // 关服（停机->删除本地镜像->删除配置项）

	// 系统管理
	TransactionTypeSetExchangeRate = 801 // 新增汇率表

	TransactionTypeWrongTxNonce = 404
)

type Transaction struct {
	Source string // 用户id
	Target string // 游戏id
	Type   int32  // 场景id
	Time   string

	Data            string // 状态机入参
	ExtraData       string // 在rocketProtocol里，用于转账。包括余额转账、FT转账、NFT转账
	ExtraDataType   int32
	SubTransactions []UserData // 用于存储状态机rpc调用的交易数据
	SubHash         common.Hash

	Hash common.Hash
	Sign *common.Sign

	Nonce           uint64 // 用户级别nonce
	RequestId       uint64 // 消息编号 由网关添加
	SocketRequestId string // websocket id，用于客户端标示请求id，方便回调处理
}

//source 在hash计算范围内
//RequestId 不列入hash计算范围
func (tx *Transaction) GenHash() common.Hash {
	if nil == tx {
		return common.Hash{}
	}
	buffer := bytes.Buffer{}

	buffer.Write([]byte(tx.Data))
	buffer.Write([]byte(strconv.FormatUint(tx.Nonce, 10)))
	buffer.Write([]byte(tx.Source))
	buffer.Write([]byte(tx.Target))
	buffer.Write([]byte(strconv.Itoa(int(tx.Type))))
	buffer.Write([]byte(tx.Time))
	buffer.Write([]byte(tx.ExtraData))
	return common.BytesToHash(common.Sha256(buffer.Bytes()))
}

func (tx *Transaction) AppendSubTransaction(sub UserData) {
	tx.SubTransactions = append(tx.SubTransactions, sub)
	buffer := bytes.Buffer{}
	buffer.Write(sub.Hash())
	buffer.Write(tx.SubHash.Bytes())

	//todo: 性能优化点
	tx.SubHash = common.BytesToHash(common.Sha256(buffer.Bytes()))
}

func (tx *Transaction) GenHashes() common.Hashes {
	if nil == tx {
		return common.Hashes{}
	}

	result := common.Hashes{}
	result[0] = tx.Hash
	result[1] = tx.SubHash

	return result
}

type Transactions []*Transaction

func (c Transactions) Len() int {
	return len(c)
}
func (c Transactions) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
func (c Transactions) Less(i, j int) bool {
	if c[i].RequestId == 0 && c[j].RequestId == 0 {
		return c[i].Nonce < c[j].Nonce
	}

	return c[i].RequestId < c[j].RequestId
}

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
	"encoding/json"
	"math/big"
	"time"

	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/utility"
)

type AddBlockOnChainSituation string

const (
	Sync                  AddBlockOnChainSituation = "sync"
	NewBlock              AddBlockOnChainSituation = "newBlock"
	FutureBlockCache      AddBlockOnChainSituation = "futureBlockCache"
	DependGroupBlock      AddBlockOnChainSituation = "dependGroupBlock"
	LocalGenerateNewBlock AddBlockOnChainSituation = "localGenerateNewBlock"
	MergeFork             AddBlockOnChainSituation = "mergeFork"
)

type AddBlockResult int8

const (
	AddBlockFailed            AddBlockResult = -1
	AddBlockSucc              AddBlockResult = 0
	BlockExisted              AddBlockResult = 1
	BlockTotalQnLessThanLocal AddBlockResult = 2
	Forking                   AddBlockResult = 3
	ValidateBlockOk           AddBlockResult = 100
)

const (
	SUCCESS                             = 0
	TxErrorCode_BalanceNotEnough        = 1
	TxErrorCode_ContractAddressConflict = 2
	TxErrorCode_DeployGasNotEnough      = 3
	TxErrorCode_NO_CODE                 = 4

	Syntax_Error = 1001
	GasNotEnough = 1002

	Sys_Error                        = 2001
	Sys_Check_Abi_Error              = 2002
	Sys_Abi_JSON_Error               = 2003
	Sys_CONTRACT_CALL_MAX_DEEP_Error = 2004
)

var (
	NO_CODE_ERROR           = 4
	NO_CODE_ERROR_MSG       = "get code from address %s,but no code!"
	ABI_JSON_ERROR          = 2003
	ABI_JSON_ERROR_MSG      = "abi json format error"
	CALL_MAX_DEEP_ERROR     = 2004
	CALL_MAX_DEEP_ERROR_MSG = "call max deep cannot more than 8"
	INIT_CONTRACT_ERROR     = 2005
	INIT_CONTRACT_ERROR_MSG = "contract init error"
)

var (
	TxErrorBalanceNotEnough   = NewTransactionError(TxErrorCode_BalanceNotEnough, "balance not enough")
	TxErrorDeployGasNotEnough = NewTransactionError(TxErrorCode_DeployGasNotEnough, "gas not enough")
	TxErrorAbiJson            = NewTransactionError(Sys_Abi_JSON_Error, "abi json format error")
)

type TransactionError struct {
	Code    int
	Message string
}

func NewTransactionError(code int, msg string) *TransactionError {
	return &TransactionError{Code: code, Message: msg}
}

//区块头结构
type BlockHeader struct {
	Hash         common.Hash // 本块的hash，to do : 是对哪些数据的哈希
	Height       uint64      // 本块的高度
	PreHash      common.Hash //上一块哈希
	PreTime      time.Time   //上一块铸块时间
	ProveValue   *big.Int    //轮转序号
	TotalQN      uint64      //整条链的QN
	CurTime      time.Time   //当前铸块时间
	Castor       []byte      //出块人ID
	GroupId      []byte      //组ID，groupsig.ID的二进制表示
	Signature    []byte      // 组签名
	Nonce        uint64      //盐 当前含义为版本号
	RequestIds   map[string]uint64
	Transactions []common.Hashes // 交易集哈希列表
	TxTree       common.Hash     // 交易默克尔树根hash
	ReceiptTree  common.Hash
	StateTree    common.Hash
	ExtraData    []byte
	Random       []byte
	EvictedTxs   []common.Hash
}

// 辅助
type header struct {
	Height       uint64      // 本块的高度
	PreHash      common.Hash //上一块哈希
	PreTime      time.Time   //上一块铸块时间
	ProveValue   *big.Int    //轮转序号
	TotalQN      uint64      //整条链的QN
	CurTime      time.Time   //当前铸块时间
	Castor       []byte      //出块人ID
	GroupId      []byte      //组ID，groupsig.ID的二进制表示
	Nonce        uint64      //盐
	RequestId    map[string]uint64
	Transactions []common.Hashes // 交易集哈希列表
	TxTree       common.Hash     // 交易默克尔树根hash
	ReceiptTree  common.Hash
	StateTree    common.Hash
	ExtraData    []byte
	ProveRoot    common.Hash
	EvictedTxs   []common.Hash
}

func (bh *BlockHeader) GenHash() common.Hash {
	header := &header{
		Height:       bh.Height,
		PreHash:      bh.PreHash,
		PreTime:      bh.PreTime,
		ProveValue:   bh.ProveValue,
		TotalQN:      bh.TotalQN,
		CurTime:      bh.CurTime,
		Castor:       bh.Castor,
		Nonce:        bh.Nonce,
		RequestId:    bh.RequestIds,
		Transactions: bh.Transactions,
		TxTree:       bh.TxTree,
		ReceiptTree:  bh.ReceiptTree,
		StateTree:    bh.StateTree,
		ExtraData:    bh.ExtraData,
		//ProveRoot:    bh.ProveRoot,
		EvictedTxs: bh.EvictedTxs,
	}
	blockByte, _ := json.Marshal(header)
	result := common.BytesToHash(common.Sha256(blockByte))

	return result
}

func (bh *BlockHeader) ToString() string {
	header := &header{
		Height:       bh.Height,
		PreHash:      bh.PreHash,
		PreTime:      bh.PreTime,
		ProveValue:   bh.ProveValue,
		TotalQN:      bh.TotalQN,
		CurTime:      bh.CurTime,
		Castor:       bh.Castor,
		GroupId:      bh.GroupId,
		Nonce:        bh.Nonce,
		Transactions: bh.Transactions,
		TxTree:       bh.TxTree,
		ReceiptTree:  bh.ReceiptTree,
		StateTree:    bh.StateTree,
		ExtraData:    bh.ExtraData,
		//ProveRoot:    bh.ProveRoot,
		EvictedTxs: bh.EvictedTxs,
	}
	blockByte, _ := json.Marshal(header)
	return string(blockByte)
}

type Block struct {
	Header       *BlockHeader
	Transactions []*Transaction
}

type Member struct {
	Id     []byte
	PubKey []byte
}

type GroupHeader struct {
	Hash            common.Hash //组头hash
	Parent          []byte      //父亲组 的组ID
	PreGroup        []byte      //前一块的ID
	CreateBlockHash []byte      //创建组的块HASH
	BeginTime       time.Time
	MemberRoot      common.Hash //成员列表hash
	CreateHeight    uint64      //建组高度
	ReadyHeight     uint64      //准备就绪最迟高度
	WorkHeight      uint64      //组开始参与铸块的高度
	DismissHeight   uint64      //组解散的高度
	Extends         string      //带外数据
}

func (gh *GroupHeader) GenHash() common.Hash {
	buf := bytes.Buffer{}
	buf.Write(gh.Parent)
	buf.Write(gh.PreGroup)
	buf.Write(gh.CreateBlockHash)

	//bt, _ := gh.BeginTime.MarshalBinary()
	//buf.Write(bt)
	buf.Write(gh.MemberRoot.Bytes())
	buf.Write(common.Uint64ToByte(gh.CreateHeight))
	buf.WriteString(gh.Extends)
	return common.BytesToHash(common.Sha256(buf.Bytes()))
}

type Group struct {
	Header *GroupHeader
	//不参与签名
	Id          []byte
	PubKey      []byte
	Signature   []byte
	Members     [][]byte //成员id列表
	GroupHeight uint64
}

// 用于状态机内通过SDK调用layer2的数据结构
type UserData struct {
	Address string `json:"address"`
	TransferData
	Assets map[string]string
}

func (sub *UserData) Hash() []byte {
	buffer := bytes.Buffer{}
	data, _ := json.Marshal(sub)
	buffer.Write(data)

	return common.Sha256(buffer.Bytes())
}

// 转账时写在extraData里的复杂结构，用于转账NFT、FT以及余额
type TransferData struct {
	Balance string            `json:"balance,omitempty"`
	Coin    map[string]string `json:"coin,omitempty"`
	FT      map[string]string `json:"ft,omitempty"`
	NFT     []NFTID           `json:"nft,omitempty"`
}

type NFTID struct {
	SetId    string `json:"setId,omitempty"`
	Id       string `json:"id,omitempty"`
	Data     string `json:"data,omitempty"`
	Property string `json:"property,omitempty"`
}

type FTID struct {
	Id       string `json:"id,omitempty"`
	Value     string `json:"value,omitempty"`
}

//提现时写在Data里的负载结构，用于提现余额，FT,NFT到不同的公链
type WithDrawReq struct {
	Address string          `json:"address,omitempty"`
	Balance string          `json:"balance,omitempty"`
	BNT     BNTWithdrawInfo `json:"bnt,omitempty"`

	ChainType string            `json:"chainType,omitempty"`
	FT        map[string]string `json:"ft,omitempty"`
	NFT       []NFTID           `json:"nft,omitempty"`
}

type WithDrawData struct {
	Address string          `json:"address,omitempty"`
	BNT     BNTWithdrawInfo `json:"bnt,omitempty"`

	ChainType string            `json:"chainType,omitempty"`
	FT        map[string]string `json:"ft,omitempty"`
	NFT       []NFTID           `json:"nft,omitempty"`
}

type BNTWithdrawInfo struct {
	TokenType string `json:"tokenType,omitempty"`
	Value     string `json:"value,omitempty"`
}

type TxJson struct {
	// 用户id
	Source string `json:"source"`
	// 游戏id
	Target string `json:"target"`
	// 场景id
	Type int32  `json:"type"`
	Time string `json:"time,omitempty"`

	// 入参
	Data      string `json:"data,omitempty"`
	ExtraData string `json:"extraData,omitempty"`

	Hash string `json:"hash,omitempty"`
	Sign string `json:"sign,omitempty"`

	Nonce           uint64 `json:"nonce,omitempty"`
	RequestId       uint64
	SocketRequestId string `json:"socketRequestId,omitempty"`
}

func (txJson TxJson) ToTransaction() Transaction {
	tx := Transaction{Source: txJson.Source, Target: txJson.Target, Type: txJson.Type, Time: txJson.Time,
		Data: txJson.Data, ExtraData: txJson.ExtraData, Nonce: txJson.Nonce,
		RequestId: txJson.RequestId, SocketRequestId: txJson.SocketRequestId}

	//tx from coiner cal hash by layer2
	//tx from coiner sign make sign nil
	if tx.Type == TransactionTypeCoinDepositAck || tx.Type == TransactionTypeFTDepositAck || tx.Type == TransactionTypeNFTDepositAck {
		tx.Hash = tx.GenHash()
		return tx
	}

	if txJson.Hash != "" {
		tx.Hash = common.HexToHash(txJson.Hash)
	}

	if txJson.Sign != "" {
		tx.Sign = common.HexStringToSign(txJson.Sign)
	}
	return tx
}

func (txJson TxJson) ToString() string {
	byte, err := json.Marshal(txJson)
	if err != nil {
		logger.Errorf("Json marshal tx error:%s", err.Error())
		return ""
	}
	return string(byte)
}

func (tx Transaction) ToTxJson() TxJson {
	txJson := TxJson{Source: tx.Source, Target: tx.Target, Type: tx.Type, Time: tx.Time,
		Data: tx.Data, ExtraData: tx.ExtraData, Nonce: tx.Nonce,
		Hash: tx.Hash.String(), RequestId: tx.RequestId, SocketRequestId: tx.SocketRequestId}

	if tx.Sign != nil {
		txJson.Sign = tx.Sign.GetHexString()
	}
	return txJson
}

type JSONObject struct {
	data map[string]interface{}
}

func NewJSONObject() JSONObject {
	obj := JSONObject{}
	obj.data = make(map[string]interface{})
	return obj
}

func (object *JSONObject) IsEmpty() bool {
	if 0 == len(object.data) {
		return true
	}

	return false
}

func (object *JSONObject) Put(key string, value interface{}) {
	object.data[key] = value
}

func (object *JSONObject) Remove(key string) interface{} {
	value := object.data[key]
	delete(object.data, key)

	return value
}

func (object *JSONObject) Merge(target *JSONObject, merge func(one, other interface{}) interface{}) {
	if target == nil {
		return
	}

	for key, value := range target.data {
		thisValue := object.data[key]
		object.Put(key, merge(thisValue, value))
	}
}

func (object *JSONObject) TOJSONString() string {
	dataBytes, _ := json.Marshal(object.data)

	return string(dataBytes)
}

func (object *JSONObject) GetData() map[string]interface{} {
	return object.data
}

//func (object *JSONObject) MarshalJSON() ([]byte, error) {
//	data := bytes.Buffer{}
//	for k, v := range object.Data {
//		value, err := json.Marshal(v)
//		if err != nil {
//			return nil, err
//		}
//
//		data.Write([]byte(k))
//		data.Write(value)
//	}
//
//	return data.Bytes(), nil
//}

func ReplaceBigInt(one, other interface{}) interface{} {
	bigInt := other.(*big.Int)
	return utility.BigIntToStr(bigInt)
}

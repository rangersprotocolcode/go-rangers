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

type AddBlockResult int8

const (
	AddBlockFailed            AddBlockResult = -1
	AddBlockSucc              AddBlockResult = 0
	BlockExisted              AddBlockResult = 1
	BlockTotalQnLessThanLocal AddBlockResult = 2
	NoPreOnChain              AddBlockResult = 3
	DependOnGroup             AddBlockResult = 4
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

type BlockHeader struct {
	Hash         common.Hash
	Height       uint64
	PreHash      common.Hash
	PreTime      time.Time
	ProveValue   *big.Int
	TotalQN      uint64
	CurTime      time.Time
	Castor       []byte
	GroupId      []byte
	Signature    []byte
	Nonce        uint64
	RequestIds   map[string]uint64
	Transactions []common.Hashes
	TxTree       common.Hash
	ReceiptTree  common.Hash
	StateTree    common.Hash
	ExtraData    []byte
	Random       []byte
	EvictedTxs   []common.Hash
}

type header struct {
	Height       uint64
	PreHash      common.Hash
	PreTime      time.Time
	ProveValue   *big.Int
	TotalQN      uint64
	CurTime      time.Time
	Castor       []byte
	GroupId      []byte
	Nonce        uint64
	RequestId    map[string]uint64
	Transactions []common.Hashes
	TxTree       common.Hash
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
		EvictedTxs:   bh.EvictedTxs,
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
	Hash            common.Hash
	Parent          []byte
	PreGroup        []byte
	CreateBlockHash []byte
	BeginTime       time.Time
	MemberRoot      common.Hash
	CreateHeight    uint64
	ReadyHeight     uint64
	WorkHeight      uint64
	DismissHeight   uint64
	Extends         string
}

func (gh *GroupHeader) GenHash() common.Hash {
	buf := bytes.Buffer{}
	buf.Write(gh.Parent)
	buf.Write(gh.PreGroup)
	buf.Write(gh.CreateBlockHash)

	buf.Write(gh.MemberRoot.Bytes())
	buf.Write(utility.UInt64ToByte(gh.CreateHeight))
	buf.WriteString(gh.Extends)
	return common.BytesToHash(common.Sha256(buf.Bytes()))
}

type Group struct {
	Header      *GroupHeader
	Id          []byte
	PubKey      []byte
	Signature   []byte
	Members     [][]byte
	GroupHeight uint64
}

type UserData struct {
	Address uint64 `json:"address"`
	TransferData
	Assets map[string]string
}

func (sub *UserData) Hash() []byte {
	buffer := bytes.Buffer{}
	data, _ := json.Marshal(sub)
	buffer.Write(data)

	return common.Sha256(buffer.Bytes())
}

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
	Id    string `json:"id,omitempty"`
	Value string `json:"value,omitempty"`
}

type TxJson struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   int32  `json:"type"`
	Time   string `json:"time,omitempty"`

	Data      string `json:"data,omitempty"`
	ExtraData string `json:"extraData,omitempty"`

	Hash string `json:"hash,omitempty"`
	Sign string `json:"sign,omitempty"`

	Nonce           uint64 `json:"nonce,omitempty"`
	RequestId       uint64
	SocketRequestId string `json:"socketRequestId,omitempty"`

	ChainId string `json:"chainId,omitempty"`
}

func (txJson TxJson) ToTransaction() Transaction {
	tx := Transaction{Source: txJson.Source, Target: txJson.Target, Type: txJson.Type, Time: txJson.Time,
		Data: txJson.Data, ExtraData: txJson.ExtraData, Nonce: txJson.Nonce,
		RequestId: txJson.RequestId, SocketRequestId: txJson.SocketRequestId, ChainId: txJson.ChainId}

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
		Hash: tx.Hash.String(), RequestId: tx.RequestId, SocketRequestId: tx.SocketRequestId, ChainId: tx.ChainId}

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

func ReplaceBigInt(one, other interface{}) interface{} {
	bigInt := other.(*big.Int)
	return utility.BigIntToStr(bigInt)
}

type ContractData struct {
	GasPrice string `json:"gasPrice,omitempty"`
	GasLimit string `json:"gasLimit,omitempty"`

	TransferValue string `json:"transferValue,omitempty"`
	AbiData       string `json:"abiData,omitempty"`
}

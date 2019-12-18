package types

import (
	"encoding/json"
	"time"
	"math/big"
	"bytes"

	"x/src/common"
	"x/src/utility"
	"strconv"
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

const (
	TransactionTypeBonus       = 1
	TransactionTypeMinerApply  = 2
	TransactionTypeMinerAbort  = 3
	TransactionTypeMinerRefund = 4

	//以下交易类型会被外部使用 禁止更改
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
	TransactionTypeTransferBNT    = 118 // 状态机给用户转主链币
	TransactionTypeTransferFT     = 119 // 状态机给用户转FT
	TransactionTypeLockNFT        = 120 // 锁定NFT
	TransactionTypeUnLockNFT      = 121 // 解锁NFT
	TransactionTypeApproveNFT     = 122 // 授权NFT
	TransactionTypeRevokeNFT      = 123 // 回收NFT
	TransactionTypeTransferNFT    = 124 // 状态机给用户转NFT
	TransactionTypeUpdateNFT      = 125 // 更新NFT数据
	TransactionTypeBatchUpdateNFT = 126 // 批量更新NFT数据

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
	TransactionTypeStartSTM        = 903 // 重启状态机存储
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
	buffer.Write([]byte( strconv.Itoa(int(tx.Type))))
	buffer.Write([]byte(tx.Time))
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

const (
	MinerTypeLight    = 0
	MinerTypeHeavy    = 1
	MinerStatusNormal = 0
	MinerStatusAbort  = 1
)

type Miner struct {
	Id           []byte
	PublicKey    []byte
	VrfPublicKey []byte
	ApplyHeight  uint64
	Stake        uint64
	AbortHeight  uint64
	Type         byte
	Status       byte
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
	Nonce        uint64      //盐
	RequestIds   map[string]uint64
	Transactions []common.Hashes // 交易集哈希列表
	TxTree       common.Hash     // 交易默克尔树根hash
	ReceiptTree  common.Hash
	StateTree    common.Hash
	ExtraData    []byte
	Random       []byte
	//ProveRoot    common.Hash
	EvictedTxs []common.Hash
}

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
	Hash          common.Hash //组头hash
	Parent        []byte      //父亲组 的组ID
	PreGroup      []byte      //前一块的ID
	Authority     uint64      //权限相关数据（父亲组赋予）
	Name          string      //父亲组取的名字
	BeginTime     time.Time
	MemberRoot    common.Hash //成员列表hash
	CreateHeight  uint64      //建组高度
	ReadyHeight   uint64      //准备就绪最迟高度
	WorkHeight    uint64      //组开始参与铸块的高度
	DismissHeight uint64      //组解散的高度
	Extends       string      //带外数据
}

func (gh *GroupHeader) GenHash() common.Hash {
	buf := bytes.Buffer{}
	buf.Write(gh.Parent)
	buf.Write(gh.PreGroup)
	buf.Write(common.Uint64ToByte(gh.Authority))
	buf.WriteString(gh.Name)

	//bt, _ := gh.BeginTime.MarshalBinary()
	//buf.Write(bt)
	buf.Write(gh.MemberRoot.Bytes())
	buf.Write(common.Uint64ToByte(gh.CreateHeight))
	buf.Write(common.Uint64ToByte(gh.ReadyHeight))
	buf.Write(common.Uint64ToByte(gh.WorkHeight))
	buf.Write(common.Uint64ToByte(gh.DismissHeight))
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
	Assets  map[string]string
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
	SetId string `json:"setId,omitempty"`
	Id    string `json:"id,omitempty"`
	Data  string `json:"data,omitempty"`
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
	txJson := TxJson{Source: tx.Source, Target: tx.Target, Type: tx.Type,
		Time: tx.Time, Data: tx.Data, ExtraData: tx.ExtraData,
		Hash: tx.Hash.String(), Nonce: tx.Nonce, RequestId: tx.RequestId, SocketRequestId: tx.SocketRequestId}

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

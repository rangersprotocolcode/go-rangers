package types

import (
	"encoding/json"
	"time"
	"math/big"
	"bytes"

	"x/src/common"
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
	TransactionTypeTransfer        = 0
	TransactionTypeContractCreate  = 1
	TransactionTypeContractCall    = 2
	TransactionTypeBonus           = 3
	TransactionTypeMinerApply      = 4
	TransactionTypeMinerAbort      = 5
	TransactionTypeMinerRefund     = 6
	TransactionTypeBlockEvent      = 7
	TransactionTypeOperatorEvent   = 8
	TransactionTypeUserEvent       = 9
	TransactionUpdateOperatorEvent = 10
	GetBalance                     = 11
	GetAsset                       = 12
	GetAllAssets                   = 13
	StateMachineNonce              = 14

	TransactionTypeDepositExecute          = 101
	TransactionTypeDepositAck              = 102
	TransactionTypeWithdrawExecute         = 103
	TransactionTypeWithDrawAck             = 104
	TransactionTypeWithAssetOnChainExecute = 105
	TransactionTypeWithDrawAssetOnChainAck = 106

	// 客户端发送的请求，提现资金与提现资产
	TransactionTypeWithdraw     = 201
	TransactionTypeAssetOnChain = 202

	TransactionTypeToBeRemoved = -1
)

type Transaction struct {
	Data      string // 入参
	Nonce     uint64 // 用户级别nonce
	Source    string // 用户id
	Target    string // 游戏id
	Type      int32  // 场景id
	RequestId uint64 // 消息编号
	Hash      common.Hash

	ExtraData     string
	ExtraDataType int32

	Sign *common.Sign
	Time string
}

//source 在hash计算范围内
//RequestId 不列入hash计算范围
func (tx *Transaction) GenHash() common.Hash {
	if nil == tx {
		return common.Hash{}
	}
	buffer := bytes.Buffer{}

	buffer.Write([]byte(tx.Data))

	buffer.Write(common.Uint64ToByte(tx.Nonce))

	if tx.Source != "" {
		buffer.Write([]byte(tx.Source))
	}

	if tx.Target != "" {
		buffer.Write([]byte(tx.Target))
	}

	buffer.Write(common.UInt32ToByte(tx.Type))

	if tx.Time != "" {
		buffer.Write([]byte(tx.Time))
	}
	return common.BytesToHash(common.Sha256(buffer.Bytes()))
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

type Bonus struct {
	TxHash     common.Hash
	TargetIds  []int32
	BlockHash  common.Hash
	GroupId    []byte
	Sign       []byte
	TotalValue uint64
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
	Transactions []common.Hash // 交易集哈希列表
	TxTree       common.Hash   // 交易默克尔树根hash
	ReceiptTree  common.Hash
	StateTree    common.Hash
	ExtraData    []byte
	Random       []byte
	ProveRoot    common.Hash
	EvictedTxs   []common.Hash
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
	Transactions []common.Hash // 交易集哈希列表
	TxTree       common.Hash   // 交易默克尔树根hash
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
		ProveRoot:    bh.ProveRoot,
		EvictedTxs:   bh.EvictedTxs,
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
		ProveRoot:    bh.ProveRoot,
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

type StateNode struct {
	Key   []byte
	Value []byte
}

type SubAccount struct {
	Balance *big.Int
	Nonce   uint64
	Assets  map[string]string
}

// 仅仅供资产上链使用
type Asset struct {
	Id string `json:"id"`

	Value string `json:"value"`
}

type UserData struct {
	Address string            `json:"address"`
	Balance string            `json:"balance"`
	Assets  map[string]string `json:"assets"`
}

type TxJson struct {
	Source string // 用户id
	Target string // 游戏id
	Type   int32  // 场景id

	Data  string // 入参
	Nonce uint64

	RequestId uint64

	Hash string
	Sign string
	Time string

	ExtraData string
}

func (txJson TxJson) ToTransaction() Transaction {
	tx := Transaction{Source: txJson.Source, Target: txJson.Target,
		Type: txJson.Type, Data: txJson.Data, Nonce: txJson.Nonce,
		RequestId: txJson.RequestId, ExtraData: txJson.ExtraData}

	if txJson.Hash != "" {
		tx.Hash = common.HexToHash(txJson.Hash)
	}

	if txJson.Sign != "" {
		tx.Sign = common.HexStringToSign(txJson.Sign)
	}
	tx.Time = txJson.Time
	return tx
}

func (tx Transaction) ToTxJson() TxJson {
	txJson := TxJson{Source: tx.Source, Target: tx.Target,
		Type: tx.Type, Data: tx.Data, Nonce: tx.Nonce,
		RequestId: tx.RequestId, Hash: tx.Hash.String(), Time: tx.Time}

	if tx.Sign != nil {
		txJson.Sign = tx.Sign.GetHexString()
	}
	return txJson
}

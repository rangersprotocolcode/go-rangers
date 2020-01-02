package cli

import (
	"x/src/common"
	"x/src/consensus/groupsig"
	"math/big"
	"x/src/middleware/types"
)

// Result rpc请求成功返回的可变参数部分
type Result struct {
	Message string      `json:"message"`
	Status  int         `json:"status"`
	Data    interface{} `json:"data"`
}

func (r *Result) IsSuccess() bool {
	return r.Status == 0
}

// ErrorResult rpc请求错误返回的可变参数部分
type ErrorResult struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// RPCReqObj 完整的rpc请求体
type RPCReqObj struct {
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Jsonrpc string        `json:"jsonrpc"`
	ID      uint          `json:"id"`
}

// RPCResObj 完整的rpc返回体
type RPCResObj struct {
	Jsonrpc string       `json:"jsonrpc"`
	ID      uint         `json:"id"`
	Result  *Result      `json:"result,omitempty"`
	Error   *ErrorResult `json:"error,omitempty"`
}

// 缓冲池交易列表中的transactions
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
	t := "重节点"
	if miner.Type == types.MinerTypeLight {
		t = "轻节点"
	}
	mg := &MortGage{
		Stake:       uint64(float64(miner.Stake) / float64(1000000000)),
		ApplyHeight: miner.ApplyHeight,
		AbortHeight: miner.AbortHeight,
		Type:        t,
	}
	return mg
}

type NodeInfo struct {
	ID           string     `json:"id"`
	Balance      float64    `json:"balance"`
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
}

//
//type BonusInfo struct {
//	BlockHeight uint64      `json:"block_height"`
//	BlockHash   common.Hash `json:"block_hash"`
//	BonusTxHash common.Hash `json:"bonus_tx_hash"`
//	GroupId     string      `json:"group_id"`
//	CasterId    string      `json:"caster_id"`
//	GroupIdW     string      `json:"group_id_w"`
//	CasterIdW    string      `json:"caster_id_W"`
//	MemberIds   []string    `json:"members"`
//	BonusValue  uint64      `json:"bonus_value"`
//}
//
//type BonusStatInfo struct {
//	MemberId        string `json:"member_id"`
//	MemberIdW        string `json:"member_id_w"`
//	BonusNum        uint64 `json:"bonus_num"`
//	TotalBonusValue uint64 `json:"total_bonus_value"`
//}
//
//type CastBlockStatInfo struct {
//	CasterId     string `json:"caster_id"`
//	CasterIdW     string `json:"caster_id_w"`
//	Stake        uint64 `json:"stake"`
//	CastBlockNum uint64 `json:"cast_block_num"`
//}
//
//type CastBlockAndBonusResult struct {
//	BonusInfoAtHeight  BonusInfo           `json:"bonus_info_at_height"`
//	BonusStatInfos     []BonusStatInfo     `json:"bonuses"`
//	CastBlockStatInfos []CastBlockStatInfo `json:"cast_blocks"`
//}

type ExplorerAccount struct {
	Balance   *big.Int               `json:"balance"`
	Nonce     uint64                 `json:"nonce"`
	Type      uint32                 `json:"type"`
	CodeHash  string                 `json:"code_hash"`
	Code      string                 `json:"code"`
	StateData map[string]interface{} `json:"state_data"`
}

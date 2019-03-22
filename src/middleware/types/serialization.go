package types

import (
	"fmt"
	"github.com/gogo/protobuf/proto"
	"x/src/middleware/pb"

	"x/src/common"
	"math/big"
	"time"

	"x/src/middleware/log"
)

var logger log.Logger

func InitSerialzation() {
	logger = log.GetLoggerByIndex(log.MiddlewareLogConfig, common.GlobalConf.GetString("instance", "index", ""))
}

// 从[]byte反序列化为*Transaction
func UnMarshalTransaction(b []byte) (Transaction, error) {
	t := new(middleware_pb.Transaction)
	error := proto.Unmarshal(b, t)
	if error != nil {
		logger.Errorf("[handler]Unmarshal transaction error:%s", error.Error())
		return Transaction{}, error
	}
	transaction := pbToTransaction(t)
	return transaction, nil
}

// 从[]byte反序列化为[]*Transaction
func UnMarshalTransactions(b []byte) ([]*Transaction, error) {
	ts := new(middleware_pb.TransactionSlice)
	error := proto.Unmarshal(b, ts)
	if error != nil {
		logger.Errorf("[handler]Unmarshal transactions error:%s", error.Error())
		return nil, error
	}

	result := PbToTransactions(ts.Transactions)
	return result, nil
}

// 从[]byte反序列化为*Block
func UnMarshalBlock(bytes []byte) (*Block, error) {
	b := new(middleware_pb.Block)
	error := proto.Unmarshal(bytes, b)
	if error != nil {
		logger.Errorf("[handler]Unmarshal Block error:%s", error.Error())
		return nil, error
	}
	block := PbToBlock(b)
	return block, nil
}

// 从[]byte反序列化为*BlockHeader
func UnMarshalBlockHeader(bytes []byte) (*BlockHeader, error) {
	b := new(middleware_pb.BlockHeader)
	error := proto.Unmarshal(bytes, b)
	if error != nil {
		logger.Errorf("[handler]Unmarshal Block error:%s", error.Error())
		return nil, error
	}
	header := PbToBlockHeader(b)
	return header, nil
}

// 从[]byte反序列化为*Member
func UnMarshalMember(b []byte) (*Member, error) {
	member := new(middleware_pb.Member)
	e := proto.Unmarshal(b, member)
	if e != nil {
		logger.Errorf("UnMarshalMember error:%s\n", e.Error())
		return nil, e
	}
	m := pbToMember(member)
	return m, nil
}

// 从[]byte反序列化为*Group
func UnMarshalGroup(b []byte) (*Group, error) {
	group := new(middleware_pb.Group)
	e := proto.Unmarshal(b, group)
	if e != nil {
		logger.Errorf("UnMarshalGroup error:%s\n", e.Error())
		return nil, e
	}
	g := PbToGroup(group)
	return g, nil
}

// 序列化*Transaction
func MarshalTransaction(t *Transaction) ([]byte, error) {
	transaction := transactionToPb(t)
	return proto.Marshal(transaction)
}

// 序列化[]*Transaction
func MarshalTransactions(txs []*Transaction) ([]byte, error) {
	transactions := TransactionsToPb(txs)
	transactionSlice := middleware_pb.TransactionSlice{Transactions: transactions}
	return proto.Marshal(&transactionSlice)
}

// 序列化*Block
func MarshalBlock(b *Block) ([]byte, error) {
	block := BlockToPb(b)
	if block == nil {
		return nil, nil
	}
	return proto.Marshal(block)
}

// 序列化*BlockHeader
func MarshalBlockHeader(b *BlockHeader) ([]byte, error) {
	block := BlockHeaderToPb(b)
	if block == nil {
		return nil, nil
	}
	return proto.Marshal(block)
}

// 序列化*Member
func MarshalMember(m *Member) ([]byte, error) {
	member := memberToPb(m)
	return proto.Marshal(member)
}

// 序列化*Group
func MarshalGroup(g *Group) ([]byte, error) {
	group := GroupToPb(g)
	return proto.Marshal(group)
}

//func MarshalGroupRequest(info *sync.GroupRequestInfo) ([]byte, error) {
//	group := GroupRequestInfoToPB(info)
//	return proto.Marshal(group)
//}

func pbToTransaction(t *middleware_pb.Transaction) Transaction {
	if t == nil {
		return Transaction{}
	}

	var source *common.Address
	var target string
	var sign *common.Sign
	if t.Source != nil {
		s := common.BytesToAddress(t.Source)
		source = &s
	}
	if t.Target != nil {

		target = *t.Target
	}

	if t.Sign != nil && len(t.Sign) != 0 {
		if len(t.Sign) != 65 {
			fmt.Printf("Bad sign hash:%v, sign:%v\n", common.BytesToHash(t.Hash), t.Sign)
		}
		sign = common.BytesToSign(t.Sign)
	}

	transaction := Transaction{Data: t.Data, Nonce: *t.Nonce, Source: source,
		Target: target, Hash: common.BytesToHash(t.Hash),
		ExtraData: t.ExtraData, ExtraDataType: *t.ExtraDataType, Type: *t.Type, Sign: sign}
	return transaction
}

func PbToTransactions(txs []*middleware_pb.Transaction) []*Transaction {
	result := make([]*Transaction, 0)
	if txs == nil {
		return result
	}
	for _, t := range txs {
		transaction := pbToTransaction(t)
		result = append(result, &transaction)
	}
	return result
}

func PbToBlockHeader(h *middleware_pb.BlockHeader) *BlockHeader {
	if h == nil {
		return nil
	}
	hashBytes := h.Transactions
	hashes := make([]common.Hash, 0, len(hashBytes.Hashes))

	if hashBytes != nil {
		for _, hashByte := range hashBytes.Hashes {
			hash := common.BytesToHash(hashByte)
			hashes = append(hashes, hash)
		}
	}

	hashBytes2 := h.EvictedTxs
	hashes2 := make([]common.Hash, 0)

	if hashBytes2 != nil {
		for _, hashByte := range hashBytes2.Hashes {
			hash := common.BytesToHash(hashByte)
			hashes2 = append(hashes2, hash)
		}
	}

	var preTime time.Time
	e1 := preTime.UnmarshalBinary(h.PreTime)
	if e1 != nil {
		logger.Errorf("[handler]pbToBlockHeader preTime UnmarshalBinary error:%s", e1.Error())
		return nil
	}

	var curTime time.Time
	curTime.UnmarshalBinary(h.CurTime)
	e2 := curTime.UnmarshalBinary(h.CurTime)
	if e2 != nil {
		logger.Errorf("[handler]pbToBlockHeader curTime UnmarshalBinary error:%s", e2.Error())
		return nil
	}

	pv := &big.Int{}
	var proveValue *big.Int
	if h.ProveValue != nil {
		proveValue = pv.SetBytes(h.ProveValue)
	} else {
		proveValue = nil
	}
	//log.Printf("PbToBlockHeader height:%d StateTree Hash:%s",*h.Height,common.Bytes2Hex(h.StateTree))
	header := BlockHeader{Hash: common.BytesToHash(h.Hash), Height: *h.Height, PreHash: common.BytesToHash(h.PreHash), PreTime: preTime,
		ProveValue: proveValue, CurTime: curTime, Castor: h.Castor, GroupId: h.GroupId, Signature: h.Signature,
		Nonce: *h.Nonce, Transactions: hashes, TxTree: common.BytesToHash(h.TxTree), ReceiptTree: common.BytesToHash(h.ReceiptTree), StateTree: common.BytesToHash(h.StateTree),
		ExtraData: h.ExtraData, TotalQN: *h.TotalQN, Random: h.Random, ProveRoot: common.BytesToHash(h.ProveRoot), EvictedTxs: hashes2}
	return &header
}

func GroupRequestInfoToPB(CurrentTopGroupId []byte, ExistGroupIds [][]byte) *middleware_pb.GroupRequestInfo {
	return &middleware_pb.GroupRequestInfo{CurrentTopGroupId: CurrentTopGroupId, ExistGroupIds: &middleware_pb.GroupIdSlice{GroupIds: ExistGroupIds}}
}

func PbToBlock(b *middleware_pb.Block) *Block {
	if b == nil {
		return nil
	}
	h := PbToBlockHeader(b.Header)
	txs := PbToTransactions(b.Transactions)
	block := Block{Header: h, Transactions: txs}
	return &block
}

func PbToGroupHeader(g *middleware_pb.GroupHeader) *GroupHeader {
	var beginTime time.Time
	beginTime.UnmarshalBinary(g.BeginTime)
	header := GroupHeader{
		Hash:          common.BytesToHash(g.Hash),
		Parent:        g.Parent,
		PreGroup:      g.PreGroup,
		Authority:     *g.Authority,
		Name:          *g.Name,
		BeginTime:     beginTime,
		MemberRoot:    common.BytesToHash(g.MemberRoot),
		CreateHeight:  *g.CreateHeight,
		ReadyHeight:   *g.ReadyHeight,
		WorkHeight:    *g.WorkHeight,
		DismissHeight: *g.DismissHeight,
		Extends:       *g.Extends,
	}
	return &header
}

func PbToGroup(g *middleware_pb.Group) *Group {
	//members := make([]Member, 0)
	//for _, m := range g.Members {
	//	member := pbToMember(m)
	//	members = append(members, *member)
	//}
	group := Group{
		Header:    PbToGroupHeader(g.Header),
		Id:        g.Id,
		Members:   g.Members,
		PubKey:    g.PubKey,
		Signature: g.Signature,
	}
	return &group
}

func PbToGroups(g *middleware_pb.GroupSlice) []*Group {
	result := make([]*Group, 0)
	for _, group := range g.Groups {
		result = append(result, PbToGroup(group))
	}
	return result
}

func pbToMember(m *middleware_pb.Member) *Member {
	member := Member{Id: m.Id, PubKey: m.PubKey}
	return &member
}

func transactionToPb(t *Transaction) *middleware_pb.Transaction {
	if t == nil {
		return nil
	}
	var (
		target *string
		source []byte
		sign   []byte
	)
	if len(t.Target) != 0 {
		target = &t.Target
	}
	if t.Source != nil {
		source = t.Source.Bytes()
	}

	if t.Sign != nil {
		sign = t.Sign.Bytes()
		if len(sign) != 65 {
			logger.Errorf("Bad sign len:%d", len(sign))
		}
	}
	//achates add for testing<<
	if len(sign) != 65 {
		fmt.Println("Bad sign in transactionToPb sign=", sign)
	}
	//>>achates add for testing
	transaction := middleware_pb.Transaction{Data: t.Data, Nonce: &t.Nonce, Source: source,
		Target: target,  Hash: t.Hash.Bytes(),
		ExtraData: t.ExtraData, ExtraDataType: &t.ExtraDataType, Type: &t.Type, Sign: sign}
	return &transaction
}

func TransactionsToPb(txs []*Transaction) []*middleware_pb.Transaction {
	if txs == nil {
		return nil
	}
	transactions := make([]*middleware_pb.Transaction, 0)
	for _, t := range txs {
		transaction := transactionToPb(t)
		transactions = append(transactions, transaction)
	}
	return transactions
}

func BlockHeaderToPb(h *BlockHeader) *middleware_pb.BlockHeader {
	hashes := h.Transactions
	hashBytes := make([][]byte, 0)

	if hashes != nil {
		for _, hash := range hashes {
			hashBytes = append(hashBytes, hash.Bytes())
		}
	}
	txHashes := middleware_pb.Hashes{Hashes: hashBytes}
	hashes2 := h.EvictedTxs
	hashBytes2 := make([][]byte, 0)

	if hashes2 != nil {
		for _, hash := range hashes2 {
			hashBytes2 = append(hashBytes2, hash.Bytes())
		}
	}
	evictedTxs := middleware_pb.Hashes{Hashes: hashBytes2}
	preTime, e1 := h.PreTime.MarshalBinary()
	if e1 != nil {
		logger.Errorf("BlockHeaderToPb marshal pre time error:%s\n", e1.Error())
		return nil
	}

	curTime, e2 := h.CurTime.MarshalBinary()
	if e2 != nil {
		logger.Errorf("BlockHeaderToPb marshal cur time error:%s", e2.Error())
		return nil
	}

	var proveValueByte []byte
	if h.ProveValue != nil {
		proveValueByte = h.ProveValue.Bytes()
	} else {
		proveValueByte = nil
	}

	header := middleware_pb.BlockHeader{Hash: h.Hash.Bytes(), Height: &h.Height, PreHash: h.PreHash.Bytes(), PreTime: preTime,
		ProveValue: proveValueByte, CurTime: curTime, Castor: h.Castor, GroupId: h.GroupId, Signature: h.Signature,
		Nonce: &h.Nonce, Transactions: &txHashes, TxTree: h.TxTree.Bytes(), ReceiptTree: h.ReceiptTree.Bytes(), StateTree: h.StateTree.Bytes(),
		ExtraData: h.ExtraData, TotalQN: &h.TotalQN, Random: h.Random, ProveRoot: h.ProveRoot.Bytes(), EvictedTxs: &evictedTxs}
	return &header
}

func BlockToPb(b *Block) *middleware_pb.Block {
	if b == nil {
		return nil
	}
	header := BlockHeaderToPb(b.Header)
	transactions := TransactionsToPb(b.Transactions)
	block := middleware_pb.Block{Header: header, Transactions: transactions}
	return &block
}

func GroupToPbHeader(g *GroupHeader) *middleware_pb.GroupHeader {
	beginTime, _ := g.BeginTime.MarshalBinary()
	header := middleware_pb.GroupHeader{
		Hash:          g.Hash.Bytes(),
		Parent:        g.Parent,
		PreGroup:      g.PreGroup,
		Authority:     &g.Authority,
		Name:          &g.Name,
		BeginTime:     beginTime,
		MemberRoot:    g.MemberRoot.Bytes(),
		CreateHeight:  &g.CreateHeight,
		ReadyHeight:   &g.ReadyHeight,
		WorkHeight:    &g.WorkHeight,
		DismissHeight: &g.DismissHeight,
		Extends:       &g.Extends,
	}
	return &header
}

func GroupToPb(g *Group) *middleware_pb.Group {
	//members := make([]*middleware_pb.Member, 0)
	//for _, m := range g.Members {
	//	member := memberToPb(&m)
	//	members = append(members, member)
	//}
	group := middleware_pb.Group{
		Header:    GroupToPbHeader(g.Header),
		Id:        g.Id,
		Members:   g.Members,
		PubKey:    g.PubKey,
		Signature: g.Signature,
	}
	return &group
}

func memberToPb(m *Member) *middleware_pb.Member {
	member := middleware_pb.Member{Id: m.Id, PubKey: m.PubKey}
	return &member
}

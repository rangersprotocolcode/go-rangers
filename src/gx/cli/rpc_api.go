package cli

import (
	"x/src/common"
	"x/src/consensus/groupsig"
	"x/src/core"
	"encoding/json"
	"x/src/middleware/types"
	"encoding/hex"
	"math"
	"x/src/consensus"
	"sync"
	"x/src/middleware/log"
	"x/src/statemachine"
	"x/src/service"
)

func successResult(data interface{}) (*Result, error) {
	return &Result{
		Message: "success",
		Data:    data,
		Status:  0,
	}, nil
}
func failResult(err string) (*Result, error) {
	return &Result{
		Message: err,
		Data:    nil,
		Status:  -1,
	}, nil
}

// GtasAPI is a single-method API handler to be returned by test services.
type GtasAPI struct {
	logger log.Logger
}

var gxLock *sync.RWMutex

// NewWallet 新建账户接口
func (api *GtasAPI) NewWallet() (*Result, error) {
	privKey, addr := walletManager.newWallet()
	data := make(map[string]string)
	data["private_key"] = privKey
	data["address"] = addr
	return successResult(data)
}

// GetWallets 获取当前节点的wallets
func (api *GtasAPI) GetWallets() (*Result, error) {
	return successResult(walletManager)
}

// DeleteWallet 删除本地节点指定序号的地址
func (api *GtasAPI) DeleteWallet(key string) (*Result, error) {
	walletManager.deleteWallet(key)
	return successResult(walletManager)
}

// BlockHeight 块高查询
func (api *GtasAPI) BlockHeight() (*Result, error) {
	height := core.GetBlockChain().TopBlock().Height
	return successResult(height)
}

// GroupHeight 组块高查询
func (api *GtasAPI) GroupHeight() (*Result, error) {
	height := core.GetGroupChain().Count()
	return successResult(height)
}

// TransPool 查询缓冲区的交易信息。
func (api *GtasAPI) TransPool() (*Result, error) {
	transactions := service.GetTransactionPool().GetReceived()
	transList := make([]Transactions, 0, len(transactions))
	for _, v := range transactions {
		transList = append(transList, Transactions{
			Hash:   v.Hash.String(),
			Source: v.Source,
			Target: v.Target,
		})
	}

	return successResult(transList)
}

// STMStatus 查询STM状态
func (api *GtasAPI) StmStatus() (*Result, error) {
	result := statemachine.STMManger.GetStmStatus()
	return successResult(result)
}

func (api *GtasAPI) GetTransaction(hash string) (*Result, error) {
	transaction, err := core.GetBlockChain().GetTransaction(common.HexToHash(hash))
	if err != nil {
		return failResult(err.Error())
	}

	return successResult(*convertTransaction(transaction))
}

//
//func convertBlock(bh *types.BlockHeader) interface{} {
//	blockDetail := make(map[string]interface{})
//	blockDetail["hash"] = bh.Hash.Hex()
//	blockDetail["height"] = bh.Height
//	blockDetail["pre_hash"] = bh.PreHash.Hex()
//	blockDetail["pre_time"] = bh.PreTime.Format("2006-01-02 15:04:05")
//	blockDetail["queue_number"] = bh.ProveValue
//	blockDetail["cur_time"] = bh.CurTime.Format("2006-01-02 15:04:05")
//	var castorId groupsig.ID
//	castorId.Deserialize(bh.Castor)
//	blockDetail["castor"] = castorId.String()
//	//blockDetail["castor"] = hex.EncodeToString(bh.Castor)
//	var gid groupsig.ID
//	gid.Deserialize(bh.GroupId)
//	blockDetail["group_id"] = gid.GetHexString()
//	blockDetail["signature"] = hex.EncodeToString(bh.Signature)
//	trans := make([]string, len(bh.Transactions))
//	for i := range bh.Transactions {
//		trans[i] = bh.Transactions[i].String()
//	}
//	blockDetail["transactions"] = trans
//	blockDetail["txs"] = len(bh.Transactions)
//	blockDetail["total_qn"] = bh.TotalQN
//	blockDetail["qn"] = mediator.Proc.CalcBlockHeaderQN(bh)
//	blockDetail["tps"] = math.Round(float64(len(bh.Transactions)) / bh.CurTime.Sub(bh.PreTime).Seconds())
//	return blockDetail
//}

func (api *GtasAPI) GetBlockByHeight(height uint64) (*Result, error) {
	b := core.GetBlockChain().QueryBlock(height)
	if b == nil {
		return failResult("height not exists")
	}
	preBlock := core.GetBlockChain().QueryBlockByHash(b.Header.PreHash)
	block := convertBlockHeader(b.Header)
	if preBlock != nil {
		block.Qn = b.Header.TotalQN - preBlock.Header.TotalQN
	} else {
		block.Qn = b.Header.TotalQN
	}
	return successResult(block)
}

func (api *GtasAPI) GetBlockByHash(hash string) (*Result, error) {
	b := core.GetBlockChain().QueryBlockByHash(common.HexToHash(hash))
	if b == nil {
		return failResult("height not exists")
	}
	bh := b.Header
	preBlock := core.GetBlockChain().QueryBlockByHash(bh.PreHash)
	preBH := preBlock.Header
	block := convertBlockHeader(bh)
	if preBH != nil {
		block.Qn = bh.TotalQN - preBH.TotalQN
	} else {
		block.Qn = bh.TotalQN
	}
	return successResult(block)
}

func (api *GtasAPI) GetCurrentBlock() (*Result, error) {
	b := core.GetBlockChain().CurrentBlock()
	if b == nil {
		return failResult("layer2 error")
	}
	bh := b.Header
	preBlock := core.GetBlockChain().QueryBlockByHash(bh.PreHash)
	preBH := preBlock.Header
	block := convertBlockHeader(bh)
	if preBH != nil {
		block.Qn = bh.TotalQN - preBH.TotalQN
	} else {
		block.Qn = bh.TotalQN
	}
	return successResult(block)
}

func (api *GtasAPI) GetBlocks(from uint64, to uint64) (*Result, error) {
	blocks := make([]*Block, 0)
	var preBH *types.BlockHeader
	for h := from; h <= to; h++ {
		b := core.GetBlockChain().QueryBlock(h)
		if b != nil {
			block := convertBlockHeader(b.Header)
			if preBH == nil {
				preBH = core.GetBlockChain().QueryBlockByHash(b.Header.PreHash).Header
			}
			block.Qn = b.Header.TotalQN - preBH.TotalQN
			preBH = b.Header
			blocks = append(blocks, block)
		}
	}
	return successResult(blocks)
}

func (api *GtasAPI) BlockDetail(h string) (*Result, error) {
	chain := core.GetBlockChain()
	b := chain.QueryBlockByHash(common.HexToHash(h))
	if b == nil {
		return successResult(nil)
	}
	bh := b.Header
	block := convertBlockHeader(bh)

	preBH := chain.QueryBlockByHash(bh.PreHash).Header
	block.Qn = bh.TotalQN - preBH.TotalQN

	trans := make([]Transaction, 0)
	for _, tx := range b.Transactions {
		trans = append(trans, *convertTransaction(tx))
	}

	bd := &BlockDetail{
		Block: *block,
		Trans: trans,
	}
	return successResult(bd)
}

//deprecated
func (api *GtasAPI) GetTopBlock() (*Result, error) {
	bh := core.GetBlockChain().TopBlock()
	blockDetail := make(map[string]interface{})
	blockDetail["hash"] = bh.Hash.Hex()
	blockDetail["height"] = bh.Height
	blockDetail["pre_hash"] = bh.PreHash.Hex()
	blockDetail["pre_time"] = bh.PreTime.Format("2006-01-02 15:04:05")
	blockDetail["queue_number"] = bh.ProveValue
	blockDetail["cur_time"] = bh.CurTime.Format("2006-01-02 15:04:05")
	blockDetail["castor"] = hex.EncodeToString(bh.Castor)
	blockDetail["group_id"] = hex.EncodeToString(bh.GroupId)
	blockDetail["signature"] = hex.EncodeToString(bh.Signature)
	blockDetail["txs"] = len(bh.Transactions)
	blockDetail["tps"] = math.Round(float64(len(bh.Transactions)) / bh.CurTime.Sub(bh.PreTime).Seconds())

	blockDetail["tx_pool_count"] = len(service.GetTransactionPool().GetReceived())
	blockDetail["tx_pool_total"] = service.GetTransactionPool().TxNum()
	blockDetail["miner_id"] = consensus.Proc.GetMinerID().ShortS()
	return successResult(blockDetail)
}

func (api *GtasAPI) WorkGroupNum(height uint64) (*Result, error) {
	groups := consensus.Proc.GetCastQualifiedGroups(height)
	return successResult(groups)
}

func convertGroup(g *types.Group) map[string]interface{} {
	gmap := make(map[string]interface{})
	if g.Id != nil && len(g.Id) != 0 {
		gmap["group_id"] = groupsig.DeserializeId(g.Id).GetHexString()
		gmap["g_hash"] = g.Header.Hash.String()
	}
	gmap["parent"] = groupsig.DeserializeId(g.Header.Parent).GetHexString()
	gmap["pre"] = groupsig.DeserializeId(g.Header.PreGroup).GetHexString()
	gmap["begin_height"] = g.Header.WorkHeight
	gmap["dismiss_height"] = g.Header.DismissHeight
	gmap["create_height"] = g.Header.CreateHeight
	gmap["create_time"] = g.Header.BeginTime
	gmap["mem_size"] = len(g.Members)
	mems := make([]string, 0)
	for _, mem := range g.Members {
		memberStr := groupsig.DeserializeId(mem).GetHexString()
		mems = append(mems, memberStr[0:6]+"-"+memberStr[len(memberStr)-6:])
	}
	gmap["members"] = mems
	gmap["extends"] = g.Header.Extends
	return gmap
}

func (api *GtasAPI) GetGroupsAfter(height uint64) (*Result, error) {
	count := core.GetGroupChain().Count()
	if count <= height {
		return failResult("exceed local height")
	}
	groups := make([]*types.Group, count-height)
	for i := height; i < count; i++ {
		group := core.GetGroupChain().GetGroupByHeight(i)
		if nil != group {
			groups[i-height] = group
		}

	}

	ret := make([]map[string]interface{}, 0)
	h := height
	for _, g := range groups {
		gmap := convertGroup(g)
		gmap["height"] = h
		h++
		ret = append(ret, gmap)
	}
	return successResult(ret)
}

func (api *GtasAPI) GetCurrentWorkGroup() (*Result, error) {
	height := core.GetBlockChain().Height()
	return api.GetWorkGroup(height)
}

func (api *GtasAPI) GetWorkGroup(height uint64) (*Result, error) {
	groups := consensus.Proc.GetCastQualifiedGroupsFromChain(height)
	ret := make([]map[string]interface{}, 0)
	h := height
	for _, g := range groups {
		gmap := convertGroup(g)
		gmap["height"] = h
		h++
		ret = append(ret, gmap)
	}
	return successResult(ret)
}

func (api *GtasAPI) MinerQuery(mtype int32) (*Result, error) {
	minerInfo := consensus.Proc.GetMinerInfo()
	address := common.BytesToAddress(minerInfo.ID.Serialize())
	miner := core.MinerManagerImpl.GetMinerById(address[:], byte(mtype), nil)
	js, err := json.Marshal(miner)
	if err != nil {
		return &Result{Message: err.Error(), Data: nil}, err
	}
	return &Result{Message: address.GetHexString(), Data: string(js)}, nil
}

//铸块统计
func (api *GtasAPI) CastStat(begin uint64, end uint64) (*Result, error) {
	proposerStat := make(map[string]int32)
	groupStat := make(map[string]int32)

	chain := core.GetBlockChain()
	if end == 0 {
		end = chain.TopBlock().Height
	}

	for h := begin; h < end; h++ {
		b := chain.QueryBlock(h)
		if b == nil {
			continue
		}
		p := string(b.Header.Castor)
		if v, ok := proposerStat[p]; ok {
			proposerStat[p] = v + 1
		} else {
			proposerStat[p] = 1
		}
		g := string(b.Header.GroupId)
		if v, ok := groupStat[g]; ok {
			groupStat[g] = v + 1
		} else {
			groupStat[g] = 1
		}
	}
	pmap := make(map[string]int32)
	gmap := make(map[string]int32)

	for key, v := range proposerStat {
		id := groupsig.DeserializeId([]byte(key))
		pmap[id.GetHexString()] = v
	}
	for key, v := range groupStat {
		id := groupsig.DeserializeId([]byte(key))
		gmap[id.GetHexString()] = v
	}
	ret := make(map[string]map[string]int32)
	ret["proposer"] = pmap
	ret["group"] = gmap
	return successResult(ret)
}

func (api *GtasAPI) MinerInfo(addr string) (*Result, error) {
	morts := make([]MortGage, 0)
	id := common.HexToAddress(addr).Bytes()
	heavyInfo := core.MinerManagerImpl.GetMinerById(id, types.MinerTypeHeavy, nil)
	if heavyInfo != nil {
		morts = append(morts, *NewMortGageFromMiner(heavyInfo))
	}
	lightInfo := core.MinerManagerImpl.GetMinerById(id, types.MinerTypeLight, nil)
	if lightInfo != nil {
		morts = append(morts, *NewMortGageFromMiner(lightInfo))
	}
	return successResult(morts)
}

func (api *GtasAPI) NodeInfo() (*Result, error) {
	ni := &NodeInfo{}
	p := consensus.Proc
	ni.ID = p.GetMinerID().GetHexString()
	balance, err := walletManager.getBalance(p.GetMinerID().GetHexString())
	if err != nil {
		return failResult(err.Error())
	}
	ni.Balance = float64(balance) / float64(1000000000)
	if !p.Ready() {
		ni.Status = "节点未准备就绪"
	} else {
		ni.Status = "运行中"
		morts := make([]MortGage, 0)
		t := "--"
		heavyInfo := core.MinerManagerImpl.GetMinerById(p.GetMinerID().Serialize(), types.MinerTypeHeavy, nil)
		if heavyInfo != nil {
			morts = append(morts, *NewMortGageFromMiner(heavyInfo))
			if heavyInfo.AbortHeight == 0 {
				t = "重节点"
			}
		}
		lightInfo := core.MinerManagerImpl.GetMinerById(p.GetMinerID().Serialize(), types.MinerTypeLight, nil)
		if lightInfo != nil {
			morts = append(morts, *NewMortGageFromMiner(lightInfo))
			if lightInfo.AbortHeight == 0 {
				t += " 轻节点"
			}
		}
		ni.NType = t
		ni.MortGages = morts

		wg, ag := p.GetJoinedWorkGroupNums()
		ni.WGroupNum = wg
		ni.AGroupNum = ag

		if txs := service.GetTransactionPool().GetReceived(); txs != nil {
			ni.TxPoolNum = len(txs)
		}
	}
	return successResult(ni)

}

func (api *GtasAPI) PageGetBlocks(page, limit int) (*Result, error) {
	chain := core.GetBlockChain()
	total := chain.Height() + 1
	pageObject := PageObjects{
		Total: total,
		Data:  make([]interface{}, 0),
	}
	if page < 1 {
		page = 1
	}
	i := 0
	num := uint64((page - 1) * limit)
	if total < num {
		return successResult(pageObject)
	}
	b := int64(total - num)

	for i < limit && b >= 0 {
		block := chain.QueryBlock(uint64(b))
		b--
		if block == nil {
			continue
		}
		h := convertBlockHeader(block.Header)
		pageObject.Data = append(pageObject.Data, h)
		i++
	}
	return successResult(pageObject)
}

func (api *GtasAPI) PageGetGroups(page, limit int) (*Result, error) {
	chain := core.GetGroupChain()
	total := chain.Count()
	pageObject := PageObjects{
		Total: total,
		Data:  make([]interface{}, 0),
	}

	i := 0
	b := int64(0)
	if page < 1 {
		page = 1
	}
	num := uint64((page - 1) * limit)
	if total < num {
		return successResult(pageObject)
	}
	b = int64(total - num)

	for i < limit && b >= 0 {
		g := chain.GetGroupByHeight(uint64(b))
		b--
		if g == nil {
			continue
		}

		mems := make([]string, 0)
		for _, mem := range g.Members {
			mems = append(mems, groupsig.DeserializeId(mem).ShortS())
		}

		group := &Group{
			Height:        uint64(b + 1),
			Id:            groupsig.DeserializeId(g.Id),
			PreId:         groupsig.DeserializeId(g.Header.PreGroup),
			ParentId:      groupsig.DeserializeId(g.Header.Parent),
			BeginHeight:   g.Header.WorkHeight,
			DismissHeight: g.Header.DismissHeight,
			Members:       mems,
		}
		pageObject.Data = append(pageObject.Data, group)
		i++
	}
	return successResult(pageObject)
}

func (api *GtasAPI) BlockReceipts(h string) (*Result, error) {
	chain := core.GetBlockChain()
	b := chain.QueryBlockByHash(common.HexToHash(h))
	if b == nil {
		return failResult("block not found")
	}
	bh := b.Header

	evictedReceipts := make([]*types.Receipt, 0)
	for _, tx := range bh.EvictedTxs {
		wrapper := service.GetTransactionPool().GetExecuted(tx)
		if wrapper != nil {
			evictedReceipts = append(evictedReceipts, wrapper.Receipt)
		}
	}
	receipts := make([]*types.Receipt, len(bh.Transactions))
	for i, tx := range bh.Transactions {
		wrapper := service.GetTransactionPool().GetExecuted(tx[0])
		if wrapper != nil {
			receipts[i] = wrapper.Receipt
		}
	}
	br := &BlockReceipt{EvictedReceipts: evictedReceipts, Receipts: receipts}
	return successResult(br)
}

func (api *GtasAPI) TransDetail(h string) (*Result, error) {
	tx, err := core.GetBlockChain().GetTransaction(common.HexToHash(h))
	if err != nil {
		return failResult(err.Error())
	}
	if tx != nil {
		trans := convertTransaction(tx)
		return successResult(trans)
	}
	return successResult(nil)
}

func (api *GtasAPI) Dashboard() (*Result, error) {
	blockHeight := core.GetBlockChain().Height()
	groupHeight := core.GetGroupChain().Count()
	workNum := len(consensus.Proc.GetCastQualifiedGroups(blockHeight))
	nodeResult, _ := api.NodeInfo()

	dash := &Dashboard{
		BlockHeight: blockHeight,
		GroupHeight: groupHeight,
		WorkGNum:    workNum,
		NodeInfo:    nodeResult.Data.(*NodeInfo),
	}
	return successResult(dash)
}

//func bonusStatByHeight(height uint64) BonusInfo {
//	bh := core.BlockChainImpl.QueryBlockByHeight(height)
//	casterId := bh.Castor
//	groupId := bh.GroupId
//
//	bonusTx := core.BlockChainImpl.GetBonusManager().GetBonusTransactionByBlockHash(bh.Hash.Bytes())
//	if bonusTx == nil {
//		return BonusInfo{}
//	}
//
//	// 从交易信息中解析出targetId列表
//	_, memIds, _, value := mediator.Proc.MainChain.GetBonusManager().ParseBonusTransaction(bonusTx)
//
//	mems := make([]string, 0)
//	for _, memId := range memIds {
//		mems = append(mems, groupsig.DeserializeId(memId).ShortS())
//	}
//
//	data := BonusInfo{
//		BlockHeight: height,
//		BlockHash:   bh.Hash,
//		BonusTxHash: bonusTx.Hash,
//		GroupId:     groupsig.DeserializeId(groupId).ShortS(),
//		CasterId:    groupsig.DeserializeId(casterId).ShortS(),
//		GroupIdW:    groupsig.DeserializeId(groupId).GetHexString(),
//		CasterIdW:   groupsig.DeserializeId(casterId).GetHexString(),
//		MemberIds:   mems,
//		BonusValue:  value,
//	}
//
//	return data
//}

func (api *GtasAPI) TxReceipt(h string) (*Result, error) {
	w := service.GetTransactionPool().GetExecuted(common.HexToHash(h))
	return successResult(w)
}

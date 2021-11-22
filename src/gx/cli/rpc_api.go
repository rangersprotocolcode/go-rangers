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

package cli

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/core"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"encoding/hex"
	"fmt"
	"math"
	"sync"
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
	privateKey string
	logger     log.Logger
}

var gxLock *sync.RWMutex

// NewWallet 新建账户接口
func (api *GtasAPI) NewWallet() (*Result, error) {
	privKey, addr, miner := walletManager.newWallet()
	data := make(map[string]string)
	data["private_key"] = privKey
	data["address"] = addr
	data["miner"] = miner

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

func (api *GtasAPI) GetTransaction(hash string) (*Result, error) {
	transaction, err := core.GetBlockChain().GetTransaction(common.HexToHash(hash))
	if err != nil {
		return failResult(err.Error())
	}

	return successResult(*convertTransaction(transaction))
}

func (api *GtasAPI) GetExecutedTransaction(hash string) (*Result, error) {
	executed := service.GetTransactionPool().GetExecuted(common.HexToHash(hash))
	if executed == nil {
		return failResult(fmt.Sprintf("%s not existed", hash))
	}

	tx, err := types.UnMarshalTransaction(executed.Transaction)
	if err != nil {
		return failResult(err.Error())
	}

	result := make(map[string]interface{}, 0)
	result["tx"] = convertTransaction(&tx)
	result["receipt"] = executed.Receipt
	return successResult(result)
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
	b := core.GetBlockChain().TopBlock()
	if b == nil {
		return failResult("layer2 error")
	}
	preBlock := core.GetBlockChain().QueryBlockByHash(b.PreHash)
	preBH := preBlock.Header
	block := convertBlockHeader(b)
	if preBH != nil {
		block.Qn = b.TotalQN - preBH.TotalQN
	} else {
		block.Qn = b.TotalQN
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
		id := groupsig.DeserializeID([]byte(key))
		pmap[id.GetHexString()] = v
	}
	for key, v := range groupStat {
		id := groupsig.DeserializeID([]byte(key))
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
	heavyInfo := service.MinerManagerImpl.GetMinerById(id, common.MinerTypeProposer, nil)
	if heavyInfo != nil {
		morts = append(morts, *NewMortGageFromMiner(heavyInfo))
	}
	lightInfo := service.MinerManagerImpl.GetMinerById(id, common.MinerTypeValidator, nil)
	if lightInfo != nil {
		morts = append(morts, *NewMortGageFromMiner(lightInfo))
	}
	return successResult(morts)
}

func (api *GtasAPI) NodeInfo() (*Result, error) {
	ni := &NodeInfo{}
	p := consensus.Proc
	ni.ID = p.GetMinerID().GetHexString()

	if !p.Ready() {
		ni.Status = "节点未准备就绪"
	} else {
		ni.Status = "运行中"
		morts := make([]MortGage, 0)
		t := ""
		balance := ""
		heavyInfo := service.MinerManagerImpl.GetMinerById(p.GetMinerID().Serialize(), common.MinerTypeProposer, service.AccountDBManagerInstance.GetLatestStateDB())
		if heavyInfo != nil {
			morts = append(morts, *NewMortGageFromMiner(heavyInfo))
			t = "提案节点"
			balance = walletManager.getBalance(heavyInfo.Account)
		}

		lightInfo := service.MinerManagerImpl.GetMinerById(p.GetMinerID().Serialize(), common.MinerTypeValidator, service.AccountDBManagerInstance.GetLatestStateDB())
		if lightInfo != nil {
			morts = append(morts, *NewMortGageFromMiner(lightInfo))
			t = " 验证节点"
			balance = walletManager.getBalance(lightInfo.Account)
		}

		ni.Balance = balance
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
	_, addr, self := walletManager.newWalletByPrivateKey(api.privateKey)

	dash := &Dashboard{
		BlockHeight: blockHeight,
		GroupHeight: groupHeight,
		WorkGNum:    workNum,
		NodeInfo:    nodeResult.Data.(*NodeInfo),
		Miner:       self,
		Addr:        addr,
	}
	return successResult(dash)
}

func (api *GtasAPI) TxReceipt(h string) (*Result, error) {
	w := service.GetTransactionPool().GetExecuted(common.HexToHash(h))
	return successResult(w)
}

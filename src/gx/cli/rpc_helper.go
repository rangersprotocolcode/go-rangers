package cli

import (
	"x/src/middleware/types"
	"x/src/consensus/groupsig"
	"x/src/common"
	"x/src/core"
	"fmt"
	"x/src/consensus"
)

func convertTransaction(tx *types.Transaction) *Transaction {
	trans := &Transaction{
		Hash:          tx.Hash,
		Source:        tx.Source,
		Target:        tx.Target,
		Type:          tx.Type,
		Data:          tx.Data,
		ExtraData:     tx.ExtraData,
		ExtraDataType: tx.ExtraDataType,
		Nonce:         tx.Nonce,

	}
	return trans
}

func convertBlockHeader(bh *types.BlockHeader) *Block {
	block := &Block{
		Height:  bh.Height,
		Hash:    bh.Hash,
		PreHash: bh.PreHash,
		CurTime: bh.CurTime,
		PreTime: bh.PreTime,
		Castor:  groupsig.DeserializeId(bh.Castor),
		GroupID: groupsig.DeserializeId(bh.GroupId),
		Prove:   bh.ProveValue,
		Txs:     bh.Transactions,
		TotalQN: bh.TotalQN,
		//Qn: mediator.Proc.CalcBlockHeaderQN(bh),
		StateRoot:   bh.StateTree,
		TxRoot:      bh.TxTree,
		ReceiptRoot: bh.ReceiptTree,
		ProveRoot:   bh.ProveRoot,
		Random:      common.ToHex(bh.Random),
		TxNum:       uint64(len(bh.Transactions)),
	}
	return block
}

func convertBonusTransaction(tx *types.Transaction) *BonusTransaction {
	if tx.Type != types.TransactionTypeBonus {
		return nil
	}
	gid, ids, bhash, value := consensus.Proc.MainChain.GetBonusManager().ParseBonusTransaction(tx)

	targets := make([]groupsig.ID, len(ids))
	for i, id := range ids {
		targets[i] = groupsig.DeserializeId(id)
	}
	return &BonusTransaction{
		Hash:      tx.Hash,
		BlockHash: bhash,
		GroupID:   groupsig.DeserializeId(gid),
		TargetIDs: targets,
		Value:     value,
	}
}

func genMinerBalance(id groupsig.ID, bh *types.BlockHeader) *MinerBonusBalance {
	mb := &MinerBonusBalance{
		ID: id,
	}
	db, err := consensus.Proc.MainChain.GetAccountDBByHash(bh.Hash)
	if err != nil {
		common.DefaultLogger.Errorf("GetAccountDBByHash err %v, hash %v", err, bh.Hash)
		return mb
	}
	mb.CurrBalance = db.GetBalance(id.ToAddress())
	preDB, err := consensus.Proc.MainChain.GetAccountDBByHash(bh.PreHash)
	if err != nil {
		common.DefaultLogger.Errorf("GetAccountDBByHash err %v hash %v", err, bh.PreHash)
		return mb
	}
	mb.PreBalance = preDB.GetBalance(id.ToAddress())
	return mb
}



func sendTransaction(trans *types.Transaction) error {
	if trans.Sign == nil {
		return fmt.Errorf("transaction sign is empty")
	}
	pk, err := trans.Sign.RecoverPubkey(trans.Hash.Bytes())
	if err != nil {
		return err
	}
	source := pk.GetAddress()
	trans.Source = &source
	//common.DefaultLogger.Debugf(trans.Sign.GetHexString(), pk.GetHexString(), source.GetHexString(), trans.Hash.String())

	if ok, err := core.GetBlockChain().GetTransactionPool().AddTransaction(trans); err != nil || !ok {
		common.DefaultLogger.Errorf("AddTransaction not ok or error:%s", err.Error())
		return err
	}
	return nil
}

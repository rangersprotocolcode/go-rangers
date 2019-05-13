package cli

import (
	"x/src/middleware/types"
	"x/src/consensus/groupsig"
	"x/src/common"
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




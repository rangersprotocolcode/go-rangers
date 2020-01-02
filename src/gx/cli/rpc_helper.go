package cli

import (
	"x/src/middleware/types"
	"x/src/consensus/groupsig"
	"x/src/common"
	"encoding/json"
	"fmt"
)

func convertTransaction(tx *types.Transaction) *Transaction {
	trans := &Transaction{
		Hash:      tx.Hash,
		Source:    tx.Source,
		Target:    tx.Target,
		Type:      tx.Type,
		Data:      tx.Data,
		ExtraData: tx.ExtraData,
	}

	if tx.Sign != nil {
		fmt.Println("tx.Sign != nil")
		fmt.Println(tx.Sign.Bytes())
		fmt.Println(tx.Sign.GetR())
		fmt.Println(tx.Sign.GetS())
		trans.Signature = tx.Sign.GetHexString()
	} else {
		fmt.Println("tx.Sign = nil")
	}

	data, err := json.Marshal(tx.SubTransactions)
	if err == nil {
		trans.SubTransactions = string(data)
	}

	return trans
}

func convertBlockHeader(bh *types.BlockHeader) *Block {
	block := &Block{
		Version:     bh.Nonce,
		Height:      bh.Height,
		Hash:        bh.Hash,
		PreHash:     bh.PreHash,
		CurTime:     bh.CurTime.String(),
		PreTime:     bh.PreTime.String(),
		Castor:      groupsig.DeserializeId(bh.Castor),
		GroupID:     groupsig.DeserializeId(bh.GroupId),
		Signature:   common.ToHex(bh.Signature),
		Prove:       bh.ProveValue,
		EvictedTxs:  bh.EvictedTxs,
		TotalQN:     bh.TotalQN,
		StateRoot:   bh.StateTree,
		TxRoot:      bh.TxTree,
		ReceiptRoot: bh.ReceiptTree,
		Random:      common.ToHex(bh.Random),
		TxNum:       uint64(len(bh.Transactions)),
	}

	block.Txs = make([]common.Hash, len(bh.Transactions))
	if 0 != len(bh.Transactions) {
		for i, tx := range bh.Transactions {
			block.Txs[i] = tx[0]
		}
	}

	return block
}

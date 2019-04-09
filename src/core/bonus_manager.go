package core

import (
	"bytes"
	"x/src/common"
	"sync"

	"x/src/middleware/types"
	"x/src/storage/account"
)

type BonusManager struct {
	lock sync.RWMutex
}

func newBonusManager() *BonusManager {
	manager := &BonusManager{}
	return manager
}

func (bm *BonusManager) GetBonusTransactionByBlockHash(blockHash []byte) *types.Transaction {
	transactionHash := blockChainImpl.LatestStateDB().GetData(common.BonusStorageAddress, string(blockHash))
	if transactionHash == nil {
		return nil
	}
	transaction, _ := blockChainImpl.transactionPool.GetTransaction(common.BytesToHash(transactionHash))
	return transaction
}

func (bm *BonusManager) GenerateBonus(targetIds []int32, blockHash common.Hash, groupId []byte, totalValue uint64) (*types.Bonus, *types.Transaction) {
	group := groupChainImpl.GetGroupById(groupId)
	buffer := &bytes.Buffer{}
	buffer.Write(groupId)
	//Logger.Debugf("GenerateBonus Group:%s",common.BytesToAddress(groupId).GetHexString())
	if len(targetIds) == 0 {
		panic("GenerateBonus targetIds size 0")
	}
	for i := 0; i < len(targetIds); i++ {
		index := targetIds[i]
		buffer.Write(group.Members[index])
		//Logger.Debugf("GenerateBonus Index:%d Member:%s",index,common.BytesToAddress(group.Members[index].Id).GetHexString())
	}
	transaction := &types.Transaction{}
	transaction.Data = blockHash.String()
	transaction.ExtraData = buffer.Bytes()
	if len(buffer.Bytes())%common.AddressLength != 0 {
		panic("GenerateBonus ExtraData Size Invalid")
	}
	//transaction.Value = totalValue / uint64(len(targetIds))
	transaction.Type = types.TransactionTypeBonus
	//transaction.GasPrice = common.MaxUint64
	transaction.Hash = transaction.GenHash()
	return &types.Bonus{TxHash: transaction.Hash, TargetIds: targetIds, BlockHash: blockHash, GroupId: groupId, TotalValue: totalValue}, transaction
}

func (bm *BonusManager) ParseBonusTransaction(transaction *types.Transaction) ([]byte, [][]byte, common.Hash, uint64) {
	reader := bytes.NewReader(transaction.ExtraData)
	groupId := make([]byte, common.GroupIdLength)
	addr := make([]byte, common.AddressLength)
	if n, _ := reader.Read(groupId); n != common.GroupIdLength {
		panic("ParseBonusTransaction Read GroupId Fail")
	}
	ids := make([][]byte, 0)
	for n, _ := reader.Read(addr); n > 0; n, _ = reader.Read(addr) {
		if n != common.AddressLength {
			logger.Debugf("ParseBonusTransaction Addr Size:%d Invalid", n)
			//panic("ParseBonusTransaction Read Address Fail")
			break
		}
		ids = append(ids, addr)
		addr = make([]byte, common.AddressLength)
	}
	blockHash := common.StringToHash(transaction.Data)
	return groupId, ids, blockHash, 0
}

func (bm *BonusManager) contain(blockHash []byte, accountdb *account.AccountDB) bool {
	value := accountdb.GetData(common.BonusStorageAddress, string(blockHash))
	if value != nil {
		return true
	}
	return false
}

func (bm *BonusManager) put(blockHash []byte, transactionHash []byte, accountdb *account.AccountDB) {
	accountdb.SetData(common.BonusStorageAddress, string(blockHash), transactionHash)
}

package core

import (
	"x/src/middleware/types"
	"x/src/storage/account"
	"x/src/utility"
	"fmt"
	"encoding/json"
	"x/src/common"
	"x/src/service"
)

type coinDepositExecutor struct {
}

type ftDepositExecutor struct {
}

type nftDepositExecutor struct {
}

//主链币充值确认
func (this *coinDepositExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) bool {
	txLogger.Tracef("Execute coin deposit ack tx:%s", transaction.ToTxJson().ToString())
	if transaction.Data == "" {
		return false
	}
	var depositCoinData types.DepositCoinData
	err := json.Unmarshal([]byte(transaction.Data), &depositCoinData)
	if err != nil {
		txLogger.Errorf("Deposit coin data unmarshal error:%s", err.Error())
		return false
	}
	txLogger.Tracef("deposit coin data:%v,target address:%s", depositCoinData, transaction.Source)
	if depositCoinData.Amount == "" || depositCoinData.ChainType == "" {
		return false
	}

	value, _ := utility.StrToBigInt(depositCoinData.Amount)
	return accountdb.AddFT(common.HexToAddress(transaction.Source), fmt.Sprintf("official-%s", depositCoinData.ChainType), value)

}

//FT充值确认
func (this *ftDepositExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) bool {
	txLogger.Tracef("Execute ft deposit ack tx:%s", transaction.ToTxJson().ToString())
	if transaction.Data == "" {
		return false
	}
	var depositFTData types.DepositFTData
	err := json.Unmarshal([]byte(transaction.Data), &depositFTData)
	if err != nil {
		txLogger.Errorf("Deposit ft data unmarshal error:%s", err.Error())
		return false
	}
	txLogger.Tracef("deposit ft data:%v, address:%s", depositFTData, transaction.Source)
	if depositFTData.Amount == "" || depositFTData.FTId == "" {
		return false
	}
	//todo 先不检查此ft是否存在
	value, _ := utility.StrToBigInt(depositFTData.Amount)
	return accountdb.AddFT(common.HexToAddress(transaction.Source), depositFTData.FTId, value)

}

//NFT充值确认
func (this *nftDepositExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) bool {
	txLogger.Tracef("Execute nft deposit ack tx:%s", transaction.ToTxJson().ToString())
	if transaction.Data == "" {
		return false
	}
	var depositNFTData types.DepositNFTData
	err := json.Unmarshal([]byte(transaction.Data), &depositNFTData)
	if err != nil {
		txLogger.Errorf("Deposit nft data unmarshal error:%s", err.Error())
		return false
	}
	//todo 这里需要重写
	txLogger.Tracef("deposit nft data:%v,target address:%s", depositNFTData, transaction.Source)
	if depositNFTData.SetId == "" || depositNFTData.ID == "" {
		return false
	}

	// 检查setId
	nftSet := service.NFTManagerInstance.GetNFTSet(depositNFTData.SetId, accountdb)
	if nil == nftSet {
		nftSet = service.NFTManagerInstance.GenerateNFTSet(depositNFTData.SetId, depositNFTData.Name, depositNFTData.Symbol, depositNFTData.Creator, depositNFTData.Owner, 0, depositNFTData.CreateTime, )
		service.NFTManagerInstance.PublishNFTSet(nftSet, accountdb)
	}

	appId := transaction.Target
	str, ok := service.NFTManagerInstance.GenerateNFT(nftSet, appId, depositNFTData.SetId, depositNFTData.ID, "", depositNFTData.Creator, depositNFTData.CreateTime, "official", common.HexToAddress(transaction.Source), depositNFTData.Data, accountdb)
	txLogger.Debugf("GenerateNFT result:%s,%t", str, ok)
	return ok
}

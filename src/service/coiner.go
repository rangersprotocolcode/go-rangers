package service

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"fmt"
)

//主链币充值确认
func CoinDeposit(accountdb *account.AccountDB, transaction *types.Transaction) (bool, string) {
	txLogger.Tracef("Execute coin deposit ack tx:%s", transaction.ToTxJson().ToString())
	if transaction.Data == "" {
		return false, fmt.Sprintf("data error, data: %s", transaction.Data)
	}
	var depositCoinData types.DepositCoinData
	err := json.Unmarshal([]byte(transaction.Data), &depositCoinData)
	if err != nil {
		txLogger.Errorf("Deposit coin data unmarshal error:%s", err.Error())
		return false, fmt.Sprintf("data error, data: %s", transaction.Data)
	}
	txLogger.Tracef("deposit coin data: %v,target address:%s", depositCoinData, transaction.Source)
	if depositCoinData.Amount == "" || depositCoinData.ChainType == "" {
		return false, fmt.Sprintf("data error, data: %s", transaction.Data)
	}

	value, _ := utility.StrToBigInt(depositCoinData.Amount)
	result := accountdb.AddFT(common.HexToAddress(transaction.Source), fmt.Sprintf("official-%s", depositCoinData.ChainType), value)
	if result {
		return result, fmt.Sprintf("coin: %s, deposit %s", fmt.Sprintf("official-%s", depositCoinData.ChainType), value)
	}
	return result, fmt.Sprintf("too much value %s", value)

}

//FT充值确认
func FTDeposit(accountdb *account.AccountDB, transaction *types.Transaction) (bool, string) {
	txLogger.Tracef("Execute ft deposit ack tx:%s", transaction.ToTxJson().ToString())
	if transaction.Data == "" {
		return false, fmt.Sprintf("data error, data: %s", transaction.Data)
	}
	var depositFTData types.DepositFTData
	err := json.Unmarshal([]byte(transaction.Data), &depositFTData)
	if err != nil {
		txLogger.Errorf("Deposit ft data unmarshal error:%s", err.Error())
		return false, fmt.Sprintf("data error, data: %s", transaction.Data)
	}
	txLogger.Tracef("deposit ft data:%v, address:%s", depositFTData, transaction.Source)
	if depositFTData.Amount == "" || depositFTData.FTId == "" {
		return false, fmt.Sprintf("data error, data: %s", transaction.Data)
	}
	//todo 先不检查此ft是否存在
	value, _ := utility.StrToBigInt(depositFTData.Amount)
	result := accountdb.AddFT(common.HexToAddress(transaction.Source), depositFTData.FTId, value)
	if result {
		return result, fmt.Sprintf("coin: %s, deposit %s", depositFTData.FTId, value)
	}
	return result, fmt.Sprintf("too much value %s", value)
}

//NFT充值确认
func NFTDeposit(accountdb *account.AccountDB, transaction *types.Transaction) (bool, string) {
	txLogger.Tracef("Execute nft deposit ack tx:%s", transaction.ToTxJson().ToString())
	if transaction.Data == "" {
		return false, fmt.Sprintf("data error, data: %s", transaction.Data)
	}
	var depositNFTData types.DepositNFTData
	err := json.Unmarshal([]byte(transaction.Data), &depositNFTData)
	if err != nil {
		txLogger.Errorf("Deposit nft data unmarshal error:%s", err.Error())
		return false, fmt.Sprintf("data error, data: %s", transaction.Data)
	}
	//todo 这里需要重写
	txLogger.Tracef("deposit nft data:%v,target address:%s", depositNFTData, transaction.Source)
	if depositNFTData.SetId == "" || depositNFTData.ID == "" {
		return false, fmt.Sprintf("data error, data: %s", transaction.Data)
	}

	// 检查setId
	nftSet := NFTManagerInstance.GetNFTSet(depositNFTData.SetId, accountdb)
	if nil == nftSet {
		nftSet = NFTManagerInstance.GenerateNFTSet(depositNFTData.SetId, depositNFTData.Name, depositNFTData.Symbol, depositNFTData.Creator, depositNFTData.Owner, 0, depositNFTData.CreateTime)
		NFTManagerInstance.PublishNFTSet(nftSet, accountdb)
	}

	appId := transaction.Target
	str, ok := NFTManagerInstance.GenerateNFT(nftSet, appId, depositNFTData.SetId, depositNFTData.ID, "", depositNFTData.Creator, depositNFTData.CreateTime, "official", common.HexToAddress(transaction.Source), depositNFTData.Data, accountdb)
	msg := fmt.Sprintf("depositNFT result: %s, %t", str, ok)
	txLogger.Debugf(msg)
	return ok, msg
}

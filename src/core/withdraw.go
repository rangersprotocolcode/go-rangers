package core

import (
	"x/src/storage/account"
	"x/src/middleware/types"
	"encoding/json"
	"x/src/common"
	"x/src/utility"
	"fmt"
	"x/src/network"
)

// 提现
func Withdraw(accountdb *account.AccountDB, transaction *types.Transaction, isSendToConnector bool) (string, bool) {
	txLogger.Debugf("Execute withdraw tx:%v", transaction)
	if transaction.Data == "" {
		return "Withdraw Data Bad Format", false
	}
	var withDrawReq types.WithDrawReq
	err := json.Unmarshal([]byte(transaction.Data), &withDrawReq)
	if err != nil {
		txLogger.Debugf("Unmarshal data error:%s", err.Error())
		return "Withdraw Data Bad Format", false
	}
	if withDrawReq.ChainType == "" || withDrawReq.Address == "" {
		return "Withdraw Data Bad Format", false
	}

	source := common.HexToAddress(transaction.Source)
	result := make(map[string]string)

	//主链币检查
	if withDrawReq.Balance != "" {
		withdrawAmount, err := utility.StrToBigInt(withDrawReq.Balance)
		if err != nil {
			txLogger.Error("Execute withdraw bad amount!Hash:%s, err:%s", transaction.Hash.String(), err.Error())
			return "Withdraw Data BNT Bad Format", false
		}

		coinId := fmt.Sprintf("official-%s", withDrawReq.ChainType)
		left, ok := accountdb.SubFT(source, coinId, withdrawAmount)
		if !ok {
			subAccountBalance := accountdb.GetFT(source, coinId)
			txLogger.Errorf("Execute withdraw balance not enough:current balance:%d,withdraw balance:%d", subAccountBalance.Uint64(), withdrawAmount.Uint64())
			return "BNT Not Enough", false
		} else {
			result["token"] = withDrawReq.ChainType
			result["balance"] = left.String()
			result["lockedBalance"] = withDrawReq.Balance
		}
	}

	//ft
	if withDrawReq.FT != nil && len(withDrawReq.FT) != 0 {
		for k, v := range withDrawReq.FT {
			subValue := accountdb.GetFT(source, k)
			compareResult, sub := canWithDraw(v, subValue)
			if !compareResult {
				return "FT Not Enough", false
			}

			// 扣ft
			accountdb.SetFT(source, k, sub)
		}
	}

	//nft
	nftInfo := make([]types.NFTID, 0)
	if withDrawReq.NFT != nil && len(withDrawReq.NFT) != 0 {
		for _, k := range withDrawReq.NFT {
			nft := accountdb.GetNFTById(source, k.SetId, k.Id)
			if nil == nft {
				return "NFT Not Exist In This Game", false
			}

			//删除要提现的NFT
			accountdb.RemoveNFT(source, nft)

			nftInfo = append(nftInfo, types.NFTID{SetId: k.SetId, Id: k.Id, Data: nft.ToJSONString()})
			result["chainType"] = withDrawReq.ChainType
			result["setId"] = k.SetId
			result["tokenId"] = k.Id
			result["targetAddress"] = withDrawReq.Address
		}
	}

	if isSendToConnector && !sendWithdrawToConnector(withDrawReq, transaction, nftInfo) {
		return "Send To Connector Error", false
	}

	resultString, _ := json.Marshal(result)
	return string(resultString), true
}

func sendWithdrawToConnector(withDrawReq types.WithDrawReq, transaction *types.Transaction, nftInfo []types.NFTID) bool {
	//发送给Coin Connector
	withdrawData := types.WithDrawData{ChainType: withDrawReq.ChainType, Balance: withDrawReq.Balance, Address: withDrawReq.Address}
	withdrawData.FT = withDrawReq.FT
	withdrawData.NFT = nftInfo
	b, err := json.Marshal(withdrawData)
	if err != nil {
		txLogger.Error("Execute withdraw tx:%s json marshal err, err:%s", transaction.Hash.String(), err.Error())
		return false
	}

	t := types.Transaction{Source: transaction.Source, Target: transaction.Target, Data: string(b), Type: transaction.Type}
	t.Hash = t.GenHash()

	msg, err := json.Marshal(t.ToTxJson())
	if err != nil {
		txLogger.Debugf("Json marshal tx json error:%s", err.Error())
		return false
	}

	txLogger.Debugf("After execute withdraw.Send msg to coin proxy:%s", msg)
	network.GetNetInstance().SendToCoinConnector(msg)
	return true
}

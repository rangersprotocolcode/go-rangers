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

package service

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/network"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"math/big"
)

// 提现
func Withdraw(accountdb *account.AccountDB, transaction *types.Transaction, isSendToConnector bool) (string, bool) {
	txLogger.Tracef("Execute withdraw tx:%s", transaction.ToTxJson().ToString())
	if transaction.Data == "" {
		return "Withdraw Data Bad Format", false
	}
	var withDrawReq types.WithDrawReq
	err := json.Unmarshal([]byte(transaction.Data), &withDrawReq)
	if err != nil {
		txLogger.Errorf("Unmarshal data error:%s", err.Error())
		return "Withdraw Data Bad Format", false
	}
	if withDrawReq.Address == "" {
		return "Withdraw Data Bad Format", false
	}

	source := common.HexToAddress(transaction.Source)
	result := types.NewJSONObject()
	result.Put("txHash", transaction.Hash.String())

	//主链币检查
	if withDrawReq.BNT.TokenType != "" {
		withdrawAmount, err := utility.StrToBigInt(withDrawReq.BNT.Value)
		if err != nil {
			txLogger.Error("Execute withdraw bad amount!Hash:%s, err:%s", transaction.Hash.String(), err.Error())
			return "Withdraw Data BNT Bad Format", false
		}

		left, ok := accountdb.SubBNT(source, withDrawReq.BNT.TokenType, withdrawAmount)
		if !ok {
			subAccountBalance := accountdb.GetBNT(source, withDrawReq.BNT.TokenType)
			txLogger.Errorf("Execute withdraw balance not enough:current balance:%d,withdraw balance:%d", subAccountBalance.Uint64(), withdrawAmount.Uint64())
			return "BNT Not Enough", false
		} else {
			responseCoin := make(map[string]string)
			responseCoin["token"] = withDrawReq.BNT.TokenType
			responseCoin["balance"] = utility.BigIntToStr(left)
			responseCoin["lockedBalance"] = withDrawReq.BNT.Value
			result.Put("coin", responseCoin)
		}
	}

	//ft
	if withDrawReq.FT != nil && len(withDrawReq.FT) != 0 {
		if withDrawReq.ChainType == "" {
			return "Withdraw Data Bad Format", false
		}

		ftList := make([]map[string]string, 0)
		for k, v := range withDrawReq.FT {
			subValue := accountdb.GetFT(source, k)
			compareResult, sub := canWithDraw(v, subValue)
			if !compareResult {
				return "FT Not Enough", false
			}

			// 扣ft
			accountdb.SetFT(source, k, sub)

			ftMap := make(map[string]string)
			ftMap["ftId"] = k
			ftMap["balance"] = utility.BigIntToStr(sub)
			ftMap["lockedBalance"] = v
			ftList = append(ftList, ftMap)
		}

		if len(ftList) != 0 {
			result.Put("ft", ftList)
		}
	}

	//nft
	nftInfo := make([]types.NFTID, 0)
	if withDrawReq.NFT != nil && len(withDrawReq.NFT) != 0 {
		if withDrawReq.ChainType == "" {
			return "Withdraw Data Bad Format", false
		}

		nftList := make([]map[string]string, 0)
		for _, k := range withDrawReq.NFT {
			nft := NFTManagerInstance.MarkNFTWithdrawn(source, k.SetId, k.Id, accountdb)
			if nil == nft {
				return "NFT Not Exist In This Game", false
			}

			nftInfo = append(nftInfo, types.NFTID{SetId: k.SetId, Id: k.Id, Data: nft.ToJSONString()})

			nftMap := make(map[string]string)
			nftMap["setId"] = k.SetId
			nftMap["tokenId"] = k.Id
			nftList = append(nftList, nftMap)
		}

		if len(nftList) != 0 {
			result.Put("nft", nftList)
		}
	}

	if isSendToConnector && !sendWithdrawToCoiner(withDrawReq, transaction, nftInfo) {
		return "Send To Connector Error", false
	}

	return result.TOJSONString(), true
}

//todo:delete after test
func sendWithdrawToCoiner(withDrawReq types.WithDrawReq, transaction *types.Transaction, nftInfo []types.NFTID) bool {
	withdrawData := types.WithDrawData{ChainType: withDrawReq.ChainType, Address: withDrawReq.Address}
	withdrawData.BNT = withDrawReq.BNT
	withdrawData.FT = withDrawReq.FT
	withdrawData.NFT = nftInfo
	b, err := json.Marshal(withdrawData)
	if err != nil {
		txLogger.Error("Execute withdraw tx:%s json marshal err, err:%s", transaction.Hash.String(), err.Error())
		return false
	}

	t := types.Transaction{Source: transaction.Source, Target: transaction.Target, Data: string(b), Type: transaction.Type, Time: transaction.Time, Nonce: transaction.Nonce, Hash: transaction.Hash}

	msg, err := json.Marshal(t.ToTxJson())
	if err != nil {
		txLogger.Debugf("Json marshal tx json error:%s", err.Error())
		return false
	}

	txLogger.Tracef("After execute withdraw.Send msg to coin proxy:%s", t.ToTxJson().ToString())
	go network.GetNetInstance().SendToCoinConnector(msg)
	return true
}

/**
字符串的余额比较
withDrawAmount金额大于等于ftValue 返回TRUE 否则返回FALSE
withDrawAmount 是浮点数的STRING "11.256"
ftValue 是bigInt的STRING "56631"
如果格式有错误返回FALSE
返回为true的话 返回二者的差值string
*/
func canWithDraw(withDrawAmount string, ftValue *big.Int) (bool, *big.Int) {
	b1, err1 := utility.StrToBigInt(withDrawAmount)
	if err1 != nil {
		return false, nil
	}

	if b1.Cmp(ftValue) > 0 {
		return false, nil
	}

	return true, ftValue.Sub(ftValue, b1)
}

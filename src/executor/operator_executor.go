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

package executor

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/statemachine"
	"com.tuntun.rocket/node/src/storage/account"
	"encoding/json"
	"fmt"
	"strconv"
)

type operatorExecutor struct {
	baseFeeExecutor
	logger log.Logger
}

func (this *operatorExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	this.logger.Debugf("txhash: %s begin transaction is not nil!", transaction.Hash.String())

	gameId := transaction.Target
	isTransferOnly := 0 == len(gameId)

	// 纯转账的场景，不用执行状态机
	if isTransferOnly {
		return this.transfer(transaction.Source, transaction.ExtraData, transaction.Hash.String(), accountdb)
	}

	// 在交易池里，表示game_executor已经执行过状态机了
	if nil != service.TxManagerInstance.BeginTransaction(gameId, accountdb, transaction) {
		if this.checkGameExecutor(context) {
			return true, "Tx Is Executed"
		}

		// 只要处理交易里的subTransaction即可
		ok, msg := this.transfer(transaction.Source, transaction.ExtraData, transaction.Hash.String(), accountdb)
		if !ok {
			return false, "fail to transfer"
		}
		if 0 != len(transaction.SubTransactions) {
			for _, user := range transaction.SubTransactions {
				this.logger.Debugf("Execute sub tx:%v", user)

				// 发币
				if user.Address == "StartFT" {
					createTime, _ := user.Assets["createTime"]
					ftSet := service.FTManagerInstance.GenerateFTSet(user.Assets["name"], user.Assets["symbol"], user.Assets["gameId"], user.Assets["totalSupply"], user.Assets["owner"], createTime, 1)
					msg, flag := service.FTManagerInstance.PublishFTSet(ftSet, accountdb)
					if !flag {
						return false, msg
					}

					continue
				}

				if user.Address == "MintFT" {
					owner := user.Assets["appId"]
					ftId := user.Assets["ftId"]
					target := user.Assets["target"]
					supply := user.Assets["balance"]
					msg, flag := service.FTManagerInstance.MintFT(owner, ftId, target, supply, accountdb)

					if !flag {
						return false, msg
					}
					continue
				}

				// 给用户币
				if user.Address == "TransferFT" {
					msg, _, flag := service.FTManagerInstance.TransferFT(user.Assets["gameId"], user.Assets["symbol"], user.Assets["target"], user.Assets["supply"], accountdb)
					if !flag {
						return false, msg
					}
					continue
				}

				// 修改NFT属性
				if user.Address == "UpdateNFT" {
					flag := service.NFTManagerInstance.UpdateNFT(user.Assets["appId"], user.Assets["setId"], user.Assets["id"], user.Assets["data"], user.Assets["property"], accountdb)
					if !flag {
						return false, fmt.Sprintf("nft not existed, setId: %s, id: %s", user.Assets["setId"], user.Assets["id"])
					}
					continue
				}

				// NFT
				if user.Address == "TransferNFT" {
					appId := user.Assets["appId"]
					msg, ok := service.NFTManagerInstance.Transfer(user.Assets["setId"], user.Assets["id"], common.HexToAddress(appId), common.HexToAddress(user.Assets["target"]), accountdb)
					if !ok {
						return false, msg
					}
					continue
				}

				// 将状态机持有的NFT的使用权授予某地址
				if user.Address == "ApproveNFT" {
					appId := user.Assets["appId"]
					ok := accountdb.ApproveNFT(common.HexToAddress(appId), appId, user.Assets["setId"], user.Assets["id"], user.Assets["target"])
					if !ok {
						return false, fmt.Sprintf("nft not existed, setId: %s, id: %s", user.Assets["setId"], user.Assets["id"])
					}
					continue
				}

				if user.Address == "changeNFTStatus" {
					appId := user.Assets["appId"]
					status, _ := strconv.Atoi(user.Assets["status"])
					ok := accountdb.ChangeNFTStatus(common.HexToAddress(appId), appId, user.Assets["setId"], user.Assets["id"], byte(status))
					if !ok {
						return false, fmt.Sprintf("nft not existed, setId: %s, id: %s", user.Assets["setId"], user.Assets["id"])
					}
					continue
				}

				if user.Address == "PublishNFTSet" {
					maxSupplyString := user.Assets["maxSupply"]
					maxSupply, err := strconv.ParseUint(maxSupplyString, 10, 64)
					if err != nil {
						msg := fmt.Sprintf("publish nft set! maxSupply bad format: %s", maxSupplyString)
						this.logger.Errorf(msg)
						return false, msg
					}
					appId := user.Assets["appId"]
					nftSet := service.NFTManagerInstance.GenerateNFTSet(user.Assets["setId"], user.Assets["name"], user.Assets["symbol"], appId, appId, types.NFTConditions{}, maxSupply, user.Assets["createTime"])
					msg, ok := service.NFTManagerInstance.PublishNFTSet(nftSet, accountdb)
					if !ok {
						return false, msg
					}
					continue
				}

				if user.Address == "MintNFT" {
					msg, ok := service.NFTManagerInstance.MintNFT(user.Assets["source"], user.Assets["appId"], user.Assets["setId"], user.Assets["id"], user.Assets["data"], user.Assets["createTime"], common.HexToAddress(user.Assets["target"]), accountdb)
					if !ok {
						return false, msg
					}
					continue
				}

				// 用户之间转账
				if !service.UpdateAsset(user, transaction.Target, accountdb) {
					return false, "fail to transfer"
				}
			}
		}

		return true, msg
	}

	// 本地没有执行过状态机(game_executor还没有收到消息)，则需要调用状态机
	// todo: RequestId 排序问题
	ok, _ := this.transfer(transaction.Source, transaction.ExtraData, transaction.Hash.String(), accountdb)
	if !ok {
		service.GetTransactionPool().PutGameData(transaction.Hash)
		service.TxManagerInstance.RollBack(gameId)
		return false, "fail to transfer"
	}

	// 调用状态机
	transaction.SubTransactions = make([]types.UserData, 0)
	outputMessage := statemachine.STMManger.Process(gameId, "operator", transaction.RequestId, transaction.Data, transaction)
	service.GetTransactionPool().PutGameData(transaction.Hash)
	result := ""
	if outputMessage != nil {
		result = outputMessage.Payload
	}

	if 0 == len(result) || result == "fail to transfer" || outputMessage == nil || outputMessage.Status == 1 {
		service.TxManagerInstance.RollBack(gameId)
		return false, "fail to transfer or stm has no response"
	}

	service.TxManagerInstance.Commit(gameId)
	return true, result
}

// 处理转账
// 支持多人转账{"address1":"value1", "address2":"value2"}

// 处理转账
// 支持source地址给多人转账，包含余额，ft，nft
// 数据格式{"address1":{"balance":"127","ft":{"name1":"189","name2":"1"},"nft":["id1","sword2"]}, "address2":{"balance":"1"}}
//
//{
//	"address1": {
//		"bnt": {
//          "ETH.ETH":"0.008",
//          "NEO.CGAS":"100"
//      },
//		"ft": {
//			"name1": "189",
//			"name2": "1"
//		},
//		"nft": [{"setId":"suit1","id":"xizhuang"},
//              {"setId":"gun","id":"rifle"}
// 				]
//	},
//	"address2": {
//		"balance": "1"
//	}
//}
func (this *operatorExecutor) transfer(source, data, hash string, accountdb *account.AccountDB) (bool, string) {
	if 0 == len(data) {
		return true, ""
	}

	mm := make(map[string]types.TransferData, 0)
	if err := json.Unmarshal([]byte(data), &mm); nil != err {
		return false, fmt.Sprintf("bad extraData: %s", data)

	}

	msg, ok := service.ChangeAssets(source, mm, accountdb)
	this.logger.Debugf("txhash: %s, finish changeAssets. msg: %s", hash, msg)
	return ok, msg
}

func (this *operatorExecutor) checkGameExecutor(context map[string]interface{}) bool {
	_, ok := context["gameExecutor"]
	return ok
}

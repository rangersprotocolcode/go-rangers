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

	// 处理转账
	// 支持多人转账{"address1":"value1", "address2":"value2"}
	// 理论上这里不应该失败，nonce保证了这一点
	if 0 != len(transaction.ExtraData) {
		mm := make(map[string]types.TransferData, 0)
		if err := json.Unmarshal([]byte(transaction.ExtraData), &mm); nil != err {
			return false, fmt.Sprintf("bad extraData: %s", transaction.ExtraData)

		}
		msg, ok := service.ChangeAssets(transaction.Source, mm, accountdb)
		this.logger.Debugf("txhash: %s, finish changeAssets. msg: %s", transaction.Hash.String(), msg)
		if !ok {
			return false, msg
		}
	}

	// 纯转账的场景，不用执行状态机
	if 0 == len(transaction.Target) {
		return true, ""
	}

	// 在交易池里，表示game_executor已经执行过状态机了
	// 只要处理交易里的subTransaction即可
	if nil != service.TxManagerInstance.BeginTransaction(transaction.Target, accountdb, transaction) {
		this.logger.Debugf("Is not game data")
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
					addr := common.HexToAddress(user.Assets["addr"])
					flag := service.NFTManagerInstance.UpdateNFT(addr, user.Assets["appId"], user.Assets["setId"], user.Assets["id"], user.Assets["data"], accountdb)
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
					maxSupply, err := strconv.Atoi(maxSupplyString)
					if err != nil {
						msg := fmt.Sprintf("publish nft set! maxSupply bad format: %s", maxSupplyString)
						this.logger.Errorf(msg)
						return false, msg
					}
					appId := user.Assets["appId"]
					nftSet := service.NFTManagerInstance.GenerateNFTSet(user.Assets["setId"], user.Assets["name"], user.Assets["symbol"], appId, appId, maxSupply, user.Assets["createTime"])
					msg, ok := service.NFTManagerInstance.PublishNFTSet(nftSet, accountdb)
					if !ok {
						return false, msg
					}
					continue
				}

				if user.Address == "MintNFT" {
					msg, ok := service.NFTManagerInstance.MintNFT(user.Assets["appId"], user.Assets["setId"], user.Assets["id"], user.Assets["data"], user.Assets["createTime"], common.HexToAddress(user.Assets["target"]), accountdb)
					if !ok {
						return false, msg
					}
					continue
				}

				// 用户之间转账
				if !service.UpdateAsset(user, transaction.Target, accountdb) {
					return false, ""
				}
			}
		}

		return true, ""
	}

	// 本地没有执行过状态机(game_executor还没有收到消息)，则需要调用状态机
	// todo: RequestId 排序问题
	transaction.SubTransactions = make([]types.UserData, 0)
	outputMessage := statemachine.STMManger.Process(transaction.Target, "operator", transaction.RequestId, transaction.Data, transaction)
	service.GetTransactionPool().PutGameData(transaction.Hash)
	result := ""
	if outputMessage != nil {
		result = outputMessage.Payload
	}

	if 0 == len(result) || result == "fail to transfer" || outputMessage == nil || outputMessage.Status == 1 {
		service.TxManagerInstance.RollBack(transaction.Target)
		return false, "fail to transfer or stm has no response"
	}

	service.TxManagerInstance.Commit(transaction.Target)
	return true, ""
}

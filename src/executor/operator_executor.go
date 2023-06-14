// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package executor

import (
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"encoding/json"
	"fmt"
)

type operatorExecutor struct {
	baseFeeExecutor
	logger log.Logger
}

func (this *operatorExecutor) Execute(transaction *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string) {
	this.logger.Debugf("txhash: %s begin transaction is not nil!", transaction.Hash.String())
	return this.transfer(transaction.Source, transaction.ExtraData, transaction.Hash.String(), accountdb)

}

// {"address1":{"balance":"127","ft":{"name1":"189","name2":"1"},"nft":["id1","sword2"]}, "address2":{"balance":"1"}}
//
//	{
//		"address1": {
//			"bnt": {
//	         "ETH.ETH":"0.008",
//	         "NEO.CGAS":"100"
//	     },
//			"ft": {
//				"name1": "189",
//				"name2": "1"
//			},
//			"nft": [{"setId":"suit1","id":"xizhuang"},
//	             {"setId":"gun","id":"rifle"}
//					]
//		},
//		"address2": {
//			"balance": "1"
//		}
//	}
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

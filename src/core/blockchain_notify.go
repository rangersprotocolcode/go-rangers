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

package core

import (
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/network"
	"encoding/json"
	"strconv"
)

func (chain *blockChain) notifyReceipts(receipts types.Receipts) {
	if nil == receipts || 0 == len(receipts) {
		return
	}

	for _, receipt := range receipts {
		if 0 == len(receipt.Source) {
			continue
		}

		msg := make(map[string]string, 0)
		msg["msg"] = receipt.Source
		msg["hash"] = receipt.TxHash.Hex()
		msg["height"] = strconv.FormatUint(receipt.Height, 10)
		msgBytes, _ := json.Marshal(msg)
		network.GetNetInstance().Notify(true, "rocketprotocol", receipt.Source, string(msgBytes))
	}
}

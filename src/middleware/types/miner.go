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

package types

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"fmt"
)

type HexBytes []byte

func (h *HexBytes) UnmarshalJSON(b []byte) error {
	if 2 > len(b) {
		return fmt.Errorf("length error, %d", len(b))
	}
	res := utility.BytesToStr(b[1 : len(b)-1])
	*h = common.FromHex(res)
	return nil
}

func (h HexBytes) MarshalJSON() ([]byte, error) {
	res := fmt.Sprintf("\"%s\"", common.ToHex(h))
	return utility.StrToBytes(res), nil
}

type Miner struct {
	// 矿工机器编号
	Id           HexBytes `json:"id,omitempty"`
	PublicKey    HexBytes `json:"publicKey,omitempty"`
	VrfPublicKey []byte   `json:"vrfPublicKey,omitempty"`

	ApplyHeight uint64
	// 当前状态
	Status byte

	// 提案者 还是验证者
	Type byte `json:"type,omitempty"`

	// 质押数
	Stake uint64 `json:"stake,omitempty"`

	// 收益账户
	Account HexBytes `json:"account,omitempty"`
}

func (miner *Miner) GetMinerInfo() []byte {
	result := make(map[string]interface{})
	result["id"] = miner.Id
	result["publicKey"] = miner.PublicKey
	result["vrfPublicKey"] = miner.VrfPublicKey
	result["applyHeight"] = miner.ApplyHeight
	result["type"] = miner.Type

	resultBytes, _ := json.Marshal(result)
	return resultBytes
}

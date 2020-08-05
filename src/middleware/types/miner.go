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

type Miner struct {
	Id           []byte `json:"id,omitempty"`
	PublicKey    []byte `json:"publicKey,omitempty"`
	VrfPublicKey []byte `json:"vrfPublicKey,omitempty"`

	// 提案者 还是验证者
	Type byte `json:"type,omitempty"`

	// 质押数
	Stake uint64 `json:"stake,omitempty"`

	ApplyHeight uint64
	AbortHeight uint64 `json:"-"`

	// 当前状态
	Status byte
}

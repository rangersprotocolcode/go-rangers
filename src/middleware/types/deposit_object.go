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

// 主链币
type DepositCoinData struct {
	ChainType        string `json:"chainType,omitempty"`
	Amount           string `json:"amount,omitempty"`
	TxId             string `json:"txId,omitempty"`
	MainChainAddress string `json:"addr,omitempty"`
}

//FT充值确认数据结构
type DepositFTData struct {
	FTId             string `json:"ftId,omitempty"`
	Amount           string `json:"amount,omitempty"`
	MainChainAddress string `json:"addr,omitempty"`
	ContractAddress  string `json:"ContractAddr,omitempty"`
	TxId             string `json:"txId,omitempty"`
}

//NFT充值确认数据结构
type DepositNFTData struct {
	SetId      string            `json:"setId,omitempty"`
	Name       string            `json:"name,omitempty"`
	Symbol     string            `json:"symbol,omitempty"`
	ID         string            `json:"id,omitempty"`
	Creator    string            `json:"creator,omitempty"`
	CreateTime string            `json:"createTime,omitempty"`
	Owner      string            `json:"owner,omitempty"`
	Renter     string            `json:"renter,omitempty"`
	Status     byte              `json:"status,omitempty"`
	Condition  byte              `json:"condition,omitempty"`
	AppId      string            `json:"appId,omitempty"`
	Data       map[string]string `json:"data,omitempty"`

	MainChainAddress string `json:"addr,omitempty"`
	ContractAddress  string `json:"ContractAddr,omitempty"`
	TxId             string `json:"txId,omitempty"`
}

type ERC20BindingData struct {
	Name            string `json:"name,omitempty"`
	Decimal         int64  `json:"decimal,omitempty"`
	Position        int64  `json:"position,omitempty"`
	ContractAddress string `json:"contract,omitempty"`
}

type DepositNotify struct {
	Method string                 `json:"type"`
	Data   map[string]interface{} `json:"data"`
}

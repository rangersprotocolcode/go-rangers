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

package common

type GenesisConf struct {
	Creator string `json:"creator"`

	ChainId string `json:"chainId"`
	Name    string `json:"name"`

	Cast       uint64 `json:"cast"`
	GroupLife  uint64 `json:"groupLife"`
	Proposals  uint64 `json:"p"`
	Validators uint64 `json:"v"`

	GenesisTime int64 `json:"genesisTime"`

	TimeCycle      int    `json:"timecycle"`
	TokenName      string `json:"tokenName"`
	TotalSupply    uint64 `json:"totalsupply"`
	Symbol         string `json:"symbol"`
	ReleaseRate    int    `json:"d"`
	ProposalToken  int    `json:"ptoken"`
	ValidatorToken int    `json:"vtoken"`

	// 生成的创始组与创始矿工
	Group        string   `json:"group"`
	JoinedGroup  string   `json:"joined"`
	ProposerInfo []string `json:"proposers"`

	Dev byte `json:"dev"`
}

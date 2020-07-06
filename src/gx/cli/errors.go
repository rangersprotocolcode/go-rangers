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

package cli

import "errors"

var (
	// ErrorBlockChainUninitialized 未初始化链
	ErrorBlockChainUninitialized = errors.New("should init blockchain module first")
	// ErrorP2PUninitialized 未初始化P2P模块。
	ErrorP2PUninitialized = errors.New("should init P2P module first")
	// ErrorGovUninitialized 未初始化共识模块。
	ErrorGovUninitialized = errors.New("should init Governance module first")
	// ErrorWalletsUninitialized
	ErrorWalletsUninitialized = errors.New("should load wallets from config")
)

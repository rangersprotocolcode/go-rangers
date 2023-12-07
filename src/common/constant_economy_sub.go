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
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package common

import "com.tuntun.rangers/node/src/utility"

var (
	EconomyContract = HexToAddress("0x71d9cfd1b7adb1e8eb4c193ce6ffbe19b4aee0db")

	ProxySubGovernance = HexToAddress("0x57d9b509004657dfade3121d9aa32a3586c1ef49")

	WhitelistForCreate = "0x9c1cbfe5328dfb1733d59a7652d0a49228c7e12c"

	RpgReward = HexToAddress("0x4e0285462d9592fdcef54c9b0f8435814096c299")

	CreateWhiteListAddr = HexToAddress(WhitelistForCreate)

	CreateWhiteListPostion = utility.UInt64ToByte(0)
)

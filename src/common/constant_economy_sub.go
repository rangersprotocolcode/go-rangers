package common

import "com.tuntun.rocket/node/src/utility"

var (
	EconomyContract = HexToAddress("0x71d9cfd1b7adb1e8eb4c193ce6ffbe19b4aee0db")

	ProxySubGovernance = HexToAddress("0x795c5d906ee23d1f46654b51c8db6ecd60526a6b")

	WhitelistForCreate = "0x9c1cbfe5328dfb1733d59a7652d0a49228c7e12c"

	CreateWhiteListAddr    = HexToAddress(WhitelistForCreate)
	CreateWhiteListPostion = utility.UInt64ToByte(0)
)

package common

import "com.tuntun.rocket/node/src/utility"

var (
	EconomyContract = HexToAddress("0x71d9cfd1b7adb1e8eb4c193ce6ffbe19b4aee0db")

	ProxySubGovernance = HexToAddress("0xe3e5df5d1ea7e2a7e107e35098412aa6016bf89c")

	WhitelistForCreate = "0x9c1cbfe5328dfb1733d59a7652d0a49228c7e12c"

	RpgReward = HexToAddress("0xcde68d63304537f693214b0ab730b99eda6281a9")

	CreateWhiteListAddr = HexToAddress(WhitelistForCreate)

	CreateWhiteListPostion = utility.UInt64ToByte(0)
)

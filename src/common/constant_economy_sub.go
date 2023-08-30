package common

import "com.tuntun.rocket/node/src/utility"

var (
	EconomyContract = HexToAddress("0x71d9cfd1b7adb1e8eb4c193ce6ffbe19b4aee0db")

	ProxySubGovernance = HexToAddress("0x57d9b509004657dfade3121d9aa32a3586c1ef49")

	WhitelistForCreate = "0x9c1cbfe5328dfb1733d59a7652d0a49228c7e12c"

	RpgReward = HexToAddress("0x4e0285462d9592fdcef54c9b0f8435814096c299")

	CreateWhiteListAddr = HexToAddress(WhitelistForCreate)

	CreateWhiteListPostion = utility.UInt64ToByte(0)
)

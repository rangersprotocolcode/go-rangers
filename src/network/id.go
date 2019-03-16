package network

var proposerList = []string{"0xa7a4b347a626d2353050328763f74cd85f83173d6886b24501c39b5c456446d2", "0xf664571c43c13c49ccf032b9d823bedc9f62e5a5858a80d42cd5ee6d897237f0", "0x86805452068d92d647ae9b7a048a09515c875d71eb3d0d3d788c3f3334aaf124"}

var verifyGroupList = []string{"0x15fc80b99d8904205b768e04ccea60b67756fd8176ce27e95f5db1da9e57735"}
var verifyGroupsInfo = map[string][]string{"0x15fc80b99d8904205b768e04ccea60b67756fd8176ce27e95f5db1da9e57735": {"0xe75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b019", "0xd3d410ec7c917f084e0f4b604c7008f01a923676d0352940f68a97264d49fb76", "0x9d2961d1b4eb4af2d78cb9e29614756ab658671e453ea1f6ec26b4e918c79d02"}}

var net2MinerIdMap map[string]string

var miner2NetIdMap map[string]string

func initIdMap() {
	miner2NetIdMap = make(map[string]string)
	miner2NetIdMap["0xe75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b019"] = "QmTn5a8UhdgmNZx1Vy82kNwJ6RmHKcqocSjTg8VbPaXu69"
	miner2NetIdMap["0xd3d410ec7c917f084e0f4b604c7008f01a923676d0352940f68a97264d49fb76"] = "QmQde3pABret82FJeoNiKSRi6BaaNBssVQvHngnrgT8Wp8"
	miner2NetIdMap["0x9d2961d1b4eb4af2d78cb9e29614756ab658671e453ea1f6ec26b4e918c79d02"] = "QmdghpeZ6MRkz49t2YkqjSfWGysCX8mqjJWABMzZRFLyaj"
	miner2NetIdMap["0xa7a4b347a626d2353050328763f74cd85f83173d6886b24501c39b5c456446d2"] = "QmY5wpgkDBFBvvsRkD62a5GA496NLBQWcwN75wEN5bBhzB"
	miner2NetIdMap["0xf664571c43c13c49ccf032b9d823bedc9f62e5a5858a80d42cd5ee6d897237f0"] = "Qmc4KN9wEmcG22zFUe9rPGDuwcVPPKWoshZTgHePftZExU"
	miner2NetIdMap["0x86805452068d92d647ae9b7a048a09515c875d71eb3d0d3d788c3f3334aaf124"] = "QmPu4MVQs5gDDYFLxFt1EvVgfREfQ8Y1Q4fr7ykZMELrzm"

	net2MinerIdMap = make(map[string]string)
	net2MinerIdMap["QmTn5a8UhdgmNZx1Vy82kNwJ6RmHKcqocSjTg8VbPaXu69"] = "0xe75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b019"
	net2MinerIdMap["QmQde3pABret82FJeoNiKSRi6BaaNBssVQvHngnrgT8Wp8"] = "0xd3d410ec7c917f084e0f4b604c7008f01a923676d0352940f68a97264d49fb76"
	net2MinerIdMap["QmdghpeZ6MRkz49t2YkqjSfWGysCX8mqjJWABMzZRFLyaj"] = "0x9d2961d1b4eb4af2d78cb9e29614756ab658671e453ea1f6ec26b4e918c79d02"
	net2MinerIdMap["QmY5wpgkDBFBvvsRkD62a5GA496NLBQWcwN75wEN5bBhzB"] = "0xa7a4b347a626d2353050328763f74cd85f83173d6886b24501c39b5c456446d2"
	net2MinerIdMap["Qmc4KN9wEmcG22zFUe9rPGDuwcVPPKWoshZTgHePftZExU"] = "0xf664571c43c13c49ccf032b9d823bedc9f62e5a5858a80d42cd5ee6d897237f0"
	net2MinerIdMap["QmPu4MVQs5gDDYFLxFt1EvVgfREfQ8Y1Q4fr7ykZMELrzm"] = "0x86805452068d92d647ae9b7a048a09515c875d71eb3d0d3d788c3f3334aaf124"
}

func getNetId(minerId string) string {
	netId, found := miner2NetIdMap[minerId]
	if !found {
		Logger.Errorf("Unknown miner id:%s", minerId)
		return ""
	}
	return netId
}

func getMinerId(netId string) string {
	minerId, found := net2MinerIdMap[netId]
	if !found {
		Logger.Errorf("Unknown net id:%s", minerId)
		return ""
	}
	return minerId
}

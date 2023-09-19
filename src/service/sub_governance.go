package service

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/storage/account"
	"math"
)

func GetSubChainStatus(stateDB *account.AccountDB) byte {
	value := stateDB.GetData(common.ProxySubGovernance, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 6})
	if nil == value || 0 == len(value) {
		return math.MaxUint8
	}

	return value[len(value)-1]
}

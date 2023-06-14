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

package access

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/middleware/log"
	"strconv"
)

var (
	logger = log.GetLoggerByIndex(log.AccessLogConfig, strconv.Itoa(common.InstanceIndex))
	pkPool pubkeyPool
)

// pubkeyPool is the cache stores public keys of miners which is used for accelerated calculation
type pubkeyPool struct {
	minerPoolReader *MinerPoolReader
}

func InitPubkeyPool(minerPoolReader *MinerPoolReader) {
	pkPool = pubkeyPool{
		minerPoolReader: minerPoolReader,
	}
}

// GetMinerPubKey GetMinerPK returns pubic key of the given id
// It firstly retrieves from the cache, if missed, it gets from the chain and updates the cache.
func GetMinerPubKey(id groupsig.ID) *groupsig.Pubkey {
	if !ready() {
		return nil
	}

	value, err := pkPool.minerPoolReader.GetPubkey(id)
	if err == nil {
		pk := groupsig.ByteToPublicKey(value)
		return &pk
	}
	return nil
}

func ready() bool {
	return pkPool.minerPoolReader != nil
}

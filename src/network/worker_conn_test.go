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

package network

import (
	"com.tuntun.rangers/node/src/middleware/log"
	"fmt"
	"hash/fnv"
	"strconv"
	"testing"
	"time"
)

func TestWorkerConn_Init(t *testing.T) {
	var worker1, worker2 WorkerConn
	logger := log.GetLoggerByIndex(log.P2PLogConfig, "1")

	worker1.Init("39.104.113.9", []byte("1"), nil, logger)
	worker2.Init("39.104.113.9", []byte("2"), nil, logger)

}

func TestWorkerConn_Init2(t *testing.T) {
	var worker WorkerConn
	logger := log.GetLoggerByIndex(log.P2PLogConfig, "1")
	p2pLogger = log.GetLoggerByIndex(log.P2PLogConfig, "1")
	worker.Init("ws://192.168.2.19/phub", []byte("1"), nil, logger)
	time.Sleep(10 * time.Hour)
}

func TestGenTargetForgroup(t *testing.T) {
	groupId := "0x2a63497b8b48bc85ae6f61576d4a2988e7b71e1c02898ea2a02ead17f076bf92"
	hash64 := fnv.New64()
	hash64.Write([]byte(groupId))

	idInt := hash64.Sum64()
	fmt.Printf("hash:%v\n", idInt)

	b16 := strconv.FormatUint(idInt, 16) //10 yo 16
	fmt.Printf("hex:%v\n", b16)
}

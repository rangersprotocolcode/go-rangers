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
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware"
	"os"
	"testing"
	"time"
)

func TestTxConn_Init(t *testing.T) {
	os.RemoveAll("logs")
	os.RemoveAll("1.ini")
	os.RemoveAll("storage0")

	common.Init(0, "1.ini", "dev")
	middleware.InitMiddleware()

	var tx TxConn
	tx.Init("ws://192.168.2.14:8888")

	time.Sleep(10 * time.Hour)
}

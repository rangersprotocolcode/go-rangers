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

package statemachine

import (
	"com.tuntun.rocket/node/src/common"
	"testing"
)

func TestWsServer_Start(t *testing.T) {
	common.InitConf("/Users/daijia/go/src/x/deploy/daily/x1.ini")
	ws := newWSServer("wstest")
	err := ws.Start()
	if err != nil {
		t.Fatalf("fail to start ws. %s", err.Error())
	}

	//time.Sleep(1000 * time.Minute)
}

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

package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/vm"
	"fmt"
	"testing"
)

func TestBlockChain_GenerateHeightKey(t *testing.T) {
	result := generateHeightKey(10)
	fmt.Println(len(result))
	fmt.Printf("%v", result)
}

func TestBlockChain_Init(t *testing.T) {
	common.InitConf("1.ini")
	middleware.InitMiddleware()
	service.InitService()
	vm.InitVM()
	InitCore(nil)
}

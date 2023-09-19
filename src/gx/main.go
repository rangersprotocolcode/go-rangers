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

package main

import (
	"com.tuntun.rocket/node/src/gx/cli"
	"fmt"
	"runtime"
	"runtime/debug"
)

func main() {
	initSysParam()

	gx := cli.NewGX()
	gx.Run()

}

func initSysParam() {
	runtime.GOMAXPROCS(8)
	debug.SetGCPercent(30)
	debug.SetMaxStack(1 * 1000 * 1000 * 1000)

	fmt.Printf("Setting gc %s, max memory %s, maxproc %s\n", "50", "1g", "8")
}

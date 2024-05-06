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

package main

import (
	"com.tuntun.rangers/node/src/gx/cli"
	"fmt"
	_ "go.uber.org/automaxprocs"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"
)

func main() {
	initSysParam()
	setupStackTrap()
	gx := cli.NewGX()
	gx.Run()

}

func initSysParam() {
	debug.SetGCPercent(30)
	debug.SetMaxStack(1 * 1000 * 1000 * 1000)

	fmt.Printf("Setting gc %s, max memory %s, maxproc %s\n", "50", "1g", runtime.GOMAXPROCS(-1))
}

const (
	timeFormat = "2006-01-02 15:04:05"
)

var (
	stdFile = "./stack.log"
)

func setupStackTrap(args ...string) {
	if len(args) > 0 {
		stdFile = args[0]
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR1)
	go func() {
		for range c {
			dumpStacks()
		}
	}()
}

func dumpStacks() {
	buf := make([]byte, 1638400)
	buf = buf[:runtime.Stack(buf, true)]
	writeStack(buf)
}

func writeStack(buf []byte) {
	fd, _ := os.OpenFile(stdFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)

	now := time.Now().Format(timeFormat)
	fd.WriteString("\n\n\n\n\n")
	fd.WriteString(now + " stdout:" + "\n\n")
	fd.Write(buf)
	fd.Close()
}

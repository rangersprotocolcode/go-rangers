package main

import (
	"x/src/gx/cli"
	"runtime/debug"
	"runtime"
	"fmt"
)

func main() {
	initSysParam()
	gx := cli.NewGX()
	gx.Run()

}

func initSysParam() {
	runtime.GOMAXPROCS(8)
	debug.SetGCPercent(50)
	debug.SetMaxStack(1 * 1000 * 1000 * 1000)

	fmt.Printf("Setting gc %s, max memory %s, maxproc %s", "50", "1g", "8")
}

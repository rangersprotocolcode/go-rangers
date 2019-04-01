
package main

import (
	"runtime"
	"x/src/gx/cli"
)

func main() {
	runtime.GOMAXPROCS(4)
	gx := cli.NewGX()
	gx.Run()
}

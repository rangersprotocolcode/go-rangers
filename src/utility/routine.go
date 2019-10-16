package utility

import (
	"fmt"
	"runtime"
	"strings"
	"strconv"
)

func GetGoroutineId() int {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("panic recover:panic info:%v\n", err)
		}
	}()

	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
	}

	return id
}

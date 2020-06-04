package utility

import (
	"fmt"
	"testing"
	"time"
)

func TestQuery(t *testing.T) {
	offset := ntpOffset(true)
	fmt.Println(offset)

	fmt.Println(GetTime().UnixNano())
	fmt.Println(GetTime())
	fmt.Println(time.Now().UnixNano())
	fmt.Println(time.Now())
}

package utility

import (
	"fmt"
	"testing"
)

func TestPortInUse(t *testing.T) {
	fmt.Println(PortInUse(9001))
}

package utility

import (
	"testing"
	"fmt"
)

func TestPortInUse(t *testing.T) {
	fmt.Println(PortInUse(9001))
}

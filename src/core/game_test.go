package core

import (
	"fmt"
	"testing"
)

func TestSwtich(t *testing.T) {
	echo(1)
}

func echo(a int ){
	switch a {
	case 1:
		fmt.Println("1")
		return
	case 2:
		fmt.Println("2")
	case 3:
		fmt.Println("3")
	}
	fmt.Println("After case")
}
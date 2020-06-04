package core

import (
	"fmt"
	"testing"
)

func TestBlockChain_GenerateHeightKey(t *testing.T) {
	result := generateHeightKey(10)
	fmt.Println(len(result))
	fmt.Printf("%v",result)
}

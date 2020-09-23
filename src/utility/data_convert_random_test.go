package utility

import (
	"fmt"
	"testing"
)

func TestRandomStrToBigInt(t *testing.T) {
	str := "999999741.819500000"
	fmt.Println(str)
	bigInt, _ := StrToBigInt(str)
	fmt.Println(bigInt.String())
	fmt.Println(BigIntToStr(bigInt))
}

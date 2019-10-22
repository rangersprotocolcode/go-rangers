package types

import (
	"testing"
	"fmt"
	"math/big"
)

func TestJSONObject_Put(t *testing.T) {
	obj := NewJSONObject()
	obj.Put("1", "a")

	fmt.Println(obj.TOJSONString())
}

func TestJSONObject_Merge(t *testing.T) {
	obj := NewJSONObject()
	obj.Put("1", big.NewInt(10))
	fmt.Println(obj.TOJSONString())

	obj2 := NewJSONObject()
	obj2.Put("1", big.NewInt(100))
	fmt.Println(obj2.TOJSONString())

	obj.Merge(&obj2, ReplaceBigInt)
	fmt.Println(obj.TOJSONString())
	fmt.Println(obj2.TOJSONString())
}

func TestJSONObject_Put2(t *testing.T) {
	obj := NewJSONObject()
	obj.Put("1", big.NewInt(10))

	mobj := NewJSONObject()
	mobj.Put("ft",obj.TOJSONString())

	fmt.Println(mobj.TOJSONString())
	fmt.Println(mobj.Remove("ft"))
}


func TestJSONObject_Put3(t *testing.T) {
	obj := NewJSONObject()
	obj.Put("1", big.NewInt(10))

	mobj := NewJSONObject()
	mobj.Put("ft",obj)

	fmt.Println(mobj.TOJSONString())
	fmt.Println(mobj.Remove("ft"))
}

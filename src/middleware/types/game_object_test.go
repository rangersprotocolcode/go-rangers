package types

import (
	"testing"
	"math/big"
	"x/src/storage/rlp"
	"fmt"
)

func TestFT_EncodeRLP(t *testing.T) {
	ft := FT{
		ID:      "test1",
		Balance: big.NewInt(100),
	}

	data, err := rlp.EncodeToBytes(ft)
	if err != nil {
		t.Fatalf("%s", err.Error())
	}
	fmt.Println(data)

	ftMap := []FT{}
	ftMap = append(ftMap,ft)
	data, err = rlp.EncodeToBytes(ftMap)
	if err != nil {
		t.Fatalf("%s", err.Error())
	}
	fmt.Println(data)

}

func Test_RLP(t *testing.T) {
	s := Student{Name: "icattlecoder", Sex: "male"}

	data, err := rlp.EncodeToBytes(s)
	if err != nil {
		t.Fatalf("%s", err.Error())
	}

	fmt.Println(data)
}

type Student struct {
	Name string
	Sex  string
}

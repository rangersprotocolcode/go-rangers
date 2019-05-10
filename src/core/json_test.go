package core

import (
	"testing"
	"encoding/json"
	"fmt"
	"math/big"
)

func TestBigInt(t *testing.T) {
	bigInt := new(big.Int).SetUint64(111)
	str := bigInt.String()
	fmt.Printf("big int str:%s\n", str)

	bi, _ := new(big.Int).SetString(str, 10)
	bi = bi.Add(bi, new(big.Int).SetUint64(222))
	u := bi.Uint64()
	fmt.Printf("big int uint64:%d\n", u)
}

func TestStrToJson(t *testing.T) {
	a := []string{"1111", "222", "333"}
	b, err := json.Marshal(a)
	if err != nil {
		fmt.Printf("Json marshal []string err:%s", err.Error())
		return
	}
	str := string(b)
	fmt.Println(str)

	var c []string
	err = json.Unmarshal(b, &c)
	if err != nil {
		fmt.Printf("Json Unmarshal []string err:%s", err.Error())
	}
	fmt.Printf("C:%v\n", c)
}

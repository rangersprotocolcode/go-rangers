package core

import (
	"testing"
	"encoding/json"
	"fmt"
	"math/big"
)

func TestJson(t *testing.T){
	w := WithdrawInfo{
		Address: "fss",
		GameId: "fsfsaf",
	}

	b, err := json.Marshal(w)
	if err != nil {
		fmt.Printf("Json marshal withdrawInfo err:%s", err.Error())
		return
	}
	fmt.Printf("%v\n",b)
	fmt.Println(string(b))
}


func TestBigInt(t *testing.T){
	bigInt := new(big.Int).SetUint64(111)
	str := bigInt.String()
	fmt.Printf("big int str:%s\n",str)

	bi,_:= new(big.Int).SetString(str,10)
	bi = bi.Add(bi,new(big.Int).SetUint64(222))
	u := bi.Uint64()
	fmt.Printf("big int uint64:%d\n",u)
}

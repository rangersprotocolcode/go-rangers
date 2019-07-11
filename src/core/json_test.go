package core

import (
	"testing"
	"encoding/json"
	"fmt"
	"math/big"
	"x/src/common"
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

func TestResponse(t *testing.T) {
	res := response{
		Data:   "{\"status\":0,\"payload\":\"{\"code\":0,\"data\":{},\"status\":0}\"}",
		Hash:   "hash",
		Status: 0,
	}

	data, err := json.Marshal(res)
	if err != nil {
		fmt.Errorf(err.Error())
	}
	fmt.Printf("data:%s", data)
}


func TestFloatConvert(t *testing.T) {
	var str = "100000.0"
	b := convert(str)
	fmt.Printf("result:%v",b)
}

func TestAddr(t *testing.T){
	s:= "TAD5ZbvETHrNobKa41hGkCkB37jEXCEQss"
	addr := common.HexToAddress(s)
	fmt.Printf("Addr:%v",addr)
}

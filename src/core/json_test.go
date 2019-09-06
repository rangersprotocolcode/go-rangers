package core

import (
	"testing"
	"encoding/json"
	"fmt"
	"math/big"
	"x/src/common"
	"x/src/middleware/types"
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

func TestStrToJson1(t *testing.T) {
	//a := []string{"1111"}
	//b, err := json.Marshal(a)
	//if err != nil {
	//	fmt.Printf("Json marshal []string err:%s", err.Error())
	//	return
	//}
	//str := string(b)
	//fmt.Println(str)

	b := "[\"yyy\"]"
	var c []string
	err := json.Unmarshal([]byte(b), &c)
	if err != nil {
		fmt.Printf("Json Unmarshal []string err:%s", err.Error())
	}
	fmt.Printf("C:%v\n", c)
}

func TestResponse(t *testing.T) {
	res := response{
		Data:   "{\"status\":0,\"payload\":\"{\"code\":0,\"data\":{},\"status\":0}\"}",
		Id:     "hash",
		Status: "0",
	}

	data, err := json.Marshal(res)
	if err != nil {
		fmt.Errorf(err.Error())
	}
	fmt.Printf("data:%s", data)
}

func TestFloatConvert(t *testing.T) {
	var str = "-15000.0"
	b := convert(str)
	fmt.Printf("result:%v", b)
	fmt.Printf("result:%v", b.Sign())
}

func TestAddr(t *testing.T) {
	s := "TAD5ZbvETHrNobKa41hGkCkB37jEXCEQss"
	addr := common.HexToAddress(s)
	fmt.Printf("Addr:%v", addr)
}

func TestJSONTransferData(t *testing.T) {
	s := "{\"address1\":{\"balance\":\"127\",\"ft\":{\"name1\":\"189\",\"name2\":\"1\"},\"nft\":[\"id1\",\"sword2\"]}, \"address2\":{\"balance\":\"1\"}}"
	mm := make(map[string]types.TransferData, 0)
	if err := json.Unmarshal([]byte(s), &mm); nil != err {
		fmt.Errorf("fail to unmarshal")
	}

	fmt.Printf("length: %d\n", len(mm))
	fmt.Printf("length: %s", mm)
}

func TestJSONWithDepositData(t *testing.T) {
	w := types.DepositData{ChainType: "ETH", Amount: "12.56", TxId: "1213r43qr"}
	ft := make(map[string]string, 0)
	ft["ft1"] = "23.55"
	ft["ft2"] = "125.68"
	w.FT = ft

	nft := make(map[string]string, 0)
	nft["nft1"] = "dafjls;djfa"
	nft["nft2"] = "{'key':'v'}"
	w.NFT = nft

	b, err := json.Marshal(w)
	if err != nil {
		fmt.Printf("json marshal err: %s\n", err.Error())
	}
	fmt.Printf("marshal result:%s\n", b)

	s := "{\"chainType\":\"ETH\",\"amount\":\"12.56\",\"txId\":\"1213r43qr\",\"ft\":{\"ft1\":\"23.55\",\"ft2\":\"125.68\"},\"nft\":{\"nft1\":\"dafjls;djfa\",\"nft2\":\"{'key':'v'}\"}}"
	a := types.DepositData{}
	err1 := json.Unmarshal([]byte(s), &a)
	if err1 != nil {
		fmt.Printf("json unmarshal err: %s\n", err.Error())
	}
	fmt.Printf("unmarshal result:%v\n", a)
}

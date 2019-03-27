package beCall

import (
	"testing"
	"strings"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"math/big"
	"time"
)

func Test_Call(t *testing.T){
	var mod CallModule

	m1 := map[string]string{
		"eth": "C:\\Users\\Quake\\AppData\\Roaming\\Ethereum Wallet\\binaries\\Geth\\unpacked\\chaindata\\keystore\\UTC--2019-03-16T12-46-07.441268200Z--f89eebcc07e820f5a8330f52111fa51dd9dfb925",
		"ont": "bb",
	}
	mod.Init(m1)
	//TestABI:= "[ { \"constant\": false, \"inputs\": [ { \"name\": \"a\", \"type\": \"int256\" } ], \"name\": \"abc\", \"outputs\": [ { \"name\": \"\", \"type\": \"int256\" } ], \"payable\": true, \"stateMutability\": \"payable\", \"type\": \"function\", \"signature\": \"0x68aa6be9\" }, { \"anonymous\": false, \"inputs\": [ { \"indexed\": true, \"name\": \"a\", \"type\": \"int256\" }, { \"indexed\": false, \"name\": \"b\", \"type\": \"int256\" } ], \"name\": \"testevent\", \"type\": \"event\", \"signature\": \"0x7c79c89e6e351fd174b98f7ded2f376ee88c34b868fa22a2b32b07055ca2ba1c\" } ]"
	//TestABI:= "[ { \"constant\": false, \"inputs\": [ { \"name\": \"a\", \"type\": \"int256\" } ], \"name\": \"abc\", \"outputs\": [ { \"name\": \"\", \"type\": \"int256\" } ], \"payable\": true, \"stateMutability\": \"payable\", \"type\": \"function\", \"signature\": \"0x68aa6be9\" }, { \"payable\": true, \"stateMutability\": \"payable\", \"type\": \"fallback\" }, { \"anonymous\": false, \"inputs\": [ { \"indexed\": true, \"name\": \"a\", \"type\": \"int256\" }, { \"indexed\": false, \"name\": \"b\", \"type\": \"int256\" } ], \"name\": \"testevent\", \"type\": \"event\", \"signature\": \"0x7c79c89e6e351fd174b98f7ded2f376ee88c34b868fa22a2b32b07055ca2ba1c\" } ]"
	TestABI:= "[ { \"constant\": false, \"inputs\": [ { \"name\": \"a\", \"type\": \"int256\" } ], \"name\": \"abc\", \"outputs\": [ { \"name\": \"\", \"type\": \"int256\" } ], \"payable\": true, \"stateMutability\": \"payable\", \"type\": \"function\", \"signature\": \"0x68aa6be9\" }, { \"payable\": true, \"stateMutability\": \"payable\", \"type\": \"fallback\" }, { \"anonymous\": false, \"inputs\": [ { \"indexed\": false, \"name\": \"data\", \"type\": \"uint256\" } ], \"name\": \"FallbackCalled\", \"type\": \"event\", \"signature\": \"0xe313b135453e5deea84af41be832dcdd6d8b7109750349eb21c3737aa5ef024f\" }, { \"anonymous\": false, \"inputs\": [ { \"indexed\": true, \"name\": \"a\", \"type\": \"int256\" }, { \"indexed\": false, \"name\": \"b\", \"type\": \"int256\" } ], \"name\": \"testevent\", \"type\": \"event\", \"signature\": \"0x7c79c89e6e351fd174b98f7ded2f376ee88c34b868fa22a2b32b07055ca2ba1c\" } ]"
	testabi, err := abi.JSON(strings.NewReader(TestABI))
	if err != nil {

	}
	input,err :=testabi.Pack("abc",big.NewInt(5))
	if err != nil {

	}
	mod.Call("eth","0x2ae33A8904a189C7f81F45d2d08154F34d344a98",input)
	//
	//time.Sleep(time.Second*20)
	//mod.Call("eth","0x2ae33A8904a189C7f81F45d2d08154F34d344a98",input)

	time.Sleep(time.Second*2000)
}


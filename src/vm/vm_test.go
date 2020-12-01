package vm

import (
	"fmt"
	"math/big"
	"testing"
)

func TestVM0(t *testing.T) {
	mockInit()
	config := new(testConfig)
	setDefaults(config)

	config.GasLimit = 100
	config.GasPrice = big.NewInt(0)

	//Contract code:xxx
	contractCodeBytes := []byte{1, 1}
	createResult, contractAddress, createLeftGas, createErr := mockCreate(contractCodeBytes, config)
	fmt.Printf("New create contract address:%s,createResult:%v,createLeftGas:%v,createErr:%v", contractAddress.GetHexString(), createResult, createLeftGas, createErr)
	input := []byte{1, 1}
	callResult, callLeftGas, callErr := mockCall(contractAddress, input, config)
	fmt.Printf("callResult:%v,callLeftGas:%v,callErr:%v", callResult, callLeftGas, callErr)
	//stateDB := config.State
}

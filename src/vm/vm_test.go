package vm

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"fmt"
	"math/big"
	"testing"
)

func TestVM0(t *testing.T) {
	mockInit()
	config := new(testConfig)
	setDefaults(config)
	defer log.Close()

	config.GasLimit = 3000000
	config.GasPrice = big.NewInt(0)

	contractCodeBytes := common.Hex2Bytes("60806040526000805534801561001457600080fd5b5060ca806100236000396000f3fe6080604052348015600f57600080fd5b506004361060325760003560e01c80632beaa90f14603757806387db03b7146053575b600080fd5b603d607e565b6040518082815260200191505060405180910390f35b607c60048036036020811015606757600080fd5b81019080803590602001909291905050506087565b005b60008054905090565b80600054016000819055505056fea265627a7a72315820dd06e5c6466fd6226b1d4db2bfa83033ef4a09278a8cd2f8efde275770f29e8a64736f6c63430005110032")
	createResult, contractAddress, createLeftGas, createErr := mockCreate(contractCodeBytes, config)
	fmt.Printf("New create contract address:%s\n", contractAddress.GetHexString())
	fmt.Printf("New create contract createResult:%v\n", createResult)
	fmt.Printf("New create contract createLeftGas:%v,createErr:%v\n", createLeftGas, createErr)

	//invoke add
	input := common.Hex2Bytes("87db03b70000000000000000000000000000000000000000000000000000000000000003")
	callResult, callLeftGas, callErr := mockCall(contractAddress, input, config)
	fmt.Printf("callResult:%v,callLeftGas:%v,callErr:%v\n", callResult, callLeftGas, callErr)

	//invoke get
	input = common.Hex2Bytes("2beaa90f")
	callResult, callLeftGas, callErr = mockCall(contractAddress, input, config)
	fmt.Printf("callResult:%v,callLeftGas:%v,callErr:%v\n", callResult, callLeftGas, callErr)
	//stateDB := config.State
}

func TestVM1(t *testing.T) {
	mockInit()
	config := new(testConfig)
	setDefaults(config)
	defer log.Close()

	config.GasLimit = 3000000
	config.GasPrice = big.NewInt(1)

	contractCodeBytes := common.Hex2Bytes("60806040526000805534801561001457600080fd5b506101d0806100246000396000f3fe608060405234801561001057600080fd5b506004361061004c5760003560e01c80630d56562c146100515780632beaa90f146100955780635b3357d2146100b357806387db03b7146100d1575b600080fd5b6100936004803603602081101561006757600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff1690602001909291905050506100ff565b005b61009d610143565b6040518082815260200191505060405180910390f35b6100bb61014c565b6040518082815260200191505060405180910390f35b6100fd600480360360208110156100e757600080fd5b810190808035906020019092919050505061018d565b005b80600160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b60008054905090565b6000600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1631905090565b80600054016000819055505056fea265627a7a72315820296d0b6f5a2874c5859fd6bbe03bb7f5f0db8d8f4395d12f087b82288d0ba46964736f6c63430005110032")
	createResult, contractAddress, createLeftGas, createErr := mockCreate(contractCodeBytes, config)
	fmt.Printf("New create contract address:%s\n", contractAddress.GetHexString())
	fmt.Printf("New create contract createResult:%v,%d\n", createResult, len(createResult))
	fmt.Printf("New create contract costGas:%v,createErr:%v\n", config.GasLimit-createLeftGas, createErr)

	input := common.Hex2Bytes("0d56562c000000000000000000000000f89eebcc07e820f5a8330f52111fa51dd9dfb925")
	callResult, callLeftGas, callErr := mockCall(contractAddress, input, config)
	fmt.Printf("callResult:%v,costGas:%d,callErr:%v\n", callResult, config.GasLimit-callLeftGas, callErr)

	stateDB := config.State
	addr := common.HexToAddress("0xf89eebcc07e820f5a8330f52111fa51dd9dfb925")
	stateDB.SetBalance(addr, big.NewInt(5))

	input = common.Hex2Bytes("5b3357d2")
	callResult2, callLeftGas2, callErr2 := mockCall(contractAddress, input, config)
	fmt.Printf("callResult:%v,costGas:%v,callErr:%v\n", callResult2, config.GasLimit-callLeftGas2, callErr2)
}

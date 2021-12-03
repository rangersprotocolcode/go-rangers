package model

import (
	"com.tuntun.rocket/node/src/common"
	"fmt"
	"testing"
)

func TestNewSelfMinerInfo(t *testing.T) {
	pkString := "0x04d0e50343ed268e90413a39e84c9a02a26aaaabe945f5e138dc45cadd810d0c68f26eb00419a6c8f3858b70bb80dd50034546a45b8da2428cebbc2bef8c507b1799d6974cd1ae9ba9bc77d94981667366841f4c87c54331c0c8bab41f7a547738"
	privateKey := common.HexStringToSecKey(pkString)
	miner := NewSelfMinerInfo(*privateKey)
	fmt.Println(miner.MinerInfo.PubKey.GetHexString())

}

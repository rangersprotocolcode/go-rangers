package model

import (
	"com.tuntun.rocket/node/src/common"
	"fmt"
	"testing"
)

func TestNewSelfMinerInfo(t *testing.T) {
	pkString := "0x0499e4e44fdfd2848d274542c4255cea7380dc4d455081dcdc9e7fce2e3d60bc1f6f508b872203ac66b98e4c64bc831152a5503dc9d947e91d0de8e528a50a287ad1e3de87f825c95998ec6cbbe4e7542c91a609929197c29500dfbe74bf9021b4"
	privateKey := common.HexStringToSecKey(pkString)
	miner := NewSelfMinerInfo(*privateKey)
	fmt.Println(miner.MinerInfo.ID.GetHexString())
	fmt.Println(miner.MinerInfo.PubKey.GetHexString())
	fmt.Println(miner.MinerInfo.VrfPK.GetHexString())
	fmt.Println(miner.MinerInfo.VrfPK)

}

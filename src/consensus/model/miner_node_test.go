package model

import (
	"com.tuntun.rocket/node/src/common"
	"fmt"
	"testing"
)

func TestNewSelfMinerInfo(t *testing.T) {
	pkString := "0x0411c6362e7ece1fcbbe98085bcc8410749c2bebf546da7d7406e1e9a20afdabe29eb6826e1ea25c3d0e8c2db953661d5e37c1f5f71dc8fbd5d0b7422e3f0aeff320f7f665561eae7693ade2f9592530c2f9b67b1d6cc2897b0c71b7d7b8d02a3a"
	privateKey := common.HexStringToSecKey(pkString)
	miner := NewSelfMinerInfo(*privateKey)
	fmt.Println(miner)

}

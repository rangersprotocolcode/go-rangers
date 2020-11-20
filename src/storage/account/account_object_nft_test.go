package account

import (
	"com.tuntun.rocket/node/src/common"
	"fmt"
	"testing"
)

func TestAccountObject_AddNFT(t *testing.T) {
	bytes := common.Hex2Bytes("0xe7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea")
	address2 := common.HexToAddress("0xe7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea")
	fmt.Println(address2.String())
	fmt.Println(common.BytesToAddress(bytes).String())
	bytes=common.Hex2Bytes("0123a")
	fmt.Println(bytes)
}

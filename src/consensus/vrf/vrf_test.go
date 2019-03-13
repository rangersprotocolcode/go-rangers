package vrf

import (
	"testing"
	"math/big"
	"fmt"

	"x/src/consensus/base"
)

func TestVRF_prove(t *testing.T) {
	//total := new(big.Int)
	pk, sk, _ := VRF_GenerateKey(nil)
	for i := 0; i < 1000000000; i ++ {
		pi, _ := VRF_prove(pk, sk, base.NewRand().Bytes())
		bi := new(big.Int).SetBytes(pi)
		if bi.Sign() < 0 {
			fmt.Errorf("error bi %d", bi )
			break
		}
		//total = total.Add(total, bi)
		//if total.Sign() < 0 {
		//	fmt.Errorf("error total %d", total)
		//	break
		//}
		if i%10000 == 0 {
			fmt.Printf("%d total: %d\n", i, bi)
		}
	}
}
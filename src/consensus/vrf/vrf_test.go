package vrf

import (
	"testing"
	"fmt"

	"x/src/consensus/base"
)

func TestVRF_prove(t *testing.T) {
	pk, sk, _ := VRFGenerateKey(nil)
	msg := base.NewRand().Bytes()
	prove, _ := VRFGenProve(pk, sk, msg)
	result, _ := VRFVerify(pk, prove, msg)
	fmt.Printf("VRF verify result:%t\n", result)
}

package core

import (
	"testing"
	"x/src/middleware/types"
	"math/big"
)

func TestTransferNFT(t *testing.T) {
	nft := make([]string, 0)
	nft = append(nft, "1")
	source := &types.SubAccount{}
	source.Assets = make(map[string]string)
	target := &types.SubAccount{}
	target.Assets = make(map[string]string)

	if transferNFT(nft, source, target) {
		t.Errorf("fail to transfer1")
	}

	source.Assets["1"] = "test"
	if !transferNFT(nft, source, target) {
		t.Errorf("fail to transfer2")
	}

	if 0 != len(source.Assets) {
		t.Errorf("fail to transfer3")
	}

	if 1 != len(target.Assets) {
		t.Errorf("fail to transfer4")
	}

	if "test" != target.Assets["1"] {
		t.Errorf("fail to transfer5")
	}
}

func TestTransferNFT2(t *testing.T) {
	nft := make([]string, 0)
	nft = append(nft, "1")
	source := &types.SubAccount{}
	source.Assets = make(map[string]string)
	target := &types.SubAccount{}
	target.Assets = make(map[string]string)

	source.Assets["1"] = "test"
	source.Assets["a"] = "b"

	if !transferNFT(nft, source, target) {
		t.Errorf("fail to transfer2")
	}

	if 1 != len(source.Assets) {
		t.Errorf("fail to transfer3")
	}

	if 1 != len(target.Assets) {
		t.Errorf("fail to transfer4")
	}

	if "test" != target.Assets["1"] {
		t.Errorf("fail to transfer5")
	}
	if "b" != source.Assets["a"] {
		t.Errorf("fail to transfer6")
	}
}

func TestTransferBalance(t *testing.T) {
	value := "1"
	source := &types.SubAccount{}
	source.Balance = big.NewInt(0)
	target := &types.SubAccount{}
	target.Balance = big.NewInt(0)

	if transferBalance(value, source, target) {
		t.Errorf("fail to transfer 1")
	}

	source.Balance = big.NewInt(1000000000)
	if !transferBalance(value, source, target) {
		t.Errorf("fail to transfer 2")
	}

	if source.Balance.Sign() != 0 {
		t.Errorf("fail to transfer 3")
	}

	if 0 != target.Balance.Cmp(big.NewInt(1000000000)) {
		t.Errorf("fail to transfer 4")
	}
}

func TestTransferBalance2(t *testing.T) {
	value := "1"
	source := &types.SubAccount{}
	source.Balance = big.NewInt(0)
	target := &types.SubAccount{}
	target.Balance = big.NewInt(0)

	if transferBalance(value, source, target) {
		t.Errorf("fail to transfer 1")
	}

	source.Balance = big.NewInt(1000000001)
	if !transferBalance(value, source, target) {
		t.Errorf("fail to transfer 2")
	}

	if 0 != source.Balance.Cmp(big.NewInt(1)) {
		t.Errorf("fail to transfer 3")
	}

	if 0 != target.Balance.Cmp(big.NewInt(1000000000)) {
		t.Errorf("fail to transfer 4")
	}
}

func TestTransferFT(t *testing.T){
	source := &types.SubAccount{}
	target := &types.SubAccount{}

	source.Ft = make(map[string]string)
	target.Ft = make(map[string]string)

	ft:= make(map[string]string)

	if !transferFT(ft,source,target){
		t.Errorf("fail to transfer ft 1")
	}

	ft["jifen"]="1"
	if transferFT(ft,source,target){
		t.Errorf("fail to transfer ft 2")
	}

	source.Ft["jifen"]="3000000000"
	if !transferFT(ft,source,target){
		t.Errorf("fail to transfer ft 3")
	}

	if source.Ft["jifen"]!="2000000000"{
		t.Errorf("fail to transfer ft 4")
	}
	if target.Ft["jifen"]!="1000000000"{
		t.Errorf("fail to transfer ft 5")
	}
}
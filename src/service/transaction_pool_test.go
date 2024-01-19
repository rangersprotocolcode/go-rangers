// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package service

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware/types"
	"encoding/json"
	"fmt"
	"github.com/gogf/gf/container/gmap"
	"testing"
)

func TestRequestId(t *testing.T) {
	s := "0x41ed2348bb544cb9e54ed6405e930ac7164e57f4cc59f6fe33f0ba84452d9bc550d31be232410a890618f3b628e2ee5a6e679581c6efed3d31ad07d4dd2398e000"
	sign := common.HexStringToSign(s)
	fmt.Println(sign.Bytes())
	fmt.Println(sign.GetR())
	fmt.Println(sign.GetS())
	fmt.Println(sign.GetHexString())
}

func TestSlice(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6}
	fmt.Println(data)

	fmt.Println(data[2:])
}

func TestGMap(t *testing.T) {
	listMap := gmap.NewListMap(true)

	listMap.Set("1", "a")
	listMap.Set("3", "b")
	listMap.Set("2", "c")
	listMap.Set("5", "d")
	listMap.Set("4", "e")

	fmt.Println(listMap.Size())
	fmt.Println(listMap.Keys())
	fmt.Println(listMap.Values())
}

func TestTxUnmarshal(t *testing.T) {
	missingTx1 := `{"source":"0x18cd99cdc57f5f21442baf5d06bcf5176e463e91","target":"","type":2,"time":"2024-01-19 17:47:34.334308011 +0800 CST m=+707184.175867286","data":"{\"id\":\"0x7a0b65d8e9fd5420a80f2c41cbd7ec20a2298f756e4259da0e4a9c08e8f224a2\",\"publicKey\":\"0x1b33e4f89a3ddf3712e2061c813b93324c46b8715e945b23068a5a03fa06bc597d217d165a3792397f5b0bf4aa584d8c0c5edeb32e8a4b4ede313593ef8653f6375c44896bd6d3d01d379fc06d0ac3b9dd790879b9afa0de47e115bc84b8936665e920fb837d5d978a131c63ee0cde38f67c60ed94c9045f9424195c4f6c69eb\",\"vrfPublicKey\":\"eDkuGhv7pG7Utlw/GAkZk/KacSjPj2R20UBwoisqKzI=\",\"ApplyHeight\":0,\"Status\":0,\"stake\":400,\"account\":\"0x18cd99cdc57f5f21442baf5d06bcf5176e463e91\"}","hash":"0x8ae1c35d909cf45ae136dd8f683a88d5cea87b1ef187f716ad90d29abd540159","RequestId":0,"chainId":"2025","sign":"0xe8fb3f557341a7c1e37c197d7dc2fdb8a9a56b50679ca12d33774010ea6e47cd3e60e383d78698132a3fc1d0152c6d74f692f53edd5bae5d6fb7cc57aa2ae8851c"}`
	var tx types.Transaction
	json.Unmarshal([]byte(missingTx1), &tx)
	return
}

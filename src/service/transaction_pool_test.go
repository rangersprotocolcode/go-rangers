// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package service

import (
	"com.tuntun.rocket/node/src/common"
	"fmt"
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

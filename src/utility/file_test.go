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

package utility

import (
	"fmt"
	"testing"
)

func TestMd5SumFolder(t *testing.T) {
	result, err := checkFolderDetail("/Users/daijia/go/src/com.tuntun.rocket/node/src/statemachine", 10)
	if err != nil {
		t.Fatal(err)
	}

	for key, value := range result {
		fmt.Printf("%s, %v\n", key, value)
	}

}

func TestCheckFolder(t *testing.T) {
	result, detail := CheckFolder("./")
	fmt.Printf("%v\n", result)

	fmt.Println("details:")
	for i, item := range detail {
		fmt.Printf("%d %v \n", i, item)
	}

}

func TestZip(t *testing.T) {
	err := Zip("/Users/daijia/go/src/com.tuntun.rocket/node/src/statemachine/logs", "1111.zip")
	if err != nil {
		panic(err)
	}

}

func TestUnzip(t *testing.T) {
	Unzip("1111.zip", "")
}

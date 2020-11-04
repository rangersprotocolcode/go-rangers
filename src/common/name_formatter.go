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

package common

import (
	"com.tuntun.rocket/node/src/utility"
	"fmt"
	"strings"
)

const (
	FTPrefix     = "f:"
	NFTPrefix    = "n:"
	NFTSetPrefix = "ns:"
)

func GenerateNFTAddress(setId, id string) Address {
	return BytesToAddress(Sha256(utility.StrToBytes(GenerateNFTKey(setId, id))))
}

func GenerateNFTKey(setId, id string) string {
	return fmt.Sprintf("%s%s:%s", NFTPrefix, setId, id)
}

func SplitNFTKey(key string) (string, string) {
	idList := strings.Split(key, ":")
	if 3 != len(idList) {
		return "", ""
	}

	return idList[1], idList[2]
}

func GenerateNFTSetAddress(setId string) Address {
	addr := fmt.Sprintf("%s%s", NFTSetPrefix, setId)
	return BytesToAddress(Sha256(utility.StrToBytes(addr)))
}

func GenerateFTKey(name string) string {
	return fmt.Sprintf("%s%s", FTPrefix, name)
}

func GenerateAppIdProperty(appId, property string) string {
	return fmt.Sprintf("%s:%s", appId, property)
}

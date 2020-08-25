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
	"fmt"
	"golang.org/x/crypto/sha3"
	"testing"
)

func TestSha256(t *testing.T) {
	hash := sha3.Sum256([]byte{})
	fmt.Println(Bytes2Hex(hash[:]))

	hash2 := sha3.Sum256([]byte("test"))
	fmt.Println(Bytes2Hex(hash2[:]))
}

func TestID(t *testing.T) {
	pubkey := HexStringToPubKey("0x04084b31e8fe11bf53acbabb035c31b5f7af35e78441b48b4b749cc588a560a67fe0b3909dee8b5302dcab0e1ad0af5bcc2bc360f2add174bfa3e5b9d39bf319f5")
	fmt.Printf(pubkey.GetAddress().String())
}

//privateKey:b14bd529fd9707271d31d4ff6329d556186a3000ae30a75015d92392bcd546d9|publicKey:0x04bdc3aff18404ed172aa660a8fcace24362be6f2376b2ee3e98447084e200e79419e555a8e13b64d1ed3711b18458aa931ce19b614f9ec60370ff9978eaf904eb|message:92767d34410fe5e8b450be0a6b2776cef82ce47887df274214679893099ccfee|sign:0xcfeaebea4bd8891e071b13cc0f23d674db5786c7d758f7440e33af26f831146f117b93839a6bfb26406cf445c71061dae7f2472f51a6e67fd949cd745057db561c|id:0x60f686a8f4817675c783bb38be2975773f082a4e2291b5bfc36281deda956616

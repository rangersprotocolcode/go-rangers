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

package types

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/common/secp256k1"
	"com.tuntun.rocket/node/src/common/sha3"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
)

var CoinerSignVerifier Ecc

type Ecc struct {
	SignLimit int
	Privkey   string

	Whitelist []string
}

type W2cBasic struct {
	Source string `json:"Source"`
	Target string `json:"Target"`
	Type   int    `json:"Type"`
	Data   string `json:"Data"`
	Hash   string `json:"Hash"`
	Sign   string `json:"Sign"`
}

type C2wDeposit struct {
	ChainType string `json:"ChainType"`
	Amount    string `json:"Amount"`
	Addr      string `json:"Addr"`
	TxID      string `json:"TxId"`
}

type C2wDepositNft struct {
	SetID        string            `json:"setId"`
	Name         string            `json:"name"`
	Symbol       string            `json:"symbol"`
	ID           string            `json:"id"`
	Creator      string            `json:"creator"`
	CreateTime   string            `json:"createTime"`
	Owner        string            `json:"owner"`
	Renter       string            `json:"renter"`
	Status       byte              `json:"status"`
	Condition    byte              `json:"condition"`
	AppID        string            `json:"appId"`
	Data         map[string]string `json:"data"`
	Addr         string            `json:"addr"`
	ContractAddr string            `json:"contractaddr"`
	TxID         string            `json:"TxId"`
}

type C2wDepositFt struct {
	FtID         string `json:"FtId"`
	Amount       string `json:"Amount"`
	Addr         string `json:"Addr"`
	ContractAddr string `json:"Contractaddr"`
	TxID         string `json:"TxId"`
}

type Incoming struct {
	Tp string
	//Gid		string
	Uid    string
	Amount string
	Addr   string
	Txid   string
}
type IncomingFt struct {
	//Tp		string

	Userid       string
	Amount       string
	Name         string
	Addr         string
	ContractAddr string
	Txid         string
}
type IncomingNft struct {
	//Tp		string

	//From	string
	//To		string
	//TokenId *big.Int
	TokenId      string
	Gameid       string
	Userid       string
	Setid        string
	Symbol       string
	Name         string
	Creator      string
	CreateTime   string
	Info         map[string]string
	Renter       string
	Status       byte
	Condition    byte
	Addr         string
	ContractAddr string
	Txid         string
}

func (self *Ecc) Verify(info []byte, signed []byte) bool {

	msg := common.Sha256(info)
	//fmt.Println(common.Bytes2Hex(msg))

	var sign *common.Sign = common.BytesToSign(signed)
	pubk, err := sign.RecoverPubkey(msg)
	if err != nil {
		fmt.Println("Verify sign.RecoverPubkey err:", err, signed)
		return false
	}

	//addr := pubk.GetAddress().String()
	addr := PubkeyToAddress(pubk.PubKey)
	var finded bool = false
	for _, v := range self.Whitelist {
		if strings.ToLower(common.ToHex(addr)) == strings.ToLower(v) {
			finded = true
			break
		}
	}
	if !finded {
		return false
	}

	return pubk.Verify(msg, sign)

}

func (self *Ecc) VerifyDeposit(msg TxJson) bool {
	if msg.Type == 201 {
		var de C2wDeposit
		err := json.Unmarshal([]byte(msg.Data), &de)
		if err != nil {
			fmt.Println(err)
			return false
		}

		var info Incoming = Incoming{de.ChainType, msg.Source, de.Amount, de.Addr, de.TxID}
		signstrs := strings.Split(msg.Sign, "|")
		if len(signstrs) < self.SignLimit {
			return false
		}

		var signeds []string
		var iCount = 0
		for i := 0; i < len(signstrs); i++ {
			if signstrs[i][0:2] == "0x" || signstrs[i][0:2] == "0X" {
				signstrs[i] = signstrs[i][2:]
			}

			found := false
			for j := 0; j < len(signeds); j++ {
				if signstrs[i] == signeds[j] {
					found = true
					break
				}
			}

			if found {
				continue
			} else {
				signeds = append(signeds, signstrs[i])
			}

			sign := common.Hex2Bytes(signstrs[i])
			if self.Verify(info.ToJson(), sign) {
				iCount++
			}
		}

		if iCount >= self.SignLimit {
			return true
		}
	} else if msg.Type == 202 {
		var de C2wDepositFt
		err := json.Unmarshal([]byte(msg.Data), &de)
		if err != nil {
			fmt.Println(err)
			return false
		}

		var info IncomingFt = IncomingFt{msg.Source, de.Amount, de.FtID, de.Addr, de.ContractAddr, de.TxID}
		signstrs := strings.Split(msg.Sign, "|")
		if len(signstrs) < self.SignLimit {
			return false
		}

		var signeds []string
		var iCount = 0
		for i := 0; i < len(signstrs); i++ {
			if signstrs[i][0:2] == "0x" || signstrs[i][0:2] == "0X" {
				signstrs[i] = signstrs[i][2:]
			}

			found := false
			for j := 0; j < len(signeds); j++ {
				if signstrs[i] == signeds[j] {
					found = true
					break
				}
			}
			if found {
				continue
			} else {
				signeds = append(signeds, signstrs[i])
			}

			sign := common.Hex2Bytes(signstrs[i])
			if self.Verify(info.ToJson(), sign) {
				iCount++
			}
		}
		if iCount >= self.SignLimit {
			return true
		}
	} else if msg.Type == 203 {
		var de C2wDepositNft
		err := json.Unmarshal([]byte(msg.Data), &de)
		if err != nil {
			fmt.Println(err)
			return false
		}

		var info IncomingNft = IncomingNft{de.ID, msg.Target, msg.Source, de.SetID, de.Symbol, de.Name, de.Creator, de.CreateTime, de.Data, de.Renter, de.Status, de.Condition, de.Addr, de.ContractAddr, de.TxID}
		signstrs := strings.Split(msg.Sign, "|")
		if len(signstrs) < self.SignLimit {
			return false
		}

		var signeds []string
		var iCount = 0
		for i := 0; i < len(signstrs); i++ {
			if signstrs[i][0:2] == "0x" || signstrs[i][0:2] == "0X" {
				signstrs[i] = signstrs[i][2:]
			}

			found := false
			for j := 0; j < len(signeds); j++ {
				if signstrs[i] == signeds[j] {
					found = true
					break
				}
			}
			if found {
				continue
			} else {
				signeds = append(signeds, signstrs[i])
			}

			sign := common.Hex2Bytes(signstrs[i])
			if self.Verify(info.ToJson(), sign) {
				iCount++
			}
		}
		if iCount >= self.SignLimit {
			return true
		}
	} else if msg.Type == 204 {
		// todo
	}

	return false
}

func (self *Ecc) Sign(info []byte) (ret []byte) {
	privateKey := self.privKeyFromHex(self.Privkey)
	msg := common.Sha256(info)

	sign := privateKey.Sign([]byte(msg))
	ret = sign.Bytes()

	privateKey.GetPubKey().GetAddress()
	return
}

func (self *Ecc) privKeyFromHex(hexstr string) (ret common.PrivateKey) {
	c := secp256k1.S256()
	var k *big.Int

	k = new(big.Int).SetBytes(common.Hex2Bytes(hexstr))

	key := new(ecdsa.PrivateKey)
	key.PublicKey.Curve = c
	key.D = k
	key.PublicKey.X, key.PublicKey.Y = c.ScalarBaseMult(k.Bytes())

	ret.PrivKey = *key
	return
}

func FromECDSAPub(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.Marshal(secp256k1.S256(), pub.X, pub.Y)
}
func Keccak256(data ...[]byte) []byte {
	//d := sha3.NewLegacyKeccak256()
	d := sha3.NewKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}
func PubkeyToAddress(p ecdsa.PublicKey) []byte {
	pubBytes := FromECDSAPub(&p)
	return Keccak256(pubBytes[1:])[12:]
}

func (self *Incoming) ToJson() []byte {
	js, err := json.Marshal(self)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return js
}

func (self *IncomingNft) ToJson() []byte {
	js, err := json.Marshal(self)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return js
}

func (self *IncomingFt) ToJson() []byte {
	js, err := json.Marshal(self)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return js
}

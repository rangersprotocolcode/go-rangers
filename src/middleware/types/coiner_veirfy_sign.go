package types

import (
	"fmt"
	"encoding/json"
	"math/big"
	"crypto/ecdsa"
	"x/src/common"
	"x/src/common/secp256k1"
	"strings"
	"crypto/elliptic"
	"x/src/storage/sha3"
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
	TxID      string `json:"TxId"`
}

type C2wDepositNft struct {
	SetID      string `json:"SetId"`
	Name       string `json:"Name"`
	Symbol     string `json:"Symbol"`
	ID         string `json:"ID"`
	Creator    string `json:"Creator"`
	CreateTime string `json:"CreateTime"`
	Owner      string `json:"Owner"`
	Value      string `json:"Value"`
	TxID       string `json:"TxId"`
}

type C2wDepositFt struct {
	FtID   string `json:"FtId"`
	Amount string `json:"Amount"`
	TxID   string `json:"TxId"`
}

type Incoming struct {
	Tp string
	//Gid		string
	Uid    string
	Amount string
	Txid   string
}
type IncomingFt struct {
	//Tp		string

	Userid string
	Amount string
	Name   string
	Txid   string
}
type IncomingNft struct {
	//Tp		string

	//From	string
	//To		string
	//TokenId *big.Int
	TokenId    string
	Gameid     string
	Userid     string
	Setid      string
	Symbol     string
	Name       string
	Creator    string
	CreateTime string
	Info       string
	Txid       string
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
		if strings.ToLower(addr.String()) == strings.ToLower(v) {
			finded = true
			break
		}
	}
	if finded == false {
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

		var info Incoming = Incoming{de.ChainType, msg.Source, de.Amount, de.TxID}
		signstrs := strings.Split(msg.Sign, "|")
		if len(signstrs) < self.SignLimit {
			return false
		}

		var iCount = 0
		for i := 0; i < len(signstrs); i++ {
			if signstrs[i][0:2] == "0x" || signstrs[i][0:2] == "0X" {
				signstrs[i] = signstrs[i][2:]
			}
			sign := common.Hex2Bytes(signstrs[i])
			//sign,err := hexutil.Decode(signstrs[i])
			//if err != nil{
			//	continue
			//}
			if (self.Verify(info.ToJson(), sign) == true) {
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

		var info IncomingFt = IncomingFt{msg.Source, de.Amount, de.FtID, de.TxID}
		signstrs := strings.Split(msg.Sign, "|")
		if len(signstrs) < self.SignLimit {
			return false
		}

		var iCount = 0
		for i := 0; i < len(signstrs); i++ {
			if signstrs[i][0:2] == "0x" || signstrs[i][0:2] == "0X" {
				signstrs[i] = signstrs[i][2:]
			}
			sign := common.Hex2Bytes(signstrs[i])
			//sign,err := hexutil.Decode(signstrs[i])
			//if err != nil{
			//	continue
			//}
			if (self.Verify(info.ToJson(), sign) == true) {
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

		var info IncomingNft = IncomingNft{de.ID, msg.Target, msg.Source, de.SetID, de.Symbol, de.Name, de.Creator, de.CreateTime, de.Value, de.TxID}
		signstrs := strings.Split(msg.Sign, "|")
		if len(signstrs) < self.SignLimit {
			return false
		}

		var iCount = 0
		for i := 0; i < len(signstrs); i++ {
			if signstrs[i][0:2] == "0x" || signstrs[i][0:2] == "0X" {
				signstrs[i] = signstrs[i][2:]
			}
			sign := common.Hex2Bytes(signstrs[i])
			//sign,err := hexutil.Decode(signstrs[i])
			//if err != nil{
			//	continue
			//}
			if (self.Verify(info.ToJson(), sign) == true) {
				iCount++
			}
		}
		if iCount >= self.SignLimit {
			return true
		}
	}
	return false
}

func (self *Ecc) Sign(info []byte) (ret []byte) {
	privateKey := self.privKeyFromHex(self.Privkey)
	msg := common.Sha256(info)
	//fmt.Println(common.Bytes2Hex(msg))
	sign := privateKey.Sign([]byte(msg))
	ret = sign.Bytes()

	privateKey.GetPubKey().GetAddress()
	return

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
func PubkeyToAddress(p ecdsa.PublicKey) common.Address {
	pubBytes := FromECDSAPub(&p)
	return common.BytesToAddress(Keccak256(pubBytes[1:])[12:])
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

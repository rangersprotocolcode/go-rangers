package dkim

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"golang.org/x/crypto/sha3"
	"math/big"
)

var (
	positionBytes = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	one           = big.NewInt(1)
)

func GetEmailPubKey(address common.Address, key string, db *account.AccountDB) string {
	start := getStringKey(key)
	length := db.GetData(address, start)
	if nil == length {
		return ""
	}

	result := make([]byte, 0)
	size := (new(big.Int).SetBytes(length).Uint64() - 1) / 64
	num := new(big.Int)
	num.SetBytes(keccakHash(start))
	for i := 0; i < int(size); i++ {
		item := db.GetData(address, num.Bytes())
		result = append(result, item...)
		num = num.Add(num, one)
	}

	return utility.BytesToStr(result)
}

func getStringKey(key string) []byte {
	keyBytes := utility.StrToBytes(key)
	data := append(keyBytes, positionBytes...)

	return keccakHash(data)
}

func keccakHash(data []byte) []byte {
	hasher2 := sha3.NewLegacyKeccak256().(common.KeccakState)
	hasher2.Write(data[:])
	result := [32]byte{}
	hasher2.Read(result[:])

	return result[:]
}

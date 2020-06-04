package types

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"reflect"
)

type bytesBacked interface {
	Bytes() []byte
}

var bloomT = reflect.TypeOf(Bloom{})

const (
	BloomByteLength = 256
)

type Bloom [BloomByteLength]byte

// BytesToBloom converts a byte slice to a bloom filter.
// It panics if b is not of suitable size.
func BytesToBloom(b []byte) Bloom {
	var bloom Bloom
	bloom.SetBytes(b)
	return bloom
}

func (b *Bloom) SetBytes(d []byte) {
	if len(b) < len(d) {
		panic(fmt.Sprintf("bloom bytes too big %d %d", len(b), len(d)))
	}
	copy(b[BloomByteLength-len(d):], d)
}

// Add adds d to the filter. Future calls of Test(d) will return true.
func (b *Bloom) Add(d *big.Int) {
	bin := new(big.Int).SetBytes(b[:])
	bin.Or(bin, bloom9(d.Bytes()))
	b.SetBytes(bin.Bytes())
}

// Big converts b to a big integer.
func (b Bloom) Big() *big.Int {
	return new(big.Int).SetBytes(b[:])
}

func (b Bloom) Bytes() []byte {
	return b[:]
}

func (b Bloom) Test(test *big.Int) bool {
	return BloomLookup(b, test)
}

func (b Bloom) TestBytes(test []byte) bool {
	return b.Test(new(big.Int).SetBytes(test))

}

//func CreateBloom(receipts Receipts) Bloom {
//	bin := new(big.Int)
//	for _, receipt := range receipts {
//		bin.Or(bin, LogsBloom(receipt.Logs))
//	}
//
//	return BytesToBloom(bin.Bytes())
//}
//
//func LogsBloom(logs []*Log) *big.Int {
//	bin := new(big.Int)
//	for _, log := range logs {
//		bin.Or(bin, bloom9(log.Address.Bytes()))
//		for _, b := range log.Topics {
//			bin.Or(bin, bloom9(b[:]))
//		}
//	}
//
//	return bin
//}

func bloom9(b []byte) *big.Int {
	bi := sha256.Sum256(b[:])

	r := new(big.Int)

	for i := 0; i < 6; i += 2 {
		t := big.NewInt(1)
		b := (uint(bi[i+1]) + (uint(bi[i]) << 8)) & 2047
		r.Or(r, t.Lsh(t, b))
	}

	return r
}

var Bloom9 = bloom9

func BloomLookup(bin Bloom, topic bytesBacked) bool {
	bloom := bin.Big()
	cmp := bloom9(topic.Bytes()[:])

	return bloom.And(bloom, cmp).Cmp(cmp) == 0
}

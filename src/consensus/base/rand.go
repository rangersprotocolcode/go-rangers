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

package base

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"strconv"
)

const RandLength = 32

type Rand [RandLength]byte

func RandFromBytes(b ...[]byte) (r Rand) {
	HashBytes(b...).Sum(r[:0])
	return
}

func NewRand() (r Rand) {
	b := make([]byte, RandLength)
	rand.Read(b)
	return RandFromBytes(b)
}

func RandFromHex(s ...string) (r Rand) {
	return RandFromBytes(mapHexToBytes(s)...)
}

func RandFromString(s string) (r Rand) {
	return RandFromBytes([]byte(s))
}

func (r Rand) Bytes() []byte {
	return r[:]
}

func (r Rand) GetHexString() string {
	return hex.EncodeToString(r[:])
}

func (r Rand) DerivedRand(x ...[]byte) Rand {
	ri := r
	for _, xi := range x {
		HashBytes(ri.Bytes(), xi).Sum(ri[:0])
	}
	return ri
}

func (r Rand) Ders(s ...string) Rand {
	return r.DerivedRand(mapStringToBytes(s)...)
}

func (r Rand) Deri(vi ...int) Rand {
	return r.Ders(mapItoa(vi)...)
}

func (r Rand) Modulo(n int) int {
	b := big.NewInt(0)
	b.SetBytes(r.Bytes())
	b.Mod(b, big.NewInt(int64(n)))
	return int(b.Int64())
}

func (r Rand) ModuloUint64(n uint64) uint64 {
	b := big.NewInt(0)
	b.SetBytes(r.Bytes())
	b.Mod(b, big.NewInt(0).SetUint64(n))
	return b.Uint64()
}

func (r Rand) RandomPerm(n int, k int) []int {
	l := make([]int, n)
	for i := range l {
		l[i] = i
	}
	for i := 0; i < k; i++ {
		j := r.Deri(i).Modulo(n-i) + i
		l[i], l[j] = l[j], l[i]
	}
	return l[:k]
}

func NewFromSeed(seed []byte) *big.Int {
	_, err := rand.Read(seed)
	if err != nil {
		return nil
	}
	var bb = make([]byte, 32)
	for i := 0; i < 32; i++ {
		bb[i] = 0xff
	}
	max := new(big.Int)
	max.SetBytes(bb)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return nil
	}
	return n
}

func mapHexToBytes(x []string) [][]byte {
	y := make([][]byte, len(x))
	for k, xi := range x {
		y[k], _ = hex.DecodeString(xi)
	}
	return y
}

func mapStringToBytes(x []string) [][]byte {
	y := make([][]byte, len(x))
	for k, xi := range x {
		y[k] = []byte(xi)
	}
	return y
}

func mapItoa(x []int) []string {
	y := make([]string, len(x))
	for k, xi := range x {
		y[k] = strconv.Itoa(xi)
	}
	return y
}

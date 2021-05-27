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
	"com.tuntun.rocket/node/src/common/secp256k1"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/utility"
	"crypto/elliptic"
	"encoding/hex"
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
)

func getDefaultCurve() elliptic.Curve {
	return secp256k1.S256()
}

var DefaultLogger log.Logger
var InstanceIndex int

//160位地址
type Address [AddressLength]byte

func (a Address) MarshalJSON() ([]byte, error) {
	return []byte("\"" + a.GetHexString() + "\""), nil
}

//构造函数族
func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}

func StringToAddress(s string) Address { return BytesToAddress(utility.StrToBytes(s)) }
func BigToAddress(b *big.Int) Address  { return BytesToAddress(b.Bytes()) }
func HexToAddress(s string) Address    { return BytesToAddress(FromHex(s)) }

//赋值函数，如b超出a的容量则截取后半部分
func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}
	copy(a[:], b[:])
}

func (a *Address) SetString(s string) {
	a.SetBytes([]byte(s))
}

func (a *Address) Set(other Address) {
	copy(a[:], other[:])
}

// MarshalText returns the hex representation of a.
//把地址编码成十六进制字符串
func (a Address) MarshalText() ([]byte, error) {
	return utility.Bytes(a[:]).MarshalText()
}

// UnmarshalText parses a hash in hex syntax.
//把十六进制字符串解码成地址
func (a *Address) UnmarshalText(input []byte) error {
	return utility.UnmarshalFixedText("Address", input, a[:])
}

// UnmarshalJSON parses a hash in hex syntax.
//把十六进制JSONG格式字符串解码成地址
func (a *Address) UnmarshalJSON(input []byte) error {
	return utility.UnmarshalFixedJSON(addressT, input, a[:])
}

//类型转换输出函数
func (a Address) Bytes() []byte        { return a[:] }
func (a Address) BigInteger() *big.Int { return new(big.Int).SetBytes(a[:]) }
func (a Address) Hash() Hash           { return BytesToHash(a[:]) }

func (a Address) IsValid() bool {
	return len(a.Bytes()) > 0
}

func (a Address) GetHexString() string {
	str := ToHex(a[:])
	return str
}

func (a Address) String() string {
	return a.GetHexString()
}

func HexStringToAddress(s string) (a Address) {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return
	}
	buf, _ := hex.DecodeString(s[len(PREFIX):])
	if len(buf) == AddressLength {
		a.SetBytes(buf)
	}
	return
}

//256位哈希
type Hash [HashLength]byte

func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}
func BigToHash(b *big.Int) Hash { return BytesToHash(b.Bytes()) }
func HexToHash(s string) Hash   { return BytesToHash(FromHex(s)) }

// Get the string representation of the underlying hash
func (h Hash) Str() string   { return string(h[:]) }
func (h Hash) Bytes() []byte { return h[:] }
func (h Hash) Big() *big.Int { return new(big.Int).SetBytes(h[:]) }
func (h Hash) Hex() string   { return utility.Encode(h[:]) }

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (h Hash) TerminalString() string {
	return fmt.Sprintf("%x…%x", h[:3], h[29:])
}

func (h Hash) IsValid() bool {
	return len(h.Bytes()) > 0
}

// String implements the stringer interface and is used also by the logger when
// doing full logging into a file.
func (h Hash) String() string {
	return h.Hex()
}

func (h Hash) ShortS() string {
	str := h.Hex()
	return ShortHex12(str)
}

// Format implements fmt.Formatter, forcing the byte slice to be formatted as is,
// without going through the stringer interface used for logging.
func (h Hash) Format(s fmt.State, c rune) {
	fmt.Fprintf(s, "%"+string(c), h[:])
}

// UnmarshalText parses a hash in hex syntax.
func (h *Hash) UnmarshalText(input []byte) error {
	return utility.UnmarshalFixedText("Hash", input, h[:])
}

// UnmarshalJSON parses a hash in hex syntax.
func (h *Hash) UnmarshalJSON(input []byte) error {
	return utility.UnmarshalFixedJSON(hashT, input, h[:])
}

// MarshalText returns the hex representation of h.
func (h Hash) MarshalText() ([]byte, error) {
	return utility.Bytes(h[:]).MarshalText()
}

// Sets the hash to the value of b. If b is larger than len(h), 'b' will be cropped (from the left).
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:] //截取右边部分
	}

	copy(h[HashLength-len(b):], b)
}

// Set string `s` to h. If s is larger than len(h) s will be cropped (from left) to fit.
func (h *Hash) SetString(s string) { h.SetBytes([]byte(s)) }

// Sets h from other
func (h *Hash) Set(other Hash) {
	for i, v := range other {
		h[i] = v
	}
}

// Generate implements testing/quick.Generator.
func (h Hash) Generate(rand *rand.Rand, size int) reflect.Value {
	m := rand.Intn(len(h))            //m为0-len(h)之间的伪随机数
	for i := len(h) - 1; i > m; i-- { //从高位到m之间进行遍历
		h[i] = byte(rand.Uint32()) //rand.Uint32为32位非负伪随机数
	}
	return reflect.ValueOf(h)
}

// UnprefixedHash allows marshaling a Hash without 0x prefix.
type UnprefixedHash Hash

// UnmarshalText decodes the hash from hex. The 0x prefix is optional.
func (h *UnprefixedHash) UnmarshalText(input []byte) error {
	return utility.UnmarshalFixedUnprefixedText("UnprefixedHash", input, h[:])
}

// MarshalText encodes the hash as hex.
func (h UnprefixedHash) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(h[:])), nil
}

type Hash256 Hash
type StorageSize float64

type Hashes [2]Hash

func (h Hashes) ShortS() string {
	str := fmt.Sprintf("%s:%s", h[0].Hex(), h[1].Hex())
	return ShortHex12(str)
}

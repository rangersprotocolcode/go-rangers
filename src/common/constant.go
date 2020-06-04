package common

import (
	"errors"
	"math/big"
	"reflect"
)

const PREFIX = "0x"

const (
	//默认曲线相关参数开始：
	PubKeyLength = 65 //公钥字节长度，1 bytes curve, 64 bytes x,y。
	SecKeyLength = 97 //私钥字节长度，65 bytes pub, 32 bytes D。
	SignLength   = 65 //签名字节长度，32 bytes r & 32 bytes s & 1 byte recid.
	//默认曲线相关参数结束。
	AddressLength = 32 //地址字节长度(golang.SHA3，256位)
	HashLength    = 32 //哈希字节长度(golang.SHA3, 256位)。to do : 考虑废弃，直接使用golang的hash.Hash，直接为SHA3_256位，类型一样。
	GroupIdLength = 32
)

const (
	MinerTypeValidator = 0
	MinerTypeProposer  = 1
	MinerTypeUnknown   = 2

	MinerStatusNormal = 0
	MinerStatusAbort  = 1
)

var (
	hashT    = reflect.TypeOf(Hash{})
	addressT = reflect.TypeOf(Address{})
)

// 地址相关常量
var (
	ValidatorDBAddress = BigToAddress(big.NewInt(1))
	ProposerDBAddress  = BigToAddress(big.NewInt(2))
	RefundAddress      = BigToAddress(big.NewInt(3))

	FTSetAddress  = BigToAddress(big.NewInt(4))
	NFTSetAddress = BigToAddress(big.NewInt(5))
)

var (
	Big1   = big.NewInt(1)
	Big2   = big.NewInt(2)
	Big3   = big.NewInt(3)
	Big0   = big.NewInt(0)
	Big32  = big.NewInt(32)
	Big256 = big.NewInt(0xff)
	Big257 = big.NewInt(257)

	ErrSelectGroupNil     = errors.New("selectGroupId is nil")
	ErrSelectGroupInequal = errors.New("selectGroupId not equal")
	ErrCreateBlockNil     = errors.New("createBlock is nil")
	ErrGroupAlreadyExist  = errors.New("group already exist")
)

const (
	MaxInt8   = 1<<7 - 1
	MinInt8   = -1 << 7
	MaxInt16  = 1<<15 - 1
	MinInt16  = -1 << 15
	MaxInt32  = 1<<31 - 1
	MinInt32  = -1 << 31
	MaxInt64  = 1<<63 - 1
	MinInt64  = -1 << 63
	MaxUint8  = 1<<8 - 1
	MaxUint16 = 1<<16 - 1
	MaxUint32 = 1<<32 - 1
	MaxUint64 = 1<<64 - 1
)

package groupsig

import (
	"math/big"

	"x/src/consensus/groupsig/bn_curve"
	"fmt"
)

const PREFIX = "0x"

func hashToG1(m string) *bn_curve.G1 {
	g := &bn_curve.G1{}
	g.HashToPoint([]byte(m))
	return g
}

type BnInt struct {
	v big.Int
}

func (bi *BnInt) getHexString() string {
	buf := bi.v.Text(16)
	return PREFIX + buf
}

func (bi *BnInt) setHexString(s string) error {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return fmt.Errorf("arg failed")
	}
	buf := s[len(PREFIX):]
	bi.v.SetString(buf[:], 16)
	return nil
}

//BlsInt导出为big.Int
func (bi *BnInt) getBigInt() *big.Int {
	return new(big.Int).Set(&bi.v)
}

func (bi *BnInt) setBigInt(b *big.Int) error {
	bi.v.Set(b)
	return nil
}

func (bi *BnInt) serialize() []byte {
	return bi.v.Bytes()
}

func (bi *BnInt) deserialize(b []byte) error {
	bi.v.SetBytes(b)
	return nil
}

func (bi *BnInt) isEqual(b *BnInt) bool {
	return 0 == bi.v.Cmp(&b.v)
}

func (bi *BnInt) add(b *BnInt) error {
	bi.v.Add(&bi.v, &b.v)
	return nil
}

func (bi *BnInt) sub(b *BnInt) error {
	bi.v.Sub(&bi.v, &b.v)
	return nil
}

func (bi *BnInt) mul(b *BnInt) error {
	bi.v.Mul(&bi.v, &b.v)
	return nil
}

func (bi *BnInt) mod() error {
	bi.v.Mod(&bi.v, bn_curve.Order)
	return nil
}

// SetDecString --
//func (bi *BnInt) SetDecString(s string) error {
//	bi.v.SetString(s, 10)
//	return nil
//}

//func (bi *BnInt) SetString(s string) error {
//	bi.v.SetString(s, 10)
//	return nil
//}

//func (bi *BnInt) GetString() string {
//	b := bi.GetBigInt().Bytes()
//	return string(b)
//}

//type bnG2 struct {
//	v bn_curve.G2
//}

//func (bg *bnG2) Serialize() []byte {
//	return bg.v.Marshal()
//}
//
//func (bg *bnG2) Deserialize(b []byte) error {
//	bg.v.Unmarshal(b)
//	return nil
//}

//func (bg *bnG2) Add(bh *bnG2) error {
//	bg.v.Add(&bg.v, &bh.v)
//	return nil
//}

//// GetMasterSecretKey --
//func (sec *Seckey) GetMasterSecretKey(k int) (msk []Seckey) {
//	msk = make([]Seckey, k)
//	msk[0] = *sec
//
//	r := base.NewRand() //生成随机数
//	for i := 1; i < k; i++ {
//		msk[i] = *NewSeckeyFromRand(r.Deri(1))
//	}
//	return msk
//}
//
//// GetMasterPublicKey --
//func GetMasterPublicKey(msk []Seckey) (mpk []Pubkey) {
//	n := len(msk)
//	mpk = make([]Pubkey, n)
//	for i := 0; i < n; i++ {
//		mpk[i] = *NewPubkeyFromSeckey(msk[i])
//	}
//	return mpk
//}

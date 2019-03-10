package network

import (
	"bytes"
	"log"

	"x/src/common"

	"github.com/libp2p/go-libp2p-crypto"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-peer"
	pb "github.com/libp2p/go-libp2p-crypto/pb"
)

// adapt common.Privatekey and common.Publickey to use key for libp2p

var keyType pb.KeyType = 3

type Pubkey struct {
	PublicKey common.PublicKey
}

type Privkey struct {
	PrivateKey common.PrivateKey
}

func getId(p common.PublicKey) peer.ID {
	pubKey := &Pubkey{PublicKey: p}
	id, e := peer.IDFromPublicKey(pubKey)
	if e != nil {
		log.Printf("[Network]IDFromPublicKey error:%s", e.Error())
		panic("GetIdFromPublicKey error!")
	}
	return id
}

func (pub *Pubkey) Verify(data []byte, sig []byte) (bool, error) {
	return pub.PublicKey.Verify(data, common.BytesToSign(sig)), nil
}

func (pub *Pubkey) Bytes() ([]byte, error) {
	pbmes := new(pb.PublicKey)
	pbmes.Type = &keyType
	pbmes.Data = pub.PublicKey.ToBytes()
	return proto.Marshal(pbmes)
}

func (pub *Pubkey) Equals(key crypto.Key) bool {
	p, ok := key.(*Pubkey)
	if !ok {
		return false
	}
	b1, e1 := pub.Bytes()
	b2, e2 := p.Bytes()
	if e1 != nil || e2 != nil {
		return false
	}
	return bytes.Equal(b1[:], b2[:])
}

func UnmarshalEcdsaPublicKey(b []byte) (crypto.PubKey, error) {
	pk := common.BytesToPublicKey(b)
	return &Pubkey{PublicKey: *pk}, nil
}

func (prv *Privkey) Sign(b []byte) ([]byte, error) {
	return prv.PrivateKey.Sign(b).Bytes(), nil
}

func (prv *Privkey) GetPublic() crypto.PubKey {
	pub := prv.PrivateKey.GetPubKey()
	return &Pubkey{pub}
}

func (prv *Privkey) Bytes() ([]byte, error) {
	pbmes := new(pb.PrivateKey)
	pbmes.Type = &keyType
	pbmes.Data = prv.PrivateKey.ToBytes()
	return proto.Marshal(pbmes)
}

func (prv *Privkey) Equals(key crypto.Key) bool {
	p, ok := key.(*Pubkey)
	if !ok {
		return false
	}
	b1, e1 := prv.Bytes()
	b2, e2 := p.Bytes()
	if e1 != nil || e2 != nil {
		return false
	}
	return bytes.Equal(b1[:], b2[:])
}

func UnmarshalEcdsaPrivateKey(b []byte) (crypto.PrivKey, error) {
	pk := common.BytesToSecKey(b)
	return &Privkey{PrivateKey: *pk}, nil
}

func init() {
	pb.KeyType_name[3] = "Ecdsa"
	pb.KeyType_value["Ecdsa"] = 3

	crypto.PubKeyUnmarshallers[keyType] = UnmarshalEcdsaPublicKey
	crypto.PrivKeyUnmarshallers[keyType] = UnmarshalEcdsaPrivateKey
}

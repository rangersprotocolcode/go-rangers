package ed25519

import (
	"encoding/hex"
	"testing"
	"crypto/rand"
	"io"
	"fmt"

	"x/src/common/ed25519/edwards25519"
)

const message = "This is a' testing message"

func DoTestECVRF(t *testing.T, pk PublicKey, sk PrivateKey, msg []byte, verbose bool) {
	pi, err := ECVRFProve(pk, sk, msg[:])
	if err != nil {
		t.Fatal(err)
	}
	res, err := ECVRFVerify(pk, pi, msg[:])
	if err != nil {
		t.Fatal(err)
	}
	if !res {
		t.Errorf("VRF failed")
	}

	// when everything get through
	if verbose {
		fmt.Printf("alpha: %s\n", hex.EncodeToString(msg))
		fmt.Printf("x: %s\n", hex.EncodeToString(sk))
		fmt.Printf("P: %s\n", hex.EncodeToString(pk))
		fmt.Printf("pi: %s\n", hex.EncodeToString(pi))
		fmt.Printf("vrf: %s\n", hex.EncodeToString(ECVRFProof2hash(pi)))

		r, c, s, err := ecVRFDecodeProof(pi)
		if err != nil {
			t.Fatal(err)
		}
		// u = (g^x)^c * g^s = P^c * g^s
		var u edwards25519.ProjectiveGroupElement
		P := os2ecp(pk, pk[31]>>7)
		edwards25519.GeDoubleScalarMultVartime(&u, c, P, s)
		fmt.Printf("r: %s\n", hex.EncodeToString(ecp2os(r)))
		fmt.Printf("c: %s\n", hex.EncodeToString(c[:]))
		fmt.Printf("s: %s\n", hex.EncodeToString(s[:]))
		fmt.Printf("u: %s\n", hex.EncodeToString(ecp2osProj(&u)))
	}
}

const howMany = 1000

func TestECVRF(t *testing.T) {
	for i := howMany; i > 0; i-- {
		pk, sk, err := GenerateKey(nil)
		if err != nil {
			t.Fatal(err)
		}
		var msg [32]byte
		io.ReadFull(rand.Reader, msg[:])
		DoTestECVRF(t, pk, sk, msg[:], false)
	}
}

const pks = "885f642c8390293eb74d08cf38d3333771e9e319cfd12a21429eeff2eddeebd2"
const sks = "1fcce948db9fc312902d49745249cfd287de1a764fd48afb3cd0bdd0a8d74674885f642c8390293eb74d08cf38d3333771e9e319cfd12a21429eeff2eddeebd2"

func TestECVRFOnce(t *testing.T) {
	pk, _ := hex.DecodeString(pks)
	sk, _ := hex.DecodeString(sks)
	m := []byte(message)
	DoTestECVRF(t, pk, sk, m, true)

	h := ecVRFHashToCurve(m, pk)
	fmt.Printf("h: %s\n", hex.EncodeToString(ecp2os(h)))
}

func BenchmarkProve(b *testing.B) {
	pk, sk, err := GenerateKey(nil)
	if err != nil {
		b.Fatal(err)
	}
	m := []byte(message)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ECVRFProve(pk, sk, m)
	}
}

func BenchmarkVRFVerify(b *testing.B) {
	pk, sk, err := GenerateKey(nil)
	if err != nil {
		b.Fatal(err)
	}
	m := []byte(message)
	pi, err := ECVRFProve(pk, sk, m)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ECVRFVerify(pk, pi, m)
	}
}

package ed25519

import (
	"bytes"
	"errors"
	"x/src/common/ed25519/edwards25519"
	"crypto/sha512"
	"encoding/hex"
)

const (
	N2 = 32 // ceil(log2(q) / 8)
	N  = N2 / 2

	ProveSize = 80
)

var (
	ErrMalformedSK = errors.New("ECVRF: malformed sk")

	ErrDecodeError = errors.New("ECVRF: decode error")

	suite, _ = hex.DecodeString("04")

	one, _ = hex.DecodeString("01")

	two, _ = hex.DecodeString("02")
)

// VRFProve is the output prove of VRF_Ed25519.
type VRFProve []byte //ProveSize = 80 in bytes

/*
Construct a VRF proof given a secret key and a message.
 */
func ECVRFProve(sk PrivateKey, m []byte) (pi VRFProve, err error) {
	if len(sk) != PrivateKeySize {
		return nil, ErrMalformedSK
	}
	pkBytes := new([32]byte)
	copy(pkBytes[:], sk[32:])
	if !stringToPoint(new(edwards25519.ExtendedGroupElement), *pkBytes) {
		return nil, ErrMalformedSK
	}
	pk := pkBytes[:]

	x, truncatedHashedSK := expandSecret(sk)

	//fmt.Printf("x:%s\n", hex.EncodeToString(x[:]))
	h := hashToCurve(m, pk)
	//fmt.Printf("h:%s\n", hex.EncodeToString(h[:]))
	hPoint := new(edwards25519.ExtendedGroupElement)
	hPoint.FromBytes(&h)

	//Gamma = x*H
	gamma := edwards25519.GeScalarMult(hPoint, x)

	kScalar := vrfNonceGeneration(*truncatedHashedSK, h)
	//fmt.Printf("kScalar:%s\n", hex.EncodeToString(kScalar[:]))
	kBPoint := new(edwards25519.ExtendedGroupElement)
	edwards25519.GeScalarMultBase(kBPoint, kScalar)

	kbBytes := new([32]byte)
	kBPoint.ToBytes(kbBytes)
	//fmt.Printf("k*b:%s\n", hex.EncodeToString(kbBytes[:]))

	kHPoint := edwards25519.GeScalarMult(hPoint, kScalar)

	khBytes := new([32]byte)
	kHPoint.ToBytes(khBytes)
	//fmt.Printf("k*h:%s\n", hex.EncodeToString(khBytes[:]))

	//c = ECVRF_hash_points(H, Gamma, k*B, k*H)  (writes only to the first 16 bytes of c_scalar
	c := hashPoints(*hPoint, *gamma, *kBPoint, *kHPoint)

	//pi[0:32] = point_to_string(Gamma)
	provePart1 := new([32]byte)
	gamma.ToBytes(provePart1)

	//pi[32:48] = c (16 bytes)
	provePart2 := c

	//pi[48:80] = c*x + k (mod q)
	cScalar := new([32]byte)
	copy(cScalar[:], c[:])
	provePart3 := new([32]byte)
	edwards25519.ScMulAdd(provePart3, cScalar, x, kScalar)

	// pi = gamma || I2OSP(c, N) || I2OSP(s, 2N)
	var buf bytes.Buffer
	buf.Write(provePart1[:]) // 2N
	buf.Write(provePart2[:])
	buf.Write(provePart3[:])

	return buf.Bytes(), nil
}

/* Verify a VRF proof (for a given a public key and message) and validate the
 * public key.
 *
 * For a given public key and message, there are many possible proofs but only
 * one possible output hash.
 *
 */
func ECVRFVerify(pk PublicKey, pi VRFProve, m []byte) (bool, error) {
	pi = tryZeroPadding(pi)
	gamma, cScalar, sScalar, err := decodeProof(pi)
	if err != nil {
		return false, err
	}
	sScalar32 := new([32]byte)
	edwards25519.ScReduce(sScalar32, sScalar)

	hPoint, yPoint := new(edwards25519.ExtendedGroupElement), new(edwards25519.ExtendedGroupElement)
	tmpCachedPoint := new(edwards25519.CachedGroupElement)
	tempP1Point := new(edwards25519.CompletedGroupElement)
	uPoint := new(edwards25519.ExtendedGroupElement)
	vPoint := new(edwards25519.ExtendedGroupElement)

	h := hashToCurve(m, pk)
	hPoint.FromBytes(&h)

	//calculate U = s*B - c*Y
	pkBytes := new([32]byte)
	copy(pkBytes[:], pk[:])
	yPoint.FromBytes(pkBytes)
	temp3Point := edwards25519.GeScalarMult(yPoint, cScalar) //tmp_p3 = c*Y
	temp3Point.ToCached(tmpCachedPoint)                      //tmp_cached = c*Y
	edwards25519.GeScalarMultBase(temp3Point, sScalar32)     // tmp_p3 = s*B

	edwards25519.GeSub(tempP1Point, temp3Point, tmpCachedPoint) // tmp_p1p1 = tmp_p3 - tmp_cached = s*B - c*Y
	tempP1Point.ToExtended(uPoint)                              // U = s*B - c*Y

	/* calculate V = s*H -  c*Gamma */
	temp3Point = edwards25519.GeScalarMult(gamma, cScalar)      //tmp_p3 = c*Gamma
	temp3Point.ToCached(tmpCachedPoint)                         //tmp_cached = c*Gamma
	temp3Point = edwards25519.GeScalarMult(hPoint, sScalar32)   //tmp_p3 = s*H
	edwards25519.GeSub(tempP1Point, temp3Point, tmpCachedPoint) //tmp_p1p1 = tmp_p3 - tmp_cached = s*H - c*Gamma
	tempP1Point.ToExtended(vPoint)                              //V = s*H - c*Gamma

	// c' = ECVRF_hash_points(g, h, g^x, gamma, u, v)
	cPrime := hashPoints(*hPoint, *gamma, *uPoint, *vPoint)

	//return cPrime.Cmp(f2ip(cScalar)) == 0, nil
	cScalar16 := new([16]byte)
	copy(cScalar16[:], cScalar[:])
	return cPrime == *cScalar16, nil
}

/* Utility function to convert a "secret key" (32-byte seed || 32-byte PK)
 * into the public point Y, the private saclar x, and truncated hash of the
 * seed to be used later in nonce generation.
 */
func expandSecret(sk PrivateKey) (x *[32]byte, truncatedHashedSK *[32]byte) {
	// copied from golang.org/x/crypto/ed25519/ed25519.go -- has to be the same
	digest := sha512.Sum512(sk[:32])
	digest[0] &= 248
	digest[31] &= 127
	digest[31] |= 64

	x = new([32]byte)
	truncatedHashedSK = new([32]byte)
	copy(x[:], digest[:32])
	copy(truncatedHashedSK[:], digest[32:])
	return
}

/* Hash a message to a curve point using Elligator2.
 * Specified in VRF draft spec section 5.4.1.2.
 */
func hashToCurve(m []byte, pk PublicKey) [32]byte {
	/* r = first 32 bytes of SHA512(suite || 0x01 || Y || alpha) */
	hash := sha512.New()
	hash.Write(suite)
	hash.Write(one)
	hash.Write(pk[:])
	hash.Write(m)
	h := hash.Sum(nil)

	r := new([32]byte)
	copy(r[:], h[:32])
	/* clear sign bit */
	r[31] &= 0x7f
	//fmt.Printf("r:%s\n", hex.EncodeToString(r[:]))

	return fromUniform(*r)
}

/* Subroutine specified in draft spec section 5.4.3.
 * Hashes four points to a 16-byte string.
 * Constant time.
*/
func hashPoints(p1, p2, p3, p4 edwards25519.ExtendedGroupElement) (c [16]byte) {
	bytes1 := new([32]byte)
	p1.ToBytes(bytes1)

	bytes2 := new([32]byte)
	p2.ToBytes(bytes2)

	bytes3 := new([32]byte)
	p3.ToBytes(bytes3)

	bytes4 := new([32]byte)
	p4.ToBytes(bytes4)

	var buf bytes.Buffer
	buf.Write(suite)
	buf.Write(two)
	buf.Write(bytes1[:])
	buf.Write(bytes2[:])
	buf.Write(bytes3[:])
	buf.Write(bytes4[:])

	hash := sha512.Sum512(buf.Bytes())
	copy(c[:], hash[:N])
	return
}

/* elligator2 */
func fromUniform(r [32]byte) [32]byte {
	e, negx, rr2, x, x2, x3 := new(edwards25519.FieldElement), new(edwards25519.FieldElement), new(edwards25519.FieldElement), new(edwards25519.FieldElement), new(edwards25519.FieldElement), new(edwards25519.FieldElement)

	s := new([32]byte)
	copy(s[:], r[:32])

	xSign := s[31] & 0x80
	s[31] &= 0x7f

	edwards25519.FeFromBytes(rr2, s)

	/* elligator */
	edwards25519.FeSquare2(rr2, rr2)
	rr2[0]++
	edwards25519.FeInvert(rr2, rr2)
	edwards25519.FeMul(x, &edwards25519.A, rr2)
	edwards25519.FeNeg(x, x)

	edwards25519.FeSquare(x2, x)
	edwards25519.FeMul(x3, x, x2)
	edwards25519.FeAdd(e, x3, x)
	edwards25519.FeMul(x2, x2, &edwards25519.A)
	edwards25519.FeAdd(e, x2, e)

	e = chi25519(e)
	edwards25519.FeToBytes(s, e)
	eIsMinus1 := s[1] & 1
	edwards25519.FeNeg(negx, x)
	edwards25519.FeCMove(x, negx, int32(eIsMinus1))
	edwards25519.FeZero(x2)
	edwards25519.FeCMove(x2, &edwards25519.A, int32(eIsMinus1))
	edwards25519.FeSub(x, x, x2)

	/* yed = (x-1)/(x+1) */
	{
		one, xPlusOne, xPlusOneInv, xMinusOne, yed := new(edwards25519.FieldElement), new(edwards25519.FieldElement), new(edwards25519.FieldElement), new(edwards25519.FieldElement), new(edwards25519.FieldElement)

		edwards25519.FeOne(one)
		edwards25519.FeAdd(xPlusOne, x, one)
		edwards25519.FeSub(xMinusOne, x, one)
		edwards25519.FeInvert(xPlusOneInv, xPlusOne)
		edwards25519.FeMul(yed, xMinusOne, xPlusOneInv)
		edwards25519.FeToBytes(s, yed)
	}

	/* recover x */
	p3 := new(edwards25519.ExtendedGroupElement)
	s[31] |= xSign
	p3.FromBytes(s)

	/* multiply by the cofactor */
	p1 := new(edwards25519.CompletedGroupElement)
	p2 := new(edwards25519.ProjectiveGroupElement)

	p3.Double(p1)
	p1.ToProjective(p2)
	p2.Double(p1)
	p1.ToProjective(p2)
	p2.Double(p1)
	p1.ToExtended(p3)

	p3.ToBytes(s)
	return *s
}

func chi25519(z *edwards25519.FieldElement) *edwards25519.FieldElement {
	t0, t1, t2, t3 := new(edwards25519.FieldElement), new(edwards25519.FieldElement), new(edwards25519.FieldElement), new(edwards25519.FieldElement)
	var i int

	edwards25519.FeSquare(t0, z)
	edwards25519.FeMul(t1, t0, z)
	edwards25519.FeSquare(t0, t1)
	edwards25519.FeSquare(t2, t0)
	edwards25519.FeSquare(t2, t2)
	edwards25519.FeMul(t2, t2, t0)
	edwards25519.FeMul(t1, t2, z)
	edwards25519.FeSquare(t2, t1)

	for i = 1; i < 5; i++ {
		edwards25519.FeSquare(t2, t2)
	}
	edwards25519.FeMul(t1, t2, t1)
	edwards25519.FeSquare(t2, t1)
	for i = 1; i < 10; i++ {
		edwards25519.FeSquare(t2, t2)
	}
	edwards25519.FeMul(t2, t2, t1)
	edwards25519.FeSquare(t3, t2)
	for i = 1; i < 20; i++ {
		edwards25519.FeSquare(t3, t3)
	}
	edwards25519.FeMul(t2, t3, t2)
	edwards25519.FeSquare(t2, t2)
	for i = 1; i < 10; i++ {
		edwards25519.FeSquare(t2, t2)
	}
	edwards25519.FeMul(t1, t2, t1)
	edwards25519.FeSquare(t2, t1)
	for i = 1; i < 50; i++ {
		edwards25519.FeSquare(t2, t2)
	}
	edwards25519.FeMul(t2, t2, t1)
	edwards25519.FeSquare(t3, t2)
	for i = 1; i < 100; i++ {
		edwards25519.FeSquare(t3, t3)
	}
	edwards25519.FeMul(t2, t3, t2)
	edwards25519.FeSquare(t2, t2)
	for i = 1; i < 50; i++ {
		edwards25519.FeSquare(t2, t2)
	}
	edwards25519.FeMul(t1, t2, t1)
	edwards25519.FeSquare(t1, t1)
	for i = 1; i < 4; i++ {
		edwards25519.FeSquare(t1, t1)
	}

	result := new(edwards25519.FieldElement)
	edwards25519.FeMul(result, t1, t0)
	return result
}

/* Deterministically generate a (secret) nonce to be used in a proof.
 * Specified in draft spec section 5.4.2.2.
 * Note: In the spec, this subroutine computes truncated_hashed_sk_string
 * Here we instead takes it as an argument, and we compute it in vrf_expand_sk
 */
func vrfNonceGeneration(truncatedHashedSK [32]byte, h [32]byte) *[32]byte {
	var buf bytes.Buffer
	buf.Write(truncatedHashedSK[:])
	buf.Write(h[:])
	hash := sha512.Sum512(buf.Bytes())
	//fmt.Printf("k:%s\n", hex.EncodeToString(hash[:]))

	var result = new([32]byte)
	edwards25519.ScReduce(result, &hash)
	return result
}

/* Decode an 80-byte proof pi into a point gamma, a 16-byte scalar c, and a
 * 32-byte scalar s, as specified in IETF draft section 5.4.4.
 */
func decodeProof(pi []byte) (gamma *edwards25519.ExtendedGroupElement, c *[N2]byte, s *[N2 * 2]byte, err error) {
	r := new([32]byte)
	copy(r[:], pi[:32])

	//gamma = decode_point(pi[0:32])
	gamma = new(edwards25519.ExtendedGroupElement)
	if !stringToPoint(gamma, *r) {
		return nil, nil, nil, ErrDecodeError
	}

	/* c = pi[32:48] */
	c = new([N2]byte)
	copy(c[:], pi[32:48])

	/* s = pi[48:80] */
	s = new([N2 * 2]byte)
	copy(s[:], pi[48:80])
	return
}

/* Decode elliptic curve point from 32-byte octet string per RFC8032 section
 * 5.1.3.
 *
 * In particular we must reject non-canonical encodings (i.e., when the encoded
 * y coordinate is not reduced mod p). We do not check whether the point is on
 * the main subgroup or whether it is of low order.
 */
func stringToPoint(point *edwards25519.ExtendedGroupElement, s [32]byte) bool {
	if isCanonical(s) == 0 || !point.FromBytes(&s) {
		return false
	}
	return true
}

func isCanonical(s [32]byte) byte {
	c := (s[31] & 0x7f) ^ 0x7f
	for i := 30; i > 0; i-- {
		c |= s[i] ^ 0xff
	}
	c = (c - 1) >> 8
	d := (0xed - 1 - s[0]) >> 8

	return 1 - (c & d & 1)
}

func tryZeroPadding(pi VRFProve) VRFProve {
	if len(pi) >= ProveSize {
		return pi
	}
	piPadding := make([]byte, ProveSize)
	copy(piPadding[ProveSize-len(pi):], pi[:])
	return piPadding
}

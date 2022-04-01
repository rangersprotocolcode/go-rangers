package dkim

import (
	"bytes"
	"com.tuntun.rocket/node/src/utility"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/go-playground/validator/v10"
	"strconv"
	"strings"
)

type SigningAlgorithm int
type CanonicalizationType int

const (
	RsaSha1 SigningAlgorithm = iota
	RsaSha256
)

const (
	Simple CanonicalizationType = iota
	Relaxed
)

var (
	ErrNoCanonicalizationType = &DkimError{"no this Canonicalization type"}
	ErrEumptyString           = &DkimError{"string is empty"}
	ErrDkimProtocol           = &DkimError{"error protocol"}
)

func getDkimParsingErrorWithKeyValue(key string, value string) *DkimError {
	return getDkimParsingError(key + ":" + value)
}

func getDkimParsingError(msg string) *DkimError {
	return &DkimError{msg}
}

type DkimError struct {
	msg string
}

func (error *DkimError) Error() string {
	return error.msg
}

type CanonicalizationTypes struct {
	TypeOne CanonicalizationType
	TypeTwo CanonicalizationType
}

type Header struct {
	Algorithm           SigningAlgorithm `validate:"required"`
	Signature           []byte           `validate:"required"`
	BodyHash            []byte           `validate:"required"`
	Canonicalization    CanonicalizationTypes
	Sdid                string   `validate:"required"`
	Selector            string   `validate:"required"`
	SignedHeaders       []string `validate:"required"`
	CopiedHeaders       string
	Auid                string
	BodyLength          uint64
	SignatureTimestamp  uint64
	SignatureExpiration uint64
	Original            string
}

func NewHeader(sdid string, selector string) *Header {
	h := new(Header)
	h.Algorithm = RsaSha256
	h.Signature = make([]byte, 0)
	h.BodyHash = make([]byte, 0)
	h.Canonicalization = CanonicalizationTypes{Relaxed, Relaxed}
	h.Sdid = sdid
	h.Selector = selector
	h.SignedHeaders = []string{"mime-version", "mime-version", "references", "in-reply-to", "from", "date", "message-id", "subject", "to"}
	return h
}

func parseHeader(name string, value string) (*Header, error) {
	const (
		B = iota
		EqualSign
		Semicolon
	)

	state := B
	b_idx := 0
	b_end_idx := 0

	for idx, c := range value {
		switch state {
		case B:
			if c == 'b' {
				state = EqualSign
			}
		case EqualSign:
			if c == '=' {
				b_idx = idx + 1
				state = Semicolon
			} else {
				state = B
			}
		case Semicolon:
			if c == ';' {
				b_end_idx = idx
				break
			}

		}
	}

	if b_end_idx == 0 && state == Semicolon {
		b_end_idx = len(value)
	}

	save := value[:b_idx]
	end := value[b_end_idx:]

	save += end

	var got_v = false
	var algorithm SigningAlgorithm = -1
	var signature []byte
	var body_hash []byte
	var sdid string
	var auid string
	var signed_headers []string
	var copied_headers string
	var body_lenght uint64
	var canonicalization *CanonicalizationTypes
	var selector string
	var signature_timestamp uint64
	var signature_expiration uint64
	var q = false
	for _, e := range strings.Split(value, ";") {
		name := strings.TrimSpace(strings.SplitN(e, "=", 2)[0])
		if "" != name {
			val := strings.TrimSpace(strings.SplitN(e, "=", 2)[1])
			switch name {
			case "v":
				if got_v {
					return nil, getDkimParsingErrorWithKeyValue("DuplicatedField", "v")
				} else if val != "1" {
					return nil, getDkimParsingErrorWithKeyValue("UnsupportedDkimVersion", val)
				} else {
					got_v = true
				}
			case "a":
				if algorithm != -1 {
					return nil, getDkimParsingErrorWithKeyValue("DuplicatedField", "a")
				} else if val == "rsa-sha1" {
					algorithm = RsaSha1
				} else if val == "rsa-sha256" {
					algorithm = RsaSha256
				} else {
					return nil, getDkimParsingErrorWithKeyValue("UnsupportedSigningAlgorithm", val)
				}
			case "b":
				if len(signature) > 0 {
					return nil, getDkimParsingErrorWithKeyValue("DuplicatedField", "b")
				} else {
					var buf bytes.Buffer
					for _, c := range val {
						if !(c == '\t' || c == '\n' || c == '\r' || c == ' ') {
							buf.WriteByte(byte(c))
						}
					}
					newVal, err := base64.StdEncoding.DecodeString(buf.String())
					if nil != err {
						return nil, getDkimParsingErrorWithKeyValue("base64 error", err.Error())
					}
					signature = newVal
				}
			case "bh":
				if len(body_hash) > 0 {
					return nil, getDkimParsingErrorWithKeyValue("DuplicatedField", "bh")
				} else {
					var buf bytes.Buffer
					for _, c := range val {
						if !(c == '\t' || c == '\n' || c == '\r' || c == ' ') {
							buf.WriteByte(byte(c))
						}
					}
					newVal, err := base64.StdEncoding.DecodeString(buf.String())
					if nil != err {
						return nil, getDkimParsingErrorWithKeyValue("base64 error", err.Error())
					}
					body_hash = newVal
				}
			case "c":
				if nil != canonicalization {
					return nil, getDkimParsingErrorWithKeyValue("DuplicatedField", "c")
				} else {
					switch val {
					case "relaxed/relaxed":
						canonicalization = &CanonicalizationTypes{Relaxed, Relaxed}
					case "relaxed/simple", "relaxed":
						canonicalization = &CanonicalizationTypes{Relaxed, Simple}
					case "simple/relaxed":
						canonicalization = &CanonicalizationTypes{Simple, Relaxed}
					case "simple/simple", "simple":
						canonicalization = &CanonicalizationTypes{Simple, Simple}
					default:
						return nil, getDkimParsingErrorWithKeyValue("InvalidCanonicalizationType", val)
					}

				}
			case "d":
				if len(sdid) > 0 {
					return nil, getDkimParsingErrorWithKeyValue("DuplicatedField", "d")
				} else {
					sdid = val
				}
			case "h":
				if len(signed_headers) > 0 {
					return nil, getDkimParsingErrorWithKeyValue("DuplicatedField", "h")
				} else {
					headers := make([]string, 0)
					for _, header := range strings.Split(val, ":") {
						headers = append(headers, header)
					}
					signed_headers = headers
				}
			case "i":
				if len(auid) > 0 {
					return nil, getDkimParsingErrorWithKeyValue("DuplicatedField", "i")
				} else {
					auid = val
				}
			case "l":
				if body_lenght != 0 {
					return nil, getDkimParsingErrorWithKeyValue("DuplicatedField", "l")
				} else {
					atoi, err := strconv.Atoi(val)
					if nil != err {
						return nil, getDkimParsingErrorWithKeyValue("InvalidBodyLenght", err.Error())
					}
					body_lenght = uint64(atoi)
				}
			case "q":
				if q {
					return nil, getDkimParsingErrorWithKeyValue("DuplicatedField", "q")
				} else {
					has := false
					methods := make([]string, 0)
					cmd := ""
					for _, method := range strings.Split(val, ":") {
						if method == "dns/txt" {
							has = true
						}
						methods = append(methods, method)
						cmd += "%s,"
					}
					if !has {
						return nil, getDkimParsingErrorWithKeyValue("UnsupportedPublicKeyQueryMethods", fmt.Sprintf(cmd, methods))
					}
					q = true
				}
			case "s":
				if len(selector) > 0 {
					return nil, getDkimParsingErrorWithKeyValue("DuplicatedField", "s")
				} else {
					selector = val
				}
			case "t":
				if signature_timestamp != 0 {
					return nil, getDkimParsingErrorWithKeyValue("DuplicatedField", "t")
				} else {
					atoi, err := strconv.Atoi(val)
					if nil != err {
						return nil, getDkimParsingErrorWithKeyValue("InvalidSignatureTimestamp", err.Error())
					} else {
						signature_timestamp = uint64(atoi)
					}
				}
			case "x":
				if signature_expiration != 0 {
					return nil, getDkimParsingErrorWithKeyValue("DuplicatedField", "x")
				} else {
					atoi, err := strconv.Atoi(val)
					if nil != err {
						return nil, getDkimParsingErrorWithKeyValue("InvalidSignatureExpiration", err.Error())
					} else {
						signature_expiration = uint64(atoi)
					}
				}
			case "z":
				if len(copied_headers) > 0 {
					return nil, getDkimParsingErrorWithKeyValue("DuplicatedField", "z")
				} else {
					copied_headers = val
				}
			}
		}
	}

	if nil == canonicalization {
		canonicalization = &CanonicalizationTypes{Simple, Simple}
	}

	if canonicalization.TypeOne == Relaxed {
		save = fmt.Sprintf("dkim-signature:%s", canonicalizeHeaderRelaxed(save))
	} else if canonicalization.TypeOne == Simple {
		save = fmt.Sprintf("%s:%s", name, save)
	}

	header := Header{
		algorithm,
		signature,
		body_hash,
		*canonicalization,
		sdid,
		selector,
		signed_headers,
		copied_headers,
		auid, body_lenght,
		signature_timestamp,
		signature_expiration,
		save,
	}

	err := validator.New().Struct(header)
	if nil != err {
		return nil, getDkimParsingErrorWithKeyValue("MissingField", err.Error())
	}
	return &header, nil
}

func Verify(data []byte) (bs []byte) {
	emailString := utility.BytesToStr(data)
	defer func() { // 必须要先声明defer，否则不能捕获到panic异常
		if err := recover(); err != nil {
			bs = nil
		}
	}()
	email, err := FromString(emailString)
	if nil != err {
		return nil
	}

	message, err := email.getDkimMessage()
	if nil != err {
		return nil
	}

	pub, err := email.GetSigPubKey()
	if err != nil {
		return nil
	}

	digest := sha256.Sum256([]byte(message))
	verifyErr := rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], email.DkimHeader.Signature)
	if nil != verifyErr {
		return nil
	}

	hash, err := email.GetSigHash()
	if nil != err {
		return nil
	}

	from, err := email.GetFromHash()
	if nil != err {
		return nil
	}
	ret := make([]byte, 0)
	ret = append(ret, hash[0:]...)
	ret = append(ret, from[0:]...)

	return ret
}

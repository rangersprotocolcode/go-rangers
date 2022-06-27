package dkim

import (
	"bufio"
	"bytes"
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/storage/account"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"mime"
	"strings"
)

const MIN_EMAIL_LEN = 6
const MAX_EMAIL_LEN = 100
const FR_EMAIL_LEN = MAX_EMAIL_LEN/31 + 1

type Email struct {
	Headers    []HeaderTuple
	Body       string
	DkimHeader *Header
}

type HeaderTuple struct {
	one   string
	tow   string
	three string
}

func getOneHeader(s string) (headLine string, rest string) {
	if s == "" {
		return "", ""
	}
	last := 0
	for i, c := range s {
		last = i
		if c == '\n' {
			if s[i+1] == ' ' || s[i+1] == '\t' {
				continue
			} else {
				break
			}
		}
	}
	headLine = s[:last]
	rest = s[last+1:]
	if last+1 == len(s) {
		headLine = s
		rest = ""
	}
	return
}

func FromString(value string) (*Email, error) {
	if value == "" {
		return nil, ErrEmptyString
	}
	allHeaders := make([]HeaderTuple, 0)
	val := strings.SplitN(value, "\r\n\r\n", 2)
	var header *Header
	if len(val) == 1 {
		val = strings.SplitN(value, "\n\n", 2)
	}
	headers := val[0]
	if headers == "" || len(headers) == 0 {
		return nil, ErrEmptyString
	}
	for {
		header, rest := getOneHeader(headers)
		if "" != header {
			keyVal := strings.SplitN(header, ":", 2)
			if len(keyVal) == 0 {
				return nil, ErrDkimProtocol
			}

			var valueVal = ""
			if len(keyVal) == 1 {
				valueVal = ""
			} else {
				valueVal = strings.TrimLeft(keyVal[1], " ")
				valueVal = strings.TrimRight(valueVal, "\r")
			}

			allHeaders = append(allHeaders, HeaderTuple{keyVal[0], ":", valueVal})
			headers = rest
		} else {
			break
		}
	}

	for _, dkim_header := range allHeaders {
		if strings.ToLower(dkim_header.one) == "dkim-signature" {
			dkim, err := parseHeader("Dkim-Signature", dkim_header.three)
			if nil != err {
				return nil, getDkimParsingErrorWithKeyValue("NotADkimSignatureHeader", err.Error())
			}
			header = dkim
		}
	}

	return &Email{
		allHeaders,
		val[1],
		header,
	}, nil

}

func (email *Email) getDkimMessage() (string, error) {
	if nil != email.DkimHeader {
		header := email.DkimHeader
		if header.Canonicalization.TypeOne == Relaxed {
			return CanonicalizeHeaderRelaxed(&email.Headers, &header.SignedHeaders) + header.Original, nil
		} else if header.Canonicalization.TypeOne == Simple {
			return CanonicalizeHeadersSimple(&email.Headers, &header.SignedHeaders) + header.Original, nil
		}
		return "", ErrNoCanonicalizationType
	} else {
		return "", nil
	}
}

func (email *Email) getHeaderItem(key string) (string, error) {
	if nil == email.Headers {
		return "", ErrDkimProtocol
	}
	if "" == key {
		return "", ErrEmptyString
	}
	headers := email.Headers
	for _, header := range headers {
		if strings.ToLower(header.one) == strings.ToLower(key) {
			if "" != header.tow {
				return header.three, nil
			}
		}
	}
	return "", nil
}

func (email *Email) GetFrom() (string, error) {
	value, err := email.getHeaderItem("from")
	if nil != err {
		return "", ErrDkimProtocol
	}
	value = strings.TrimSpace(value)
	n := strings.Split(value, " ")
	if len(n) >= 2 {
		from := n[len(n)-1]
		if strings.Count(from, "<") != 1 || strings.Count(from, ">") != 1 || !strings.HasPrefix(from, "<") || !strings.HasSuffix(from, ">") {
			return "", ErrDkimProtocol
		}
		from = strings.TrimLeft(from, "<")
		from = strings.TrimRight(from, ">")
		return from, nil
	}

	if strings.HasPrefix(value, "=?") || strings.HasPrefix(value, "\"") {
		if strings.HasPrefix(value, "\"") {
			idx := strings.LastIndex(value, "\"")
			value = fmt.Sprintf("%s %s", value[:idx+1], value[idx+1:])
		}

		values := strings.FieldsFunc(
			value,
			func(r rune) bool {
				return r == ' ' || r == '\t' || r == '\r' || r == '\n'
			},
		)

		if len(values) > 1 {
			value = strings.Join(values, " ")
			n := strings.Split(value, " ")
			if len(n) >= 2 {
				from := n[len(n)-1]
				if strings.Count(from, "<") != 1 || strings.Count(from, ">") != 1 || !strings.HasPrefix(from, "<") || !strings.HasSuffix(from, ">") {
					return "", ErrDkimProtocol
				}
				from = strings.TrimLeft(from, "<")
				from = strings.TrimRight(from, ">")
				return from, nil
			}
			return "", ErrDkimProtocol
		}

		dec := new(mime.WordDecoder)
		dec.CharsetReader = charsetReader
		header, err := dec.DecodeHeader(value)
		if err != nil {
			return "", ErrEmptyString
		}
		if strings.Count(header, "<") != 1 || strings.Count(header, ">") != 1 {
			return "", ErrDkimProtocol
		}

		header = strings.ReplaceAll(header, "<", " ")
		header = strings.ReplaceAll(header, ">", "")
		n = strings.Split(header, " ")
		if len(n) != 2 {
			return "", ErrDkimProtocol
		}
		return n[1], nil
	} else if strings.HasPrefix(value, "<") && strings.HasSuffix(value, ">") {
		if strings.Count(value, "<") != 1 || strings.Count(value, ">") != 1 {
			return "", ErrDkimProtocol
		}
		value = strings.TrimLeft(value, "<")
		value = strings.TrimRight(value, ">")
		return value, nil
	}

	if strings.Count(value, "<") != 1 || strings.Count(value, ">") != 1 {
		return value, nil
	}

	return "", ErrDkimProtocol
}

func ConvertToFromHash(from string) (*[32]byte, error) {
	if "" == from {
		return nil, ErrEmptyString
	}

	if strings.HasPrefix(from, "@") || strings.HasSuffix(from, "@") || strings.Count(from, "@") != 1 {
		return nil, ErrDkimProtocol
	}

	from = strings.ToLower(from)

	split := strings.Split(from, "@")
	local := split[0]
	domain := split[1]

	if domain == "gmail.com" {
		local = strings.ReplaceAll(local, ".", "")
	}

	len := len(local) + len(domain) + 1
	if len > MAX_EMAIL_LEN || len < MIN_EMAIL_LEN {
		return nil, ErrDkimProtocol
	}

	var buf bytes.Buffer
	buf.WriteString(local)
	buf.WriteString("@")
	buf.WriteString(domain)
	filling := make([]byte, FR_EMAIL_LEN*31-len)
	buf.Write(filling)

	sum := sha256.Sum256(buf.Bytes())
	sum = reverse(sum)
	sum[31] &= 0x1f
	return &sum, nil
}

func (email *Email) GetFromHash() (*[32]byte, error) {
	from, err := email.GetFrom()
	if nil != err {
		return nil, err
	}
	return ConvertToFromHash(from)

}

func reverse(s [32]byte) [32]byte {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

func (email *Email) GetSigHash() ([]byte, error) {
	item, err := email.getHeaderItem("subject")
	if nil != err {
		return nil, ErrDkimProtocol
	}

	item = strings.TrimSpace(item)

	if len(item) < 8 || !strings.HasPrefix(item, "=?") {
		hexString := strings.Split(item, "0x")[1]
		decodeString, err := hex.DecodeString(hexString)
		if nil != err {
			return nil, ErrDkimProtocol
		}
		return decodeString, nil
	}

	lines := bufio.NewReader(bytes.NewReader(([]byte)(item)))
	var build strings.Builder
	dec := new(mime.WordDecoder)
	dec.CharsetReader = charsetReader
	for {
		line, _, error := lines.ReadLine()
		if error == io.EOF {
			break
		} else if error != nil {
			return nil, ErrDkimProtocol
		}
		decode, _ := dec.Decode(strings.TrimSpace(string(line)))
		build.WriteString(decode)
	}
	hexString := strings.Split(build.String(), "0x")[1]
	hexBytes, err := hex.DecodeString(hexString)
	if nil != err {
		return nil, ErrDkimProtocol
	}
	return hexBytes, nil
}

func (email *Email) GetSigPubKey(address common.Address, db *account.AccountDB) (*rsa.PublicKey, error) {
	index := email.DkimHeader.Selector + "@" + email.DkimHeader.Sdid
	n := GetEmailPubKey(address, index, db)
	if 0 == len(n) {
		common.DefaultLogger.Errorf("no pubkey for  %s", index)
		return nil, ErrDkimProtocol
	}

	bigN := new(big.Int)
	_, ok := bigN.SetString(n, 16)
	if !ok {
		common.DefaultLogger.Errorf("error big int. %s", n)
		return nil, ErrDkimProtocol
	}

	return &rsa.PublicKey{
		N: bigN,
		E: 65537,
	}, nil
}

func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	content, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(bytes.ToValidUTF8(content, ([]byte)("\uFFFD"))), nil
}

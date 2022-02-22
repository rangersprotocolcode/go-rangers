package dkim

import (
	"bufio"
	"bytes"
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

var sign_pubkey_map = map[string]string{
	"20161025@gmail.com":         "be23c6064e1907ae147d2a96c8089c751ee5a1d872b5a7be11845056d28384cfb59978c4a91b4ffe90d3dec0616b3926038f27da4e4d254c8c1283bc9dcdabeac500fbf0e89b98d1059a7aa832893b08c9e51fcea476a69511be611250a91b6a1204a22561bb87b79f1985a687851184533d93dfab986fc2c02830c7b12df9cf0e3259e068b974e3f6cf99fa63744c8b5b23629a4efad425fa2b29b3622443373d4c389389ececc5692e0f15b54b9f49b999fd0754db41a4fc16b8236f68555f9546311326e56c1ea1fe858e3c66f3a1282d440e3b487579dd2c198c8b15a5bab82f1516f48c4013063319c4a06789f943c5fc4e7768c2c0d4ce871c3c51a177",
	"20161025@googlemail.com":    "be23c6064e1907ae147d2a96c8089c751ee5a1d872b5a7be11845056d28384cfb59978c4a91b4ffe90d3dec0616b3926038f27da4e4d254c8c1283bc9dcdabeac500fbf0e89b98d1059a7aa832893b08c9e51fcea476a69511be611250a91b6a1204a22561bb87b79f1985a687851184533d93dfab986fc2c02830c7b12df9cf0e3259e068b974e3f6cf99fa63744c8b5b23629a4efad425fa2b29b3622443373d4c389389ececc5692e0f15b54b9f49b999fd0754db41a4fc16b8236f68555f9546311326e56c1ea1fe858e3c66f3a1282d440e3b487579dd2c198c8b15a5bab82f1516f48c4013063319c4a06789f943c5fc4e7768c2c0d4ce871c3c51a177",
	"s201512@qq.com":             "cfb0520e4ad78c4adb0deb5e605162b6469349fc1fde9269b88d596ed9f3735c00c592317c982320874b987bcc38e8556ac544bdee169b66ae8fe639828ff5afb4f199017e3d8e675a077f21cd9e5c526c1866476e7ba74cd7bb16a1c3d93bc7bb1d576aedb4307c6b948d5b8c29f79307788d7a8ebf84585bf53994827c23a5",
	"s201512@foxmail.com":        "cfb0520e4ad78c4adb0deb5e605162b6469349fc1fde9269b88d596ed9f3735c00c592317c982320874b987bcc38e8556ac544bdee169b66ae8fe639828ff5afb4f199017e3d8e675a077f21cd9e5c526c1866476e7ba74cd7bb16a1c3d93bc7bb1d576aedb4307c6b948d5b8c29f79307788d7a8ebf84585bf53994827c23a5",
	"s110527@163.com":            "a9f49a52ec4391363c089ed5c8235ee626ec286fe849a15987af68761cfa5213b418821f35e641dd602e096f15e070fd26359398dd1d5593a7540d1f0d4222fec41f5f44d5854b7e93abb0b33c4fd423ff8fc5684fccf9ef001af881b41cadeb3c79ef5c80430f143a5c9383bb50b3493711f4d3739f7268752bec431b2a8f59",
	"s110527@126.com":            "a9f49a52ec4391363c089ed5c8235ee626ec286fe849a15987af68761cfa5213b418821f35e641dd602e096f15e070fd26359398dd1d5593a7540d1f0d4222fec41f5f44d5854b7e93abb0b33c4fd423ff8fc5684fccf9ef001af881b41cadeb3c79ef5c80430f143a5c9383bb50b3493711f4d3739f7268752bec431b2a8f59",
	"s110527@yeah.net":           "a9f49a52ec4391363c089ed5c8235ee626ec286fe849a15987af68761cfa5213b418821f35e641dd602e096f15e070fd26359398dd1d5593a7540d1f0d4222fec41f5f44d5854b7e93abb0b33c4fd423ff8fc5684fccf9ef001af881b41cadeb3c79ef5c80430f143a5c9383bb50b3493711f4d3739f7268752bec431b2a8f59",
	"s110527@188.com":            "a9f49a52ec4391363c089ed5c8235ee626ec286fe849a15987af68761cfa5213b418821f35e641dd602e096f15e070fd26359398dd1d5593a7540d1f0d4222fec41f5f44d5854b7e93abb0b33c4fd423ff8fc5684fccf9ef001af881b41cadeb3c79ef5c80430f143a5c9383bb50b3493711f4d3739f7268752bec431b2a8f59",
	"s110527@vip.163.com":        "a9f49a52ec4391363c089ed5c8235ee626ec286fe849a15987af68761cfa5213b418821f35e641dd602e096f15e070fd26359398dd1d5593a7540d1f0d4222fec41f5f44d5854b7e93abb0b33c4fd423ff8fc5684fccf9ef001af881b41cadeb3c79ef5c80430f143a5c9383bb50b3493711f4d3739f7268752bec431b2a8f59",
	"s110527@vip.126.com":        "a9f49a52ec4391363c089ed5c8235ee626ec286fe849a15987af68761cfa5213b418821f35e641dd602e096f15e070fd26359398dd1d5593a7540d1f0d4222fec41f5f44d5854b7e93abb0b33c4fd423ff8fc5684fccf9ef001af881b41cadeb3c79ef5c80430f143a5c9383bb50b3493711f4d3739f7268752bec431b2a8f59",
	"s110527@vip.188.com":        "a9f49a52ec4391363c089ed5c8235ee626ec286fe849a15987af68761cfa5213b418821f35e641dd602e096f15e070fd26359398dd1d5593a7540d1f0d4222fec41f5f44d5854b7e93abb0b33c4fd423ff8fc5684fccf9ef001af881b41cadeb3c79ef5c80430f143a5c9383bb50b3493711f4d3739f7268752bec431b2a8f59",
	"s2048@yahoo.com":            "ba85ae7e06d6c39f0c7335066ccbf5efa45ac5d64638c9109a7f0e389fc71a843a75a95231688b6a3f0831c1c2d5cb9b271da0ce200f40754fb4561acb22c0e1ac89512364d74feea9f072894f2a88f084e09485ae9c5f961308295e1bb7e835b87c3bc0bce0b827f8600a11e97c54291b00a07ba817b33ebfa6cc67f5f51bebe258790197851f80943a3bc17572428aa19e4aa949091f9a436aa6e0b3e1773e9ca201441f07a104cce03528c3d15891a9ce03ed2a8ba40dc42e294c3d180ba5ee4488c84722ceaadb69428d2c6026cf47a592a467cc8b15a73ea3753d7f615e518ba614390e6c3796ea37367c4f1a109646d5472e9e28e8d49e84924e648087",
	"dbd5af2cbaf7@mail.com":      "ede596d226cb20962f0813f0f77192bffa52b5ef8668a4eee295ce446ec8f683edbb7ad2373023ff3267d44c1ba792381f68dbee3d17431db3e11f521513f126444a0cc134cb702bd693f7a000be9f0c6b57f2b67ea2462de0ef85c9929b937bd5f58e66882b82b9d23e08648318602c8de499e9b1287b6682a3f2dd3e22e2f5",
	"selector1@outlook.com":      "bd6ca4b6b20bf033bff941af31bbfb70f77f5e88296ecee9815c3ccbd95d3ba00032683cfa28c4365fdcec56f531f28ceee1b72ccc00af475554ac8cfa66e4a17da4e4bee5b11390af467d8064a9bbbc6b9d939ae7cfbfa4885dd9793f0c53e96b9f9329b5a875bb1264f44df33d11639c79f6377349c957944c8df661197663a2293b0e3fa03bbd0c5f4b26bd8e0e4df3575f34dbcfec79d67a330cb0ac8832b5e9b713a1201b84607ebb2437bdf10817d78a07bc6336533e7789ffd25bc305d3dad887db29e19b1a58b220e93df8dc9ce56edaec1911820c9f493e9c515998a6b73f94a7f0652b34fab020ab06285bfc18b7a59884041e148bfbebb8be5109",
	"selector1@hotmail.com":      "bd6ca4b6b20bf033bff941af31bbfb70f77f5e88296ecee9815c3ccbd95d3ba00032683cfa28c4365fdcec56f531f28ceee1b72ccc00af475554ac8cfa66e4a17da4e4bee5b11390af467d8064a9bbbc6b9d939ae7cfbfa4885dd9793f0c53e96b9f9329b5a875bb1264f44df33d11639c79f6377349c957944c8df661197663a2293b0e3fa03bbd0c5f4b26bd8e0e4df3575f34dbcfec79d67a330cb0ac8832b5e9b713a1201b84607ebb2437bdf10817d78a07bc6336533e7789ffd25bc305d3dad887db29e19b1a58b220e93df8dc9ce56edaec1911820c9f493e9c515998a6b73f94a7f0652b34fab020ab06285bfc18b7a59884041e148bfbebb8be5109",
	"20210112@gmail.com":         "abc27154130b1d9463d56bc83121c0a516370eb684fc4885891e88300943abd1809eb572d2d0d0c81343d46f3ed5fcb9470b2c43d0e07cd7bbac89b0c5a6d67d6c49d4b4a6a3f0f311d38738935088ffe7c3b31d986bbe47d844bc17864500269f58e43b8e8a230fe9da51af98f49edfe0150fe5f2697678bc919364a1540a7a1cb40554c878d20d3eca9c4b1a88d0f6ad5b03bf28ac254007f84c917e61d20707c954701d27da03f1c9fd36322e9ff1072d2230842c5798b26568978d005b5c19e0f669119b1da4bb33a69314ffaa9387f6b9c471df57320b16eee7408355f53e778264203341143895f8c22968315721fd756c6a12d3ca010508b23d7817d3",
	"20210112@googlemail.com":    "abc27154130b1d9463d56bc83121c0a516370eb684fc4885891e88300943abd1809eb572d2d0d0c81343d46f3ed5fcb9470b2c43d0e07cd7bbac89b0c5a6d67d6c49d4b4a6a3f0f311d38738935088ffe7c3b31d986bbe47d844bc17864500269f58e43b8e8a230fe9da51af98f49edfe0150fe5f2697678bc919364a1540a7a1cb40554c878d20d3eca9c4b1a88d0f6ad5b03bf28ac254007f84c917e61d20707c954701d27da03f1c9fd36322e9ff1072d2230842c5798b26568978d005b5c19e0f669119b1da4bb33a69314ffaa9387f6b9c471df57320b16eee7408355f53e778264203341143895f8c22968315721fd756c6a12d3ca010508b23d7817d3",
	"protonmail@protonmail.com":  "ca678aeacca0caadf24728d7d3821d41ff736da07ad1f13e185d3b8796da4526585cf867230c4a5fdadbf31e747b47b11b84e762c32e122e0097a8421141eeecc0e4fcbeae733d9ebf239d28f22b31cf9d10964bcda085b27a2350aa50cf40b41ecb441749f2f39d063f6c7c6f280a808b7dc2087c12fce3eeb96707abc0c2a9",
	"smtp@mail.unipass.me":       "be49a095a68715465a37ba5f91ba63cbb37194888d7724207801a4c4c1d2a15dfb50a239c6f279e87c3f3ad8df78dbc704910fdecf24cd53b618ebca3e5004cbe4d1e2deb99d4b32f8583cef51f4289377ca5267f4b002e8c2d98c8115a4ec944799032ada9b633b6d31043dcd6a7a84a3d7e614f861ddcd5a1284c83318a5a1fc5e4fbbc736fa2546b7be3c9ffee190e3960d00541a3b7adbb93d27ff4ea63c76069f6faa901c7ad1b52fe43a86afd54bd886f762afa0459bc470ede8d9de0d4e774f7d6179245dc22cf279268761290891f81cda728683273b195d664c378031a94b44304fdd77ca31872a7656d9c9c8a7b296117bc36364ac653a3ffc8f61",
	"eth@unipass.id":             "8d74c57e6c2736060b667c71bfa0beec86f13b89e29502f537101de6af3dc1a9e1e31d1490ddd739509d6c5ed7faecfed1392665f312939c3f0314700c2f22e38d54d3f57c423101bffa7b807f8c9e0e2619e8ecf63d1afae22ae6269126a8a9ac3b6ecbd8176f8065f275688a5598e0dc8b77a5341873c5024c2d1c0f3e628d",
	"protonmail@pm.me":           "a66408196cdf68bf5c7be5611dcad34f32bdaf19fc1f7f4f3eeff3b833b98af8baf1accc646cfc6aa3d3bcc017471d96b58bddf5b3e3897d9fb6172050fc86da55246122c4cb973ea027d69faf8e0e656cff6d1f2bad70d42d2eedf38ccd8b203a39a9d8aa133dc401a721df31b566cc219eb9ee55256be36a8d0a5f51849c39999d9d0cad3705e5b4a243ab40b9457818d1f27f2b2101e03021201bf94b4093d83e2e9e218c3bb12ee1cad100ef04d2b54ddbb42c6e1b138da18f780dea12cf7cda903556ebc1969b49c5ae0262e84add20afbe5cd11f9087d8fd181081f2169d7501b27019b2684092eef62b0c8c7c8093e3995f919516fe55e7fa01dbbda5",
	"1a1hai@icloud.com":          "d5911f6e47f84db3b64c3648ebb5a127a1bc0f0937489c806c1944fd029dc971590781df114ec072e641cdc5d2245134dc3966c8e91402669a47cc85970a281f268a44b21a4f77a91a52f9606a30c9f22d2e5cb6934263d08388091c56ac1dfbf1beea31e8a613c2a51f550d695a514d38b45c862320a00ea539419a433d6bfdd1978356cbca4b600a570fe582918c4f731a0002068df28b2a193089e6bf951c553b5a6f71aaadcc6669dfa346f322717851a8c22a324041af8736e87de4358860fff057beff3f19046a43adce46c932514988afe1309f87414bd36ed296dacfade2e0caf03235e91a2db27e9ed214bcc6e5cf995b5ef59ce9943d1f4209b6ab",
	"protonmail2@protonmail.com": "c60888c0203e5a2fa001b4bbb81ca6ae5e32c700125484b4e8f20e59a3eb5ce2ecfdae187d546a94d27f257f5dda3ad475a34439bfdc1624066bba0071e738106434fdb044e53aa2728f7fcf987817840a32893774b0bb4ec03cfdad0ca11be20d48c8410d5a3c48cb3f07a4efababac3a186755d6abc70700a3c6b88e823fbc97356117a79c27e54b88a237ca6a1cd3e35ed423dd4abcb195727472346183105960ae2828081d276ce11b8839a722b9a012990e68aca2085dfa4ca20372d6b11f4d50a3d4ac39776c371ca95d88646b65faf573cd87c06c09a11f330996b9f5b44014c369029a01182aa0064ac59e1492b6f2b00f9e0805234d0fa2772a89d1",
	"protonmail2@pm.me":          "b1dfac7052467ccf79f7870f2aa3a514effcbb0da7e5981945b386512de1fd9dd70bae840b13aac4a6083b585228825e7be1e0ea144746598e42ca340279c8039e2d13c066f33ff30bb97c231350c2c3d4169fd9d73d1fd1acf2d650ddeba77852ed8fbb8f1177ac717d4cb3eecfd38317b939ce98a858f1dc0e5dabcaf9be9636b6a24ec74d6dc532496aaa83b2d9f7191aedb595073a99baa5524093c7629f4b39ca20e6c1a17f894e18e5e44fa7ea4a7177bb4038c2bcbbab413e733494bbae41ae70ec059791b2e508f396058b9e6a2c581417f7a4f59332460f08a2565ed057182a1e34a3ece7ab9622b131472104c4fbed30c672012571847bba281b25",
}

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
	for {
		header, rest := getOneHeader(headers)
		if "" != header {
			keyVal := strings.SplitN(header, ":", 2)
			valueVal := strings.TrimLeft(keyVal[1], " ")
			valueVal = strings.TrimRight(valueVal, "\r")
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

	if strings.Count(n[0], "<") > 0 || strings.Count(n[0], ">") > 0 {
		return "", ErrDkimProtocol
	}

	from := n[0]
	from = strings.TrimLeft(from, "<")
	from = strings.TrimRight(from, ">")
	return from, nil
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

func (email *Email) GetSigPubKey() (*rsa.PublicKey, error) {
	if email.DkimHeader == nil {
		return nil, ErrDkimProtocol
	}

	index := email.DkimHeader.Selector + "@" + email.DkimHeader.Sdid
	n, ok := sign_pubkey_map[index]
	if !ok {
		return nil, ErrDkimProtocol
	}
	bigN := new(big.Int)
	_, ok = bigN.SetString(n, 16)
	if !ok {
		fmt.Println("error big int")
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

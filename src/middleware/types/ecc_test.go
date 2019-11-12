package types

import (
	"testing"
	"fmt"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestECC_Sign(t *testing.T) {
	//var ou []Out=[]Out{Out{"eth","aaa","bbb","1000000000","0x7edd0ef9da9cec334a7887966cc8dd71d590eeb7","001"}}
	var ou []byte = []byte{'a', 'b', 'c', 'd', 'e'}
	var ecc = Ecc{2, "10695fe7b429427aa01044d97f48e14e1244d206eda8dfa812996310100f4cd1",
		[]string{"0xf89eebcc07e820f5a8330f52111fa51dd9dfb925", "0x9951146d4fdbd0903d450b315725880a90383f38", "0x7edd0ef9da9cec334a7887966cc8dd71d590eeb7"}}
	by := ecc.Sign(ou)
	by2 := ecc.Sign(ou)
	ret := ecc.Verify(ou, by)
	ret2 := ecc.Verify(ou, by2)
	fmt.Println(ret, ret2)
}

func TestECC_VerifyDeposite(t *testing.T) {
	var ecc = Ecc{2, "10695fe7b429427aa01044d97f48e14e1244d206eda8dfa812996310100f4cd1",
		[]string{"0xf89eebcc07e820f5a8330f52111fa51dd9dfb925", "0x9951146d4fdbd0903d450b315725880a90383f38", "0x7edd0ef9da9cec334a7887966cc8dd71d590eeb7"}}

	////msg := Sha256([]byte("abcde"))
	////fmt.Println("hash:",hexutil.Encode(msg))
	//msg := []byte("abcde")
	//fmt.Println("sign:",hexutil.Encode(ecc.Sign(msg)))

	//coin
	info := Incoming{"ETH.ETH", "0x1234567", "1.23", "0xffffeeee"}
	var de C2wDeposit = C2wDeposit{info.Tp, info.Amount, info.Txid}
	dedata, _ := json.Marshal(de)

	var Signed [][]byte
	Signed = append(Signed, ecc.Sign(info.ToJson()))
	fmt.Println(Signed[0])
	Signed = append(Signed, ecc.Sign(info.ToJson()))

	var signstr string
	for _, v := range Signed {
		signstr += hexutil.Encode(v)
		signstr += "|"
	}
	signstr = signstr[:len(signstr)-1]

	var ba = TxJson{Source: info.Uid, Target: "", Type: 201, Data: string(dedata), Hash: "", Sign: signstr}
	ret := ecc.VerifyDeposit(ba)
	fmt.Println("coin VerifyDeposite", ret)
	//nft
	info2 := IncomingNft{"nft-tokenid 1", "0xaaaaaaaaaa", "0x1234567", "nft-setid", "nft-symbol", "nft-name", "0xffffeeee", "2019-01-01 00:00:00", "nft-value", "nft-txid"}
	var de2 C2wDepositNft = C2wDepositNft{info2.Setid, info2.Name, info2.Symbol, info2.TokenId, info2.Creator, info2.CreateTime, info2.Userid, info2.Info, info2.Txid}
	dedata2, _ := json.Marshal(de2)

	var Signed2 [][]byte
	Signed2 = append(Signed2, ecc.Sign(info2.ToJson()))
	Signed2 = append(Signed2, ecc.Sign(info2.ToJson()))

	var signstr2 string
	for _, v := range Signed2 {
		signstr2 += hexutil.Encode(v)
		signstr2 += "|"
	}
	signstr2 = signstr2[:len(signstr2)-1]

	var ba2 = TxJson{Source: info2.Userid, Target: info2.Gameid, Type: 203, Data: string(dedata2), Hash: "", Sign: signstr2}
	ret2 := ecc.VerifyDeposit(ba2)
	fmt.Println("nft VerifyDeposite", ret2)

	//ft
	info3 := IncomingFt{"0x1111111111", "3.21", "ft-setid", "0xffffeeee"}
	var de3 C2wDepositFt = C2wDepositFt{info3.Name, info3.Amount, info3.Txid}
	dedata3, _ := json.Marshal(de3)

	var Signed3 [][]byte
	Signed3 = append(Signed3, ecc.Sign(info3.ToJson()))
	Signed3 = append(Signed3, ecc.Sign(info3.ToJson()))

	var signstr3 string
	for _, v := range Signed3 {
		signstr3 += hexutil.Encode(v)
		signstr3 += "|"
	}
	signstr3 = signstr3[:len(signstr3)-1]

	var ba3 = TxJson{Source: info3.Userid, Target: "", Type: 202, Data: string(dedata3), Hash: "", Sign: signstr3}
	ret3 := ecc.VerifyDeposit(ba3)
	fmt.Println("ft VerifyDeposite", ret3)
}

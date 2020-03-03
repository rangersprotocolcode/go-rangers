package types

import (
	"testing"
	"encoding/json"
	"fmt"
	"x/src/common"
)

func TestCoinerVerifySign(t *testing.T) {
	threshold := 2
	privateKey := "10695fe7b429427aa01044d97f48e14e1244d206eda8dfa812996310100f4cd1"
	signer0 := "0xf89eebcc07e820f5a8330f52111fa51dd9dfb925"
	signer1 := "0x9951146d4fdbd0903d450b315725880a90383f38"
	signer2 := "0x7edd0ef9da9cec334a7887966cc8dd71d590eeb7"

	CoinerSignVerifier = Ecc{SignLimit: threshold, Privkey: privateKey, Whitelist: []string{signer0, signer1, signer2}}
	if nil != common.DefaultLogger {
		common.DefaultLogger.Debugf("coiner sign verifier:%v", CoinerSignVerifier)
	}


	//str := `{"source":"0x124cbb42cfb5b9dd75701ac2d1e0e623c2795fc1adc7caea29ed29ae1ca50b13","target":"","type":201,"data":"{\"ChainType\":\"ETH.ETH\",\"Amount\":\"0.00001\",\"Addr\":\"0x69fC10174057A672FbC9130c2B0A8A460FbEa2c2\",\"TxId\":\"0xaadead7eda1298f29868bbc908677596e7831a3c3214e1bdedc567dad477e138\"}","hash":"0xaadead7eda1298f29868bbc908677596e7831a3c3214e1bdedc567dad477e138","sign":"0x4d443d56c6b794f514dadda7906ece8ba0012019944a7e900f01bc1be441d43d1ef77204a78fdf971f123ac2e35730f550b2c364991462f4e1a7b25fed6cc78e00|0x900948a1e2e0b5abbd808d0ae0d84762d5e582cbdc17f9300ed448db48879ad50a34eb0829595eec6c09a7c1e5467f3d52b60532aca4448bcc89677dadd1e6d700","RequestId":0}`
	//str:=`{"source":"0x124cbb42cfb5b9dd75701ac2d1e0e623c2795fc1adc7caea29ed29ae1ca50b13","target":"","type":201,"data":"{\"ChainType\":\"ETH.ETH\",\"Amount\":\"0.00001\",\"Addr\":\"0x69fC10174057A672FbC9130c2B0A8A460FbEa2c2\",\"TxId\":\"0x6b8874ba92246db92f0a83368af71cc60b7d23193a2b2232686ff156cd09d455\"}","sign":"0xb5bf73fe05d85e60b1d9e0695f441fb3f449d1b1d5b424a8d6aff3f2cde9261a71f5e5b800ff6d61dceecfab5ded4ed880e1f2c3281339e382a2e2c103a79d5d00|0xe4e11d3cc7d2d52a693339a91cd2ff9afac0726acc434bd807437d04567157381c0009c98ea801b1312fe67c486a723aa51ee161e630f267697c0af61a7ed89a00","RequestId":0}`
	str:=`{"source":"0x124cbb42cfb5b9dd75701ac2d1e0e623c2795fc1adc7caea29ed29ae1ca50b13","target":"","type":201,"data":"{\"ChainType\":\"ETH.ETH\",\"Amount\":\"0.000001\",\"Addr\":\"0x69fC10174057A672FbC9130c2B0A8A460FbEa2c2\",\"TxId\":\"0xfed711fc1324a868a2dfcf4d1cb28d064108934be0f2983a722b5bc90db41215\"}","hash":"0xfed711fc1324a868a2dfcf4d1cb28d064108934be0f2983a722b5bc90db41215","sign":"0x1d06ee5f70bd3dccca098a284511e2647c9ef6dbf7dd59c40d776df14cf6f31111152be95ae252f0f386bbea2ab1973f51f3e8467262591ca963d71d3c2f4fca00|0x48297b80bcca6fb72557ceaa4bc3d77ec054ceb8f98258aee28646b16217ca14450986540881db0d5da90e8ec2a3613f0b08cb342a519d3b47a72900ec39beee00","RequestId":0}`
	var txJson TxJson
	err := json.Unmarshal([]byte(str), &txJson)
	if err != nil {
		panic("Json unmarshal coiner msg err:"+err.Error())
		return
	}
	result :=CoinerSignVerifier.VerifyDeposit(txJson)
	fmt.Printf("%v \n",result)
}
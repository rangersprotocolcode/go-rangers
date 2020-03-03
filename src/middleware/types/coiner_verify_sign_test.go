package types

import (
	"testing"
	"encoding/json"
	"fmt"
)

func TestCoinerVerifySign(t *testing.T) {

	str := `{"source":"0x124cbb42cfb5b9dd75701ac2d1e0e623c2795fc1adc7caea29ed29ae1ca50b13","target":"","type":201,"data":"{\"ChainType\":\"ETH.ETH\",\"Amount\":\"0.00001\",\"Addr\":\"0x69fC10174057A672FbC9130c2B0A8A460FbEa2c2\",\"TxId\":\"0xaadead7eda1298f29868bbc908677596e7831a3c3214e1bdedc567dad477e138\"}","hash":"0xaadead7eda1298f29868bbc908677596e7831a3c3214e1bdedc567dad477e138","sign":"0x4d443d56c6b794f514dadda7906ece8ba0012019944a7e900f01bc1be441d43d1ef77204a78fdf971f123ac2e35730f550b2c364991462f4e1a7b25fed6cc78e00|0x900948a1e2e0b5abbd808d0ae0d84762d5e582cbdc17f9300ed448db48879ad50a34eb0829595eec6c09a7c1e5467f3d52b60532aca4448bcc89677dadd1e6d700","RequestId":0}`
	var txJson TxJson
	err := json.Unmarshal([]byte(str), &txJson)
	if err != nil {
		panic("Json unmarshal coiner msg err:"+err.Error())
		return
	}
	result :=CoinerSignVerifier.VerifyDeposit(txJson)
	fmt.Printf("%v \n",result)
}
package types

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"fmt"
	"testing"
)

func TestJSONMiner(t *testing.T) {
	id := "0x613b5c50c736ccd8da80300a42c135d21d86181f14b1e8fcc07f530aabb5e8df"

	miner := Miner{
		Id:    common.FromHex(id),
		Stake: 1000,
	}

	data, _ := json.Marshal(miner)
	fmt.Println(string(data))

	str := `{"id":"YTtcUMc2zNjagDAKQsE10h2GGB8Usej8wH9TCqu16N8=","stake":1000}`
	var miner2 Miner
	err := json.Unmarshal([]byte(str), &miner2)
	if err == nil {
		fmt.Println(miner2.Stake)
	} else {
		t.Fatalf(err.Error())
	}

	fmt.Println(common.ToHex(miner2.Id))

	stake := uint64(1000)
	fmt.Println(utility.Float64ToBigInt(float64(stake)))
}

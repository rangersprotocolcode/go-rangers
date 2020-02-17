package types

import (
	"testing"
	"encoding/json"
	"fmt"
	"x/src/utility"
)

func TestJSONMiner(t *testing.T) {
	miner := Miner{
		Stake: 1000,
	}

	data, _ := json.Marshal(miner)
	fmt.Println(string(data))

	str := `{"Id":null,"PublicKey":null,"VrfPublicKey":null,"Type":0,"Stake":1000,"ApplyHeight":0,"AbortHeight":0,"Status":0}`

	var miner2 Miner
	err := json.Unmarshal([]byte(str), &miner2)
	if err == nil {
		fmt.Println(miner2.Stake)
	} else {
		t.Fatalf(err.Error())
	}

	stake := uint64(1000)
	fmt.Println(utility.Float64ToBigInt(float64(stake)))
}

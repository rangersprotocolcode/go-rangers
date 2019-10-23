package statemachine

import (
	"testing"
	"encoding/json"
	"fmt"
	"x/src/middleware/types"
	"strings"
)

func TestTransferData(t *testing.T) {
	data := transferdata{}
	data.Source = "0x1234"
	data.Balance = "2"

	//bnt := make(map[string]string)
	//bnt["ETH.ETH"] = "0.001"
	//bnt["NEO.CGAS"] = "100"
	//data.Bnt = bnt
	//
	//ft := make(map[string]string)
	//data.Ft = ft
	//ft["SET_ID_1"] = "189"
	//ft["SET_ID_2"] = "1"

	//nft := make([]map[string][]string, 2)
	//data.Nft = nft
	//nft1 := make(map[string][]string)
	//nft2 := make(map[string][]string)
	//nft[0] = nft1
	//nft[1] = nft2
	//nft1list := []string{"1", "1002", "20938"}
	//nft1["set_id_1"] = nft1list
	//nft2list := []string{"2", "222", "23232"}
	//nft2["set_id_1"] = nft2list

	nftlist := make([]types.NFTID, 5)
	nftlist[0] = types.NFTID{SetId: "set_id_1", Id: "1"}
	nftlist[1] = types.NFTID{SetId: "set_id_1", Id: "1002"}
	nftlist[2] = types.NFTID{SetId: "set_id_2", Id: "2"}
	nftlist[3] = types.NFTID{SetId: "set_id_1", Id: "20938"}
	nftlist[4] = types.NFTID{SetId: "set_id_2", Id: "222"}
	data.transfer(nftlist)

	dataBytes, _ := json.Marshal(data)

	fmt.Println(string(dataBytes))
}

func TestStringPrefix(t *testing.T) {
	s := "official-asd"
	fmt.Println(strings.TrimPrefix(s, "official-"))

	fmt.Println(strings.TrimPrefix("eth.Eth", "official-"))
}

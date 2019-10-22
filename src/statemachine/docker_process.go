package statemachine

import (
	"x/src/middleware/types"
	"fmt"
	"net/url"
	"x/src/common"
	"io/ioutil"
	"encoding/json"
)

func (d *StateMachineManager) Process(name string, kind string, nonce string, payload string, tx *types.Transaction) *types.OutputMessage {
	prefix := d.getUrlPrefix(name)
	if 0 == len(prefix) {
		return nil
	}

	path := fmt.Sprintf("%sprocess", prefix)
	values := url.Values{}
	values["payload"] = []string{payload}
	values["transfer"] = []string{d.generateTransfer(tx)}

	resp, err := d.httpClient.PostForm(path, values)
	if err != nil {
		common.DefaultLogger.Debugf("Docker process post error.Path:%s,values:%v,error:%s", path, values, values, err.Error())
		return nil
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		common.DefaultLogger.Debugf("Docker process read response error:%s", err.Error())
		return nil
	}

	output := types.OutputMessage{}
	if err = json.Unmarshal(body, &output); nil != err {
		common.DefaultLogger.Debugf("Docker process result unmarshal error:%s", err.Error())
		return nil
	}

	return &output
}

func (d *StateMachineManager) generateTransfer(tx *types.Transaction) string {
	result := transferdata{}
	if nil != tx && 0 != len(tx.ExtraData) {
		result.Source = tx.Source

		mm := make(map[string]types.TransferData, 0)
		json.Unmarshal([]byte(tx.ExtraData), &mm)

		for _, v := range mm {
			result.Balance = v.Balance
			result.Ft = v.FT
			result.Bnt = v.Coin
			result.transfer(v.NFT)
		}

	}

	data, _ := json.Marshal(result)
	return string(data)
}

type transferdata struct {
	Source  string                `json:"source,omitempty"`
	Bnt     map[string]string     `json:"bnt,omitempty"`
	Ft      map[string]string     `json:"ft,omitempty"`
	Nft     []map[string][]string `json:"nft,omitempty"`
	Balance string                `json:"balance,omitempty"`
}

func (self *transferdata) transfer(nftList []types.NFTID) {
	if nil == nftList || 0 == len(nftList) {
		return
	}

	nft := make([]map[string][]string, 0)

	for _, item := range nftList {
		flag := false
		for _, entry := range nft {
			set := entry[item.SetId]
			if set != nil && 0 != len(set) {
				flag = true
				entry[item.SetId] = append(set, item.Id)
				break
			}
		}

		if !flag {
			list := []string{item.Id}
			set := make(map[string][]string)
			set[item.SetId] = list
			nft = append(nft, set)
		}

	}

	if 0 != len(nft) {
		self.Nft = nft
	}
}

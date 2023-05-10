package network

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

// RPCReqObj 完整的rpc请求体
type rpcReqObj struct {
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Jsonrpc string        `json:"jsonrpc"`
	ID      uint          `json:"id"`
}

// RPCResObj 完整的rpc返回体
type RPCResObj struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      uint        `json:"id"`
	Result  interface{} `json:"result,omitempty"`
}

func JSONRPCPost(url string, method string, params ...interface{}) (*RPCResObj, error) {
	obj := rpcReqObj{
		Method: method,
		Params: params,
	}
	objBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(
		url,
		"application/json",
		bytes.NewReader(objBytes),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	responseBytes, err := ioutil.ReadAll(resp.Body)
	//fmt.Println(string(responseBytes))

	if err != nil {
		return nil, err
	}
	var resJSON RPCResObj
	if err := json.Unmarshal(responseBytes, &resJSON); err != nil {
		return nil, err
	}
	return &resJSON, nil
}

// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package cli

import (
	"bytes"
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
)

func getMessage(addr string, port uint, method string, params ...interface{}) (string, error) {
	res, err := rpcPost(addr, port, method, params...)
	if err != nil {
		return "", err
	}
	if res.Error != nil {
		return "", errors.New(res.Error.Message)
	}
	return res.Result.Message, nil
}

func rpcPost(addr string, port uint, method string, params ...interface{}) (*RPCResObj, error) {
	obj := RPCReqObj{
		Method:  method,
		Params:  params,
		Jsonrpc: "2.0",
		ID:      1,
	}
	objBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(
		fmt.Sprintf("http://%s:%d", addr, port),
		"application/json",
		bytes.NewReader(objBytes),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	responseBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var resJSON RPCResObj
	if err := json.Unmarshal(responseBytes, &resJSON); err != nil {
		return nil, err
	}
	return &resJSON, nil
}

func genHash(hash string) []byte {
	bytes3 := []byte(hash)
	return common.Sha256(bytes3)
}

func getRandomString(l int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytess := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(utility.GetTime().UnixNano()))
	for i := 0; i < l; i++ {
		result = append(result, bytess[r.Intn(len(bytess))])
	}
	return string(result)
}

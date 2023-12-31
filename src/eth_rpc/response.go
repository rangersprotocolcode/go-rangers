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
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package eth_rpc

import (
	"encoding/json"
)

const (
	jsonrpcVersion = "2.0"
)

type jsonResponse interface {
	encodeJson() ([]byte, error)
}

type jsonSuccessResponse struct {
	Version string      `json:"jsonrpc"`
	Id      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result"`
}

type jsonError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type jsonErrResponse struct {
	Version string      `json:"jsonrpc"`
	Id      interface{} `json:"id,omitempty"`
	Error   jsonError   `json:"error"`
}

func (r jsonSuccessResponse) encodeJson() ([]byte, error) {
	return json.Marshal(r)
}

func (r jsonErrResponse) encodeJson() ([]byte, error) {
	return json.Marshal(r)
}

func makeResponse(returnValue interface{}, error error, id interface{}) jsonResponse {
	if error == nil {
		successResponse := jsonSuccessResponse{Version: jsonrpcVersion, Id: id, Result: returnValue}
		return successResponse
	} else {
		errResponse := jsonErrResponse{Version: jsonrpcVersion, Id: id}

		errMsg := jsonError{Code: defaultErrorCode, Message: error.Error(), Data: returnValue}
		ec, ok := error.(Error)
		if ok {
			errMsg.Code = ec.ErrorCode()
		}
		de, ok := error.(DataError)
		if ok {
			errMsg.Data = de.ErrorData()
		}

		errResponse.Error = errMsg
		return errResponse
	}
}

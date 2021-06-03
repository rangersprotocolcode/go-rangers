package eth_rpc

import (
	"encoding/json"
)

const (
	jsonrpcVersion = "1.0"
)

type jsonSuccessResponse struct {
	Version string      `json:"jsonrpc"`
	Id      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result"`
}

type jsonError struct {
	Code    int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type jsonErrResponse struct {
	Version string      `json:"jsonrpc"`
	Id      interface{} `json:"id,omitempty"`
	Error   jsonError   `json:"result"`
}

func makeResponse(returnValue interface{}, error Error, id uint64) string {
	var response []byte
	if error == nil {
		successResponse := jsonSuccessResponse{Version: jsonrpcVersion, Id: id, Result: returnValue}
		response, _ = json.Marshal(successResponse)
	} else {
		err := jsonError{Code: error.ErrorCode(), Message: error.Error(), Data: returnValue}
		errResponse := jsonErrResponse{Version: jsonrpcVersion, Id: id, Error: err}
		response, _ = json.Marshal(errResponse)
	}
	return string(response)
}

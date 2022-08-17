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
	Code    int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type jsonErrResponse struct {
	Version string      `json:"jsonrpc"`
	Id      interface{} `json:"id,omitempty"`
	Error   jsonError   `json:"result"`
}

func (r jsonSuccessResponse) encodeJson() ([]byte, error) {
	return json.Marshal(r)
}

func (r jsonErrResponse) encodeJson() ([]byte, error) {
	return json.Marshal(r)
}

func makeResponse(returnValue interface{}, error Error, id interface{}) jsonResponse {
	if error == nil {
		successResponse := jsonSuccessResponse{Version: jsonrpcVersion, Id: id, Result: returnValue}
		return successResponse
	} else {
		err := jsonError{Code: error.ErrorCode(), Message: error.Error(), Data: returnValue}
		errResponse := jsonErrResponse{Version: jsonrpcVersion, Id: id, Error: err}
		return errResponse
	}
}

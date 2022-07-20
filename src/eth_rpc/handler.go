package eth_rpc

import (
	"bytes"
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/notify"
	"com.tuntun.rocket/node/src/network"
	"encoding/json"
	"fmt"
	"reflect"
)

const sendRawTransactionMethod = "eth_sendRawTransaction"

type execFunc struct {
	receiver reflect.Value  // receiver of method
	method   reflect.Method // callback
	argTypes []reflect.Type // input argument types
	errPos   int            // err return idx, of -1 when method cannot return error
}

var handler ethMsgHandler
var logger log.Logger

type ethMsgHandler struct {
	service map[string]*execFunc
}

func InitEthMsgHandler() {
	handler = ethMsgHandler{}
	index := common.GlobalConf.GetString("instance", "index", "")
	logger = log.GetLoggerByIndex(log.ETHRPCLogConfig, index)

	handler.registerAPI(&ethAPIService{})
	notify.BUS.Subscribe(notify.ClientETHRPC, handler.process)
}

func (handler ethMsgHandler) process(message notify.Message) {
	singleMessage, single := message.GetData().(*notify.ETHRPCMessage)
	if single {
		logger.Debugf("Rcv single eth prc message.requestId: %d, session id: %s", singleMessage.RequestId, singleMessage.SessionId)
		response := handler.processSingleRequest(singleMessage.Message)
		responseJson, err := json.Marshal(response)
		if err != nil {
			logger.Debugf("marshal err:%v", err)
		}
		logger.Debugf("Response:%s,socketRequestId:%v,sessionId:%v", string(responseJson), singleMessage.RequestId, singleMessage.SessionId)
		network.GetNetInstance().SendToJSONRPC(responseJson, singleMessage.SessionId, singleMessage.RequestId)
		return
	}

	batchMessage, batch := message.GetData().(*notify.ETHRPCBatchMessage)
	if batch {
		logger.Debugf("Rcv batch eth prc message.requestId: %d, session id: %s", batchMessage.RequestId, batchMessage.SessionId)
		response := handler.processBatchRequest(batchMessage.Message)
		responseJson, _ := json.Marshal(response)
		logger.Debugf("Response:%s,socketRequestId:%v,sessionId:%v", string(responseJson), batchMessage.RequestId, batchMessage.SessionId)
		network.GetNetInstance().SendToJSONRPC(responseJson, batchMessage.SessionId, batchMessage.RequestId)
		return
	}

}

func (handler ethMsgHandler) processBatchRequest(ethRpcMessage []notify.ETHRPCPiece) []jsonResponse {
	result := make([]jsonResponse, 0)
	if ethRpcMessage == nil {
		return result
	}

	for _, msg := range ethRpcMessage {
		response := handler.processSingleRequest(msg)
		result = append(result, response)
	}
	return result
}

func (handler ethMsgHandler) processSingleRequest(ethRpcMessage notify.ETHRPCPiece) jsonResponse {
	logger.Debugf("Method:%s,params:%s,nonce:%d,id:%v", ethRpcMessage.Method, ethRpcMessage.Params, ethRpcMessage.Nonce, ethRpcMessage.Id)
	handlerFunc, arguments, err := handler.parseRequest(ethRpcMessage)
	var response jsonResponse
	if err != nil {
		response = makeResponse(nil, err, ethRpcMessage.Id)
	} else {
		returnValue, err := handler.exec(handlerFunc, arguments, ethRpcMessage.Method, ethRpcMessage.Nonce, string(ethRpcMessage.Params))
		response = makeResponse(returnValue, err, ethRpcMessage.Id)
	}
	return response
}

func (handler *ethMsgHandler) registerAPI(service interface{}) {
	if handler.service == nil {
		handler.service = make(map[string]*execFunc, 0)
	}
	serviceType := reflect.TypeOf(service)
	serviceValue := reflect.ValueOf(service)

METHODS:
	for m := 0; m < serviceType.NumMethod(); m++ {
		method := serviceType.Method(m)
		mtype := method.Type
		mname := formatName(method.Name)
		if method.PkgPath != "" { // method must be exported
			continue
		}

		var h execFunc
		h.receiver = serviceValue
		h.method = method
		h.errPos = -1

		firstArg := 1
		numIn := mtype.NumIn()

		// determine method arguments, ignore first arg since it's the receiver type
		// Arguments must be exported or builtin types
		h.argTypes = make([]reflect.Type, numIn-firstArg)
		for i := firstArg; i < numIn; i++ {
			argType := mtype.In(i)
			if !isExportedOrBuiltinType(argType) {
				continue METHODS
			}
			h.argTypes[i-firstArg] = argType
		}

		// check that all returned values are exported or builtin types
		for i := 0; i < mtype.NumOut(); i++ {
			if !isExportedOrBuiltinType(mtype.Out(i)) {
				continue METHODS
			}
		}

		// when a method returns an error it must be the last returned value
		h.errPos = -1
		for i := 0; i < mtype.NumOut(); i++ {
			if isErrorType(mtype.Out(i)) {
				h.errPos = i
				break
			}
		}

		if h.errPos >= 0 && h.errPos != mtype.NumOut()-1 {
			continue METHODS
		}

		switch mtype.NumOut() {
		case 0, 1, 2, 3:
			if mtype.NumOut() == 2 && h.errPos == -1 { // method must one return value and 1 error
				continue METHODS
			}
			if mtype.NumOut() == 3 && h.errPos == -1 { // method must one return value and 1 error
				continue METHODS
			}
			handler.service[mname] = &h
		}
	}
}

func (handler ethMsgHandler) parseRequest(ethRpcMessage notify.ETHRPCPiece) (handlerFunc *execFunc, arguments []reflect.Value, error Error) {
	handlerFunc = handler.service[ethRpcMessage.Method]
	if handlerFunc == nil {
		return nil, nil, &methodNotFoundError{ethRpcMessage.Method}
	}

	args, err := parseRequestArguments(handlerFunc.argTypes, ethRpcMessage.Params)
	if err != nil {
		return handlerFunc, nil, &invalidParamsError{err.Error()}
	}

	// regular RPC call, prepare arguments
	if len(args) != len(handlerFunc.argTypes) {
		rpcErr := &invalidParamsError{fmt.Sprintf("%s expects %d parameters, got %d", ethRpcMessage.Method, len(handlerFunc.argTypes), len(args))}
		return handlerFunc, nil, rpcErr
	}

	arguments = []reflect.Value{handlerFunc.receiver}
	if len(args) > 0 {
		arguments = append(arguments, args...)
	}

	return handlerFunc, arguments, nil
}

// execute RPC method and return result
func (handler ethMsgHandler) exec(handlerFunc *execFunc, arguments []reflect.Value, method string, nonce uint64, params string) (interface{}, Error) {
	reply := handlerFunc.method.Func.Call(arguments)
	if len(reply) == 0 {
		return nil, nil
	}
	if handlerFunc.errPos >= 0 { // test if method returned an error
		if !reply[handlerFunc.errPos].IsNil() {
			e := reply[handlerFunc.errPos].Interface().(error)
			logger.Debugf("after exec.error:%v", e)
			return &callbackError{e.Error()}, nil
		}
	}
	return reply[0].Interface(), nil
}

// ParseRequestArguments tries to parse the given params (json.RawMessage) with the given
// types. It returns the parsed values or an error when the parsing failed.
func parseRequestArguments(argTypes []reflect.Type, params interface{}) ([]reflect.Value, Error) {
	if args, ok := params.(json.RawMessage); !ok {
		return nil, &invalidParamsError{"Invalid params supplied"}
	} else {
		return parsePositionalArguments(args, argTypes)
	}
}

// parsePositionalArguments tries to parse the given args to an array of values with the
// given types. It returns the parsed values or an error when the args could not be
// parsed. Missing optional arguments are returned as reflect.Zero values.
func parsePositionalArguments(rawArgs json.RawMessage, types []reflect.Type) ([]reflect.Value, Error) {
	// Read beginning of the args array.
	dec := json.NewDecoder(bytes.NewReader(rawArgs))
	if tok, _ := dec.Token(); tok != json.Delim('[') {
		return nil, &invalidParamsError{"non-array args"}
	}
	// Read args.
	args := make([]reflect.Value, 0, len(types))
	for i := 0; dec.More(); i++ {
		if i >= len(types) {
			return nil, &invalidParamsError{fmt.Sprintf("too many arguments, want at most %d", len(types))}
		}
		argval := reflect.New(types[i])
		if err := dec.Decode(argval.Interface()); err != nil {
			return nil, &invalidParamsError{fmt.Sprintf("invalid argument %d: %v", i, err)}
		}
		if argval.IsNil() && types[i].Kind() != reflect.Ptr {
			return nil, &invalidParamsError{fmt.Sprintf("missing value for required argument %d", i)}
		}
		args = append(args, argval.Elem())
	}
	// Read end of args array.
	if _, err := dec.Token(); err != nil {
		return nil, &invalidParamsError{err.Error()}
	}
	// Set any missing args to nil.
	for i := len(args); i < len(types); i++ {
		if types[i].Kind() != reflect.Ptr {
			return nil, &invalidParamsError{fmt.Sprintf("missing value for required argument %d", i)}
		}
		args = append(args, reflect.Zero(types[i]))
	}
	return args, nil
}

package eth_rpc

import "fmt"

type Error interface {
	Error() string  // returns the message
	ErrorCode() int // returns the code
}

// request is for an unknown service
type methodNotFoundError struct {
	method string
}

func (e *methodNotFoundError) ErrorCode() int { return -32601 }

func (e *methodNotFoundError) Error() string {
	return fmt.Sprintf("The method %s does not exist/is not available", e.method)
}

// received message isn't a valid request
type invalidRequestError struct{ message string }

func (e *invalidRequestError) ErrorCode() int { return -32600 }

func (e *invalidRequestError) Error() string { return e.message }

// received message is invalid
type invalidMessageError struct{ message string }

func (e *invalidMessageError) ErrorCode() int { return -32700 }

func (e *invalidMessageError) Error() string { return e.message }

// unable to decode supplied params, or an invalid number of parameters
type invalidParamsError struct{ message string }

func (e *invalidParamsError) ErrorCode() int { return -32602 }

func (e *invalidParamsError) Error() string { return e.message }

// logic error, callback returned an error
type callbackError struct{ message string }

func (e *callbackError) ErrorCode() int { return -32000 }

func (e *callbackError) Error() string { return e.message }

// issued when a request is received after the server is issued to stop.
type shutdownError struct{}

func (e *shutdownError) ErrorCode() int { return -32000 }

func (e *shutdownError) Error() string { return "server is shutting down" }
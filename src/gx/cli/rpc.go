package cli

import (
	"x/src/gx/rpc"
	"net"

	"x/src/common"
	"fmt"
	"strings"
	"sync"
)

// startHTTP initializes and starts the HTTP RPC endpoint.
func startHTTP(endpoint string, apis []rpc.API, modules []string, cors []string, vhosts []string) error {
	// Short circuit if the HTTP endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}
	// Generate the whitelist based on the allowed modules
	whitelist := make(map[string]bool)
	for _, module := range modules {
		whitelist[module] = true
	}
	// Register all the APIs exposed by the services
	handler := rpc.NewServer()
	for _, api := range apis {
		if whitelist[api.Namespace] || (len(whitelist) == 0 && api.Public) {
			if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
				return err
			}
		}
	}
	// All APIs registered, start the HTTP listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return err
	}
	go rpc.NewHTTPServer(cors, vhosts, handler).Serve(listener)
	//go rpc.NewWSServer(cors, handler).Serve(listener)
	return nil
}

var GtasAPIImpl *GtasAPI

// StartRPC RPC 功能
func StartRPC(host string, port uint) error {
	var err error
	GtasAPIImpl = &GtasAPI{}
	gxLock = &sync.RWMutex{}
	apis := []rpc.API{
		{Namespace: "GTAS", Version: "1", Service: GtasAPIImpl, Public: true},
	}
	for plus := 0; plus < 40; plus++ {
		err = startHTTP(fmt.Sprintf("%s:%d", host, port+uint(plus)), apis, []string{}, []string{}, []string{})
		if err == nil {
			if nil != common.DefaultLogger {
				common.DefaultLogger.Infof("RPC serving on http://%s:%d\n", host, port+uint(plus))
			}
			return nil
		}
		if strings.Contains(err.Error(), "address already in use") {
			if nil != common.DefaultLogger {
				common.DefaultLogger.Infof("address: %s:%d already in use\n", host, port+uint(plus))
			}
			continue
		}
		return err
	}
	return err
}

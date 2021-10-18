// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package cli

import (
	"com.tuntun.rocket/node/src/gx/rpc"
	"net"

	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
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
func StartRPC(host string, port uint, privateKey string) error {
	var err error
	GtasAPIImpl = &GtasAPI{}
	GtasAPIImpl.privateKey = privateKey
	GtasAPIImpl.logger = log.GetLoggerByIndex(log.RPCLogConfig, common.GlobalConf.GetString("instance", "index", ""))

	gxLock = &sync.RWMutex{}
	apis := []rpc.API{
		{Namespace: "Rocket", Version: "1", Service: GtasAPIImpl, Public: true},
		{Namespace: "Rangers", Version: "1", Service: GtasAPIImpl, Public: true},
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

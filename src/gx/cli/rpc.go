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

package cli

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/eth_rpc"
	"com.tuntun.rangers/node/src/gx/rpc"
	"com.tuntun.rangers/node/src/middleware/log"
	"fmt"
	"net"
	"net/http"
	"strconv"
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

func startHttps(httpPort uint, privateKey string) error {
	endpoint := fmt.Sprintf("0.0.0.0:%d", httpPort+1000)
	fmt.Println("self http: " + endpoint)
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return err
	}
	server := &http.Server{Handler: NewSelfServer(privateKey)}
	go server.Serve(listener)
	common.DefaultLogger.Infof("Self Http serving on %s for dev\n", endpoint)

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
	if common.IsSub() {
		apis = append(apis, rpc.API{Namespace: common.Genesis.Name, Version: "1", Service: GtasAPIImpl, Public: true})
	}

	for plus := 0; plus < 40; plus++ {
		err = startHTTP(fmt.Sprintf("%s:%d", host, port+uint(plus)), apis, []string{}, []string{}, []string{})
		if err == nil {
			if nil != common.DefaultLogger {
				common.DefaultLogger.Infof("RPC serving on http://%s:%d\n", host, port+uint(plus))
			}
			break
		}
		if strings.Contains(err.Error(), "address already in use") {
			if nil != common.DefaultLogger {
				common.DefaultLogger.Infof("address: %s:%d already in use\n", host, port+uint(plus))
			}
			continue
		}
		return err
	}

	err = startHttps(port, privateKey)
	return err
}

func StartJSONRPCHttp(port uint) error {
	endpoint := fmt.Sprintf("0.0.0.0:%d", port)
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return err
	}
	server := &http.Server{Handler: NewETHServer()}
	go server.Serve(listener)
	common.DefaultLogger.Infof("JSONRPC http serving on %s \n", endpoint)
	return nil
}

func StartJSONRPCWS(port uint) error {
	endpoint := fmt.Sprintf("0.0.0.0:%d", port)

	subscribeAPI := &SubscribeAPI{}
	subscribeAPI.events = newEventSystem()
	subscribeAPI.logger = log.GetLoggerByIndex(log.ETHRPCLogConfig, strconv.Itoa(common.InstanceIndex))
	apis := []rpc.API{
		{Namespace: "eth", Version: "1", Service: &eth_rpc.EthAPIService{}, Public: true},
		{Namespace: "eth", Version: "1", Service: subscribeAPI, Public: true},
		{Namespace: "net", Version: "1", Service: &NetAPI{}, Public: true},
		{Namespace: "web3", Version: "1", Service: &Web3API{}, Public: true},
	}

	// Register all the APIs exposed by the services
	handler := rpc.NewServer()
	for _, api := range apis {
		if api.Public {
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
	allowOrigins := []string{"*"}
	go rpc.NewWSServer(allowOrigins, handler).Serve(listener)
	common.DefaultLogger.Infof("JSONRPC ws serving on %s \n", endpoint)
	return nil
}

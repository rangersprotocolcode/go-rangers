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
	"io/fs"
	"net"
	"net/http"

	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"embed"
	"fmt"
	"strings"
	"sync"
)

//go:embed dist
var frontResourceFS embed.FS

//startResourceLoader load front static resources
func startResourceLoader(port uint) error {
	endpoint := fmt.Sprintf("0.0.0.0:%d", port)

	mux := http.NewServeMux()
	fsys, _ := fs.Sub(frontResourceFS, "dist")
	fileServer := http.FileServer(http.FS(fsys))
	mux.Handle("/", fileServer)
	mux.Handle("/index", http.StripPrefix("/index", fileServer))
	mux.Handle("/minerInfo", http.StripPrefix("/minerInfo", fileServer))
	mux.Handle("/blockDetail", http.StripPrefix("/blockDetail", fileServer))
	go http.ListenAndServe(endpoint, mux)

	common.DefaultLogger.Infof("Self resource loader serving on http://%s", endpoint)
	return nil
}

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
	fmt.Println(endpoint)
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return err
	}
	server := &http.Server{Handler: NewSelfServer(privateKey)}
	go server.Serve(listener)
	common.DefaultLogger.Infof("Self Https serving on http://%s", endpoint)

	return nil
}

var GtasAPIImpl *GtasAPI

// StartRPC RPC 功能
func StartRPC(host string, port uint, privateKey string) error {
	var err error
	err = startResourceLoader(port)
	if err != nil {
		return err
	}

	GtasAPIImpl = &GtasAPI{}
	GtasAPIImpl.privateKey = privateKey
	GtasAPIImpl.logger = log.GetLoggerByIndex(log.RPCLogConfig, common.GlobalConf.GetString("instance", "index", ""))

	gxLock = &sync.RWMutex{}
	apis := []rpc.API{
		{Namespace: "Rocket", Version: "1", Service: GtasAPIImpl, Public: true},
		{Namespace: "Rangers", Version: "1", Service: GtasAPIImpl, Public: true},
	}

	for plus := 2000; plus < 2040; plus++ {
		endpoint := fmt.Sprintf("%s:%d", host, port+uint(plus))
		common.DefaultLogger.Infof("RPC http: endpoint:%s", endpoint)
		err = startHTTP(endpoint, apis, []string{}, []string{}, []string{})
		if err == nil {
			common.DefaultLogger.Infof("RPC serving on http://%s", endpoint)
			break
		}
		if strings.Contains(err.Error(), "address already in use") {
			common.DefaultLogger.Infof("address: %s already in use", endpoint)
			continue
		}
		return err
	}

	err = startHttps(port, privateKey)
	return err
}

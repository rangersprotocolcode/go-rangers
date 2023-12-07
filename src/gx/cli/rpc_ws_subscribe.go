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
	"com.tuntun.rangers/node/src/gx/rpc"
	"com.tuntun.rangers/node/src/middleware/log"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/utility"
	"context"
	"math/big"
	"sync"
)

var (
	gasPrice               = big.NewInt(1)
	gasLimit        uint64 = 2000000
	callLock               = sync.Mutex{}
	nonce                  = []byte{1, 2, 3, 4, 5, 6, 7, 8}
	difficulty             = utility.Big(*big.NewInt(32))
	totalDifficulty        = utility.Big(*big.NewInt(180))
)

type SubscribeAPI struct {
	events *EventSystem
	logger log.Logger
}

// Logs creates a subscription that fires for all new log that match the given filter criteria.
func (api *SubscribeAPI) Logs(ctx context.Context, crit types.FilterCriteria) (*rpc.Subscription, error) {
	api.logger.Debugf("ws subscribe logs rcv:%v", crit)
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	var (
		rpcSub      = notifier.CreateSubscription()
		matchedLogs = make(chan []*types.Log)
	)

	logsSub, err := api.events.SubscribeLogs(FilterQuery(crit), matchedLogs)
	if err != nil {
		return nil, err
	}

	go func() {

		for {
			select {
			case logs := <-matchedLogs:
				api.logger.Debugf("got matched logs:%v", logs)
				for _, log := range logs {
					notifier.Notify(rpcSub.ID, &log)
				}
			case <-rpcSub.Err(): // client send an unsubscribe request
				api.logger.Debugf("rpc sub error:%v,id:%v", rpcSub.Err(), rpcSub.ID)
				logsSub.Unsubscribe()
				return
			case <-notifier.Closed(): // connection dropped
				api.logger.Debugf("rpc sub closed. error:%v,id:%v", rpcSub.Err(), rpcSub.ID)
				logsSub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}

// NewHeads send a notification each time a new (header) block is appended to the chain.
func (api *SubscribeAPI) NewHeads(ctx context.Context) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		headers := make(chan *types.BlockHeader)
		headersSub := api.events.SubscribeNewHeads(headers)

		for {
			select {
			case h := <-headers:
				header := adaptRPCBlockHeader(h)
				notifier.Notify(rpcSub.ID, header)
			case <-rpcSub.Err():
				headersSub.Unsubscribe()
				return
			case <-notifier.Closed():
				headersSub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}

// adapt ws net_* api
type NetAPI struct {
}

// Version returns the current ethereum protocol version.
// net_version
func (s *NetAPI) Version() string {
	return common.NetworkId()
}

// Listening Returns true if client is actively listening for network connections.
// net_listening
func (s *NetAPI) Listening() bool {
	return true
}

// adapt ws web3_* api
type Web3API struct {
}

// ClientVersion returns the current client version.
// web3_clientVersion
func (api *Web3API) ClientVersion() string {
	return "Rangers/" + common.Version + "/centos-amd64/go1.17.3"
}

type RPCBlockHeader struct {
	Difficulty       *utility.Big   `json:"difficulty"`
	ExtraData        utility.Bytes  `json:"extraData"`
	GasLimit         utility.Uint64 `json:"gasLimit"`
	GasUsed          utility.Uint64 `json:"gasUsed"`
	Hash             common.Hash    `json:"hash"`
	Bloom            string         `json:"logsBloom"`
	Miner            common.Address `json:"miner"`
	MixHash          string         `json:"mixHash"`
	Nonce            utility.Bytes  `json:"nonce"`
	Number           utility.Uint64 `json:"number"`
	ParentHash       common.Hash    `json:"parentHash"`
	ReceiptsRoot     common.Hash    `json:"receiptsRoot"`
	UncleHash        common.Hash    `json:"sha3Uncles"`
	Size             utility.Uint64 `json:"size"`
	StateRoot        common.Hash    `json:"stateRoot"`
	Timestamp        utility.Uint64 `json:"timestamp"`
	TotalDifficulty  *utility.Big   `json:"totalDifficulty"`
	TransactionsRoot common.Hash    `json:"transactionsRoot"`
	Uncles           []string       `json:"uncles"`
}

func adaptRPCBlockHeader(header *types.BlockHeader) *RPCBlockHeader {
	rpcBlockHeader := RPCBlockHeader{
		Difficulty:   &difficulty,
		ExtraData:    utility.Bytes(nonce[:]),
		GasLimit:     utility.Uint64(gasLimit),
		GasUsed:      utility.Uint64(200000),
		Hash:         header.Hash,
		Bloom:        "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
		Miner:        common.BytesToAddress(header.Castor),
		MixHash:      "0x0000000000000000000000000000000000000000000000000000000000000000",
		Nonce:        utility.Bytes(nonce[:]),
		Number:       utility.Uint64(header.Height),
		ParentHash:   header.PreHash,
		ReceiptsRoot: header.ReceiptTree,
		//uncle has to be this value(rlpHash([]*Header(nil))) for pass go ethereum client verify because tx uncles is empty
		UncleHash:       common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
		Size:            utility.Uint64(1234),
		StateRoot:       header.StateTree,
		Timestamp:       utility.Uint64(header.CurTime.Unix()),
		TotalDifficulty: &totalDifficulty,
		Uncles:          []string{},
	}
	if len(header.Transactions) == 0 {
		//transactionsRoot  has to be this value(EmptyRootHash) for pass go ethereum client verify because tx uncles is empty
		rpcBlockHeader.TransactionsRoot = common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	} else {
		rpcBlockHeader.TransactionsRoot = header.TxTree
	}
	return &rpcBlockHeader
}

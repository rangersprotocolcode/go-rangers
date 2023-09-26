package cli

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/core"
	"com.tuntun.rocket/node/src/gx/rpc"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"context"
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
				header := core.ConvertBlockByHeader(h)
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

//adapt ws net_* api
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

//adapt ws web3_* api
type Web3API struct {
}

// ClientVersion returns the current client version.
// web3_clientVersion
func (api *Web3API) ClientVersion() string {
	return "Rangers/" + common.Version + "/centos-amd64/go1.17.3"
}

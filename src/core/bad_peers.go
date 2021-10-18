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

package core

import (
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/utility"
	"sync"
	"time"
)

const badPeersCleanInterval = time.Second * 30

var PeerManager *peerManager

type peerManager struct {
	badPeers map[string]time.Time
	cleaner  *time.Ticker

	lock   sync.RWMutex
	logger log.Logger
}

func initPeerManager(logger log.Logger) {
	badPeerMeter := peerManager{badPeers: make(map[string]time.Time), cleaner: time.NewTicker(badPeersCleanInterval), lock: sync.RWMutex{}, logger: logger}
	go badPeerMeter.loop()
	PeerManager = &badPeerMeter
}

func (manager *peerManager) markEvil(id string) {
	if id == "" {
		return
	}
	manager.lock.Lock()
	defer manager.lock.Unlock()
	_, exit := manager.badPeers[id]
	if exit {
		return
	}

	manager.badPeers[id] = utility.GetTime()
	manager.logger.Debugf("[PeerManager]Mark evil:%s", id)
}

func (manager *peerManager) isEvil(id string) bool {
	if id == "" {
		return false
	}
	manager.lock.RLock()
	defer manager.lock.RUnlock()
	_, exit := manager.badPeers[id]
	return exit
}

func (manager *peerManager) loop() {
	for {
		select {
		case <-manager.cleaner.C:
			manager.lock.Lock()
			manager.logger.Debugf("Bad peers cleaner time up!")
			cleanIds := make([]string, 0, len(manager.badPeers))
			for id, markTime := range manager.badPeers {
				if utility.GetTime().Sub(markTime) >= badPeersCleanInterval {
					cleanIds = append(cleanIds, id)
				}
			}
			for _, id := range cleanIds {
				delete(manager.badPeers, id)
				manager.logger.Debugf("[PeerManager]Release id:%s", id)
			}
			manager.lock.Unlock()
		}
	}
}

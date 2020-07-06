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
	"com.tuntun.rocket/node/src/utility"
	"sync"
	"time"
)

const (
	badPeersCleanInterval = time.Minute * 3
	evilMaxCount          = 3
)

var PeerManager *peerManager

type peerManager struct {
	badPeerMeter map[string]uint64
	badPeers     map[string]time.Time
	cleaner      *time.Ticker

	lock sync.RWMutex
}

func initPeerManager() {
	badPeerMeter := peerManager{badPeerMeter: make(map[string]uint64), badPeers: make(map[string]time.Time), cleaner: time.NewTicker(badPeersCleanInterval), lock: sync.RWMutex{}}
	go badPeerMeter.loop()
	PeerManager = &badPeerMeter
}

func (bpm *peerManager) markEvil(id string) {
	if id == "" {
		return
	}
	bpm.lock.Lock()
	defer bpm.lock.Unlock()
	_, exit := bpm.badPeers[id]
	if exit {
		return
	}

	evilCount, meterExit := bpm.badPeerMeter[id]
	if !meterExit {
		bpm.badPeerMeter[id] = 1
		return
	} else {
		evilCount++
		if evilCount > evilMaxCount {
			delete(bpm.badPeerMeter, id)
			bpm.badPeers[id] = utility.GetTime()
			logger.Debugf("[PeerManager]Add bad peer:%s", id)
		} else {
			bpm.badPeerMeter[id] = evilCount
			logger.Debugf("[PeerManager]EvilCount:%s,%d", id, evilCount)
		}
	}
}

func (bpm *peerManager) isEvil(id string) bool {
	if id == "" {
		return false
	}
	bpm.lock.RLock()
	defer bpm.lock.RUnlock()
	_, exit := bpm.badPeers[id]
	return exit
}

func (bpm *peerManager) loop() {
	for {
		select {
		case <-bpm.cleaner.C:
			bpm.lock.Lock()
			logger.Debugf("[PeerManager]Bad peers cleaner time up!")
			cleanIds := make([]string, 0, len(bpm.badPeers))
			for id, markTime := range bpm.badPeers {
				if utility.GetTime().Sub(markTime) >= badPeersCleanInterval {
					cleanIds = append(cleanIds, id)
				}
			}
			for _, id := range cleanIds {
				delete(bpm.badPeers, id)
				logger.Debugf("[PeerManager]Clean id:%s", id)
			}
			bpm.lock.Unlock()
		}
	}
}

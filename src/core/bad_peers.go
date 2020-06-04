package core

import (
	"sync"
	"time"
	"x/src/utility"
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

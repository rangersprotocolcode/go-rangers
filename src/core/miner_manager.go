package core

import (
	"errors"
	"sync"
	"time"
	"x/src/utility"

	"x/src/common"

	"x/src/middleware/types"
	"github.com/vmihailenco/msgpack"
	"x/src/storage/account"
	"x/src/consensus/groupsig"
	"x/src/storage/trie"
	"github.com/hashicorp/golang-lru"
	"x/src/service"
)

const (
	heavyMinerNetTriggerInterval = time.Second * 10
	heavyMinerCountKey           = "heavy_miner_count"
	lightMinerCountKey           = "light_miner_count"
)

var (
	emptyValue         [0]byte
	minerCountIncrease = MinerCountOperation{0}
	minerCountDecrease = MinerCountOperation{1}
)

var MinerManagerImpl *MinerManager

type MinerManager struct {
	hasNewHeavyMiner     bool
	heavyMiners          []string
	heavyMinerNetTrigger *time.Timer

	lock sync.RWMutex
}

type MinerCountOperation struct {
	Code int
}

func initMinerManager() {
	MinerManagerImpl = &MinerManager{hasNewHeavyMiner: true, heavyMinerNetTrigger: time.NewTimer(heavyMinerNetTriggerInterval), heavyMiners: make([]string, 0)}
	MinerManagerImpl.lock = sync.RWMutex{}
	//go MinerManagerImpl.loop()
}

func (mm *MinerManager) GetMinerById(id []byte, ttype byte, accountdb *account.AccountDB) *types.Miner {
	if accountdb == nil {
		accountdb = service.AccountDBManagerInstance.GetLatestStateDB()
	}
	db := mm.getMinerDatabase(ttype)
	data := accountdb.GetData(db, id)
	if data != nil && len(data) > 0 {
		var miner types.Miner
		msgpack.Unmarshal(data, &miner)
		return &miner
	}
	return nil
}

func (mm *MinerManager) GetTotalStake(height uint64) uint64 {
	accountDB, err := blockChainImpl.getAccountDBByHeight(height)
	if err != nil {
		logger.Errorf("Get account db by height %d error:%s", height, err.Error())
		return 0
	}

	iter := mm.minerIterator(types.MinerTypeHeavy, accountDB)
	var total uint64 = 0
	for iter.Next() {
		miner, _ := iter.Current()
		if height >= miner.ApplyHeight {
			if miner.Status == types.MinerStatusNormal || height < miner.AbortHeight {
				total += miner.Stake
			}
		}
	}
	if total == 0 {
		iter = mm.minerIterator(types.MinerTypeHeavy, accountDB)
		for ; iter.Next(); {
			miner, _ := iter.Current()
			logger.Debugf("GetTotalStakeByHeight %+v", miner)
		}
	}
	return total
}

func (mm *MinerManager) GetHeavyMiners() []string {
	return mm.heavyMiners
}

func (mm *MinerManager) MinerIterator(minerType byte, height uint64) *MinerIterator {
	accountDB, err := blockChainImpl.getAccountDBByHeight(height)
	if err != nil {
		logger.Error("Get account db by height %d error:%s", height, err.Error())
		return nil
	}
	return mm.minerIterator(minerType, accountDB)
}

func (mm *MinerManager) HeavyMinerCount(height uint64) uint64 {
	accountDB, err := blockChainImpl.getAccountDBByHeight(height)
	if err != nil {
		logger.Error("Get account db by height %d error:%s", height, err.Error())
		return 0
	}
	heavyMinerCountByte := accountDB.GetData(common.MinerCountDBAddress, []byte(heavyMinerCountKey))
	return utility.ByteToUInt64(heavyMinerCountByte)

}

func (mm *MinerManager) LightMinerCount(height uint64) uint64 {
	accountDB, err := blockChainImpl.getAccountDBByHeight(height)
	if err != nil {
		logger.Error("Get account db by height %d error:%s", height, err.Error())
		return 0
	}
	lightMinerCountByte := accountDB.GetData(common.MinerCountDBAddress, []byte(lightMinerCountKey))
	return utility.ByteToUInt64(lightMinerCountByte)
}

//func (mm *MinerManager) loop() {
//	for {
//		<-mm.heavyMinerNetTrigger.C
//		if mm.hasNewHeavyMiner {
//			iterator := mm.minerIterator(types.MinerTypeHeavy, nil)
//			array := make([]string, 0)
//			for iterator.Next() {
//				miner, _ := iterator.Current()
//				gid := groupsig.DeserializeId(miner.Id)
//				array = append(array, gid.String())
//			}
//			mm.heavyMiners = array
//			network.GetNetInstance().BuildGroupNet(network.FullNodeVirtualGroupId, array)
//			logger.Infof("MinerManager HeavyMinerUpdate Size:%d", len(array))
//			mm.hasNewHeavyMiner = false
//		}
//		mm.heavyMinerNetTrigger.Reset(heavyMinerNetTriggerInterval)
//	}
//}

func (mm *MinerManager) getMinerDatabase(minerType byte) (common.Address) {
	switch minerType {
	case types.MinerTypeLight:
		return common.LightDBAddress
	case types.MinerTypeHeavy:
		return common.HeavyDBAddress
	}
	return common.Address{}
}

func (mm *MinerManager) addMiner(id []byte, miner *types.Miner, accountdb *account.AccountDB) int {
	logger.Debugf("Miner manager add miner %d", miner.Type)
	db := mm.getMinerDatabase(miner.Type)

	if accountdb.GetData(db, id) != nil {
		return -1
	} else {
		data, _ := msgpack.Marshal(miner)
		accountdb.SetData(db, id, data)
		if miner.Type == types.MinerTypeHeavy {
			mm.hasNewHeavyMiner = true
		}
		mm.updateMinerCount(miner.Type, minerCountIncrease, accountdb)
		return 1
	}
}

func (mm *MinerManager) addGenesesVerifier(miners []*types.Miner, accountdb *account.AccountDB) {
	dbl := mm.getMinerDatabase(types.MinerTypeLight)

	for _, miner := range miners {
		if accountdb.GetData(dbl, miner.Id) == nil {
			miner.Type = types.MinerTypeLight
			data, _ := msgpack.Marshal(miner)
			accountdb.SetData(dbl, miner.Id, data)
			mm.updateMinerCount(types.MinerTypeLight, minerCountIncrease, accountdb)
		}
	}
	mm.hasNewHeavyMiner = true
}

func (mm *MinerManager) addGenesesProposer(miners []*types.Miner, accountdb *account.AccountDB) {
	dbh := mm.getMinerDatabase(types.MinerTypeHeavy)

	for _, miner := range miners {
		if accountdb.GetData(dbh, miner.Id) == nil {
			miner.Type = types.MinerTypeHeavy
			data, _ := msgpack.Marshal(miner)
			logger.Debugf("Miner manager add genesis miner %v", miner.Id)
			accountdb.SetData(dbh, miner.Id, data)
			mm.heavyMiners = append(mm.heavyMiners, groupsig.DeserializeID(miner.Id).GetHexString())
			mm.updateMinerCount(types.MinerTypeHeavy, minerCountIncrease, accountdb)
		}
	}
	mm.hasNewHeavyMiner = true
}

func (mm *MinerManager) removeMiner(id []byte, ttype byte, accountdb *account.AccountDB) {
	logger.Debugf("Miner manager remove miner %d", ttype)
	db := mm.getMinerDatabase(ttype)
	accountdb.SetData(db, id, emptyValue[:])
}

func (mm *MinerManager) abortMiner(id []byte, ttype byte, height uint64, accountdb *account.AccountDB) bool {
	miner := mm.GetMinerById(id, ttype, accountdb)
	if miner != nil && miner.Status == types.MinerStatusNormal {
		miner.Status = types.MinerStatusAbort
		miner.AbortHeight = height

		db := mm.getMinerDatabase(ttype)
		data, _ := msgpack.Marshal(miner)
		accountdb.SetData(db, id, data)
		mm.updateMinerCount(ttype, minerCountDecrease, accountdb)
		logger.Debugf("Miner manager abort miner update success %+v", miner)
		return true
	} else {
		logger.Debugf("Miner manager abort miner update fail %+v", miner)
		return false
	}
}

func (mm *MinerManager) minerIterator(minerType byte, accountdb *account.AccountDB) *MinerIterator {
	db := mm.getMinerDatabase(minerType)
	if accountdb == nil {
		accountdb = service.AccountDBManagerInstance.GetLatestStateDB()
	}
	iterator := &MinerIterator{iterator: accountdb.DataIterator(db, []byte(""))}
	return iterator
}

func (mm *MinerManager) updateMinerCount(minerType byte, operation MinerCountOperation, accountdb *account.AccountDB) {
	if minerType == types.MinerTypeHeavy {
		heavyMinerCountByte := accountdb.GetData(common.MinerCountDBAddress, []byte(heavyMinerCountKey))
		heavyMinerCount := utility.ByteToUInt64(heavyMinerCountByte)
		if operation == minerCountIncrease {
			heavyMinerCount++
		} else if operation == minerCountDecrease {
			heavyMinerCount--
		}
		accountdb.SetData(common.MinerCountDBAddress, []byte(heavyMinerCountKey), utility.UInt64ToByte(heavyMinerCount))
		return
	}

	if minerType == types.MinerTypeLight {
		lightMinerCountByte := accountdb.GetData(common.MinerCountDBAddress, []byte(lightMinerCountKey))
		lightMinerCount := utility.ByteToUInt64(lightMinerCountByte)
		if operation == minerCountIncrease {
			lightMinerCount++
		} else if operation == minerCountDecrease {
			lightMinerCount--
		}
		accountdb.SetData(common.MinerCountDBAddress, []byte(lightMinerCountKey), utility.UInt64ToByte(lightMinerCount))
		return
	}
	logger.Error("Unknown miner type:%d", minerType)
}

type MinerIterator struct {
	iterator *trie.Iterator
	cache    *lru.Cache
}

func (mi *MinerIterator) Current() (*types.Miner, error) {
	if mi.cache != nil {
		if result, ok := mi.cache.Get(string(mi.iterator.Key)); ok {
			return result.(*types.Miner), nil
		}
	}
	var miner types.Miner
	err := msgpack.Unmarshal(mi.iterator.Value, &miner)
	if err != nil {
		logger.Debugf("MinerIterator Unmarshal Error %+v %+v %+v", mi.iterator.Key, err, mi.iterator.Value)
	}

	if len(miner.Id) == 0 {
		err = errors.New("empty miner")
	}
	return &miner, err
}

func (mi *MinerIterator) Next() bool {
	return mi.iterator.Next()
}

package core

import (
	"errors"
	"sync"
	"time"
	"x/src/common"

	"x/src/middleware/types"
	"github.com/vmihailenco/msgpack"
	"x/src/storage/account"
	"x/src/storage/trie"
	"x/src/service"
)

const (
	heavyMinerNetTriggerInterval = time.Second * 10
)

var (
	emptyValue         [0]byte
	minerCountIncrease = MinerCountOperation{0}
	minerCountDecrease = MinerCountOperation{1}
)

var MinerManagerImpl *MinerManager

type MinerManager struct {
	heavyMinerNetTrigger *time.Timer

	lock sync.RWMutex
}

type MinerCountOperation struct {
	Code int
}

func initMinerManager() {
	MinerManagerImpl = &MinerManager{heavyMinerNetTrigger: time.NewTimer(heavyMinerNetTriggerInterval)}
	MinerManagerImpl.lock = sync.RWMutex{}
	//go MinerManagerImpl.loop()
}

func (mm *MinerManager) GetMinerById(id []byte, kind byte, accountdb *account.AccountDB) *types.Miner {
	if accountdb == nil {
		accountdb = service.AccountDBManagerInstance.GetLatestStateDB()
	}

	db := mm.getMinerDatabase(kind)
	data := accountdb.GetData(db, id)
	if data != nil && len(data) > 0 {
		var miner types.Miner
		msgpack.Unmarshal(data, &miner)
		return &miner
	}
	return nil
}

func (mm *MinerManager) GetValidatorsStake(height uint64, members [][]byte) (total uint64, membersDetail map[common.Address]uint64) {
	accountDB, err := blockChainImpl.getAccountDBByHeight(height)
	if err != nil {
		logger.Errorf("Get account db by height %d error:%s", height, err.Error())
		return 0, nil
	}

	total = 0
	membersDetail = make(map[common.Address]uint64, len(members))
	for _, member := range members {
		id := getAddressFromID(member)
		miner := mm.GetMinerById(member, common.MinerTypeValidator, accountDB)
		if nil == miner {
			logger.Errorf("fail to get Member,id: %s", id)
			continue
		}
		membersDetail[id] = miner.Stake
		total += miner.Stake
	}

	return total, membersDetail
}

func (mm *MinerManager) GetProposerTotalStakeWithAccountDB(height uint64, accountDB *account.AccountDB) (total uint64, membersDetail map[common.Address]uint64) {
	if accountDB == nil {
		return 0, nil
	}

	total = 0
	membersDetail = make(map[common.Address]uint64)

	iter := mm.minerIterator(common.MinerTypeProposer, accountDB)
	for iter.Next() {
		miner, _ := iter.Current()
		if height >= miner.ApplyHeight && (miner.Status == common.MinerStatusNormal || height < miner.AbortHeight) {
			total += miner.Stake
			membersDetail[getAddressFromID(miner.Id)] = miner.Stake
		}
	}

	if total == 0 {
		iter = mm.minerIterator(common.MinerTypeProposer, accountDB)
		for iter.Next() {
			miner, _ := iter.Current()
			logger.Debugf("GetTotalStakeByHeight %+v", miner)
		}
	}

	return
}

func (mm *MinerManager) GetProposerTotalStake(height uint64) (total uint64, membersDetail map[common.Address]uint64) {
	accountDB, err := blockChainImpl.getAccountDBByHeight(height)
	if err != nil {
		logger.Errorf("Get account db by height %d error:%s", height, err.Error())
		return 0, nil
	}

	return mm.GetProposerTotalStakeWithAccountDB(height, accountDB)
}

func (mm *MinerManager) MinerIterator(minerType byte, height uint64) *MinerIterator {
	accountDB, err := blockChainImpl.getAccountDBByHeight(height)
	if err != nil {
		logger.Error("Get account db by height %d error:%s", height, err.Error())
		return nil
	}
	return mm.minerIterator(minerType, accountDB)
}

//func (mm *MinerManager) HeavyMinerCount(height uint64) uint64 {
//	accountDB, err := blockChainImpl.getAccountDBByHeight(height)
//	if err != nil {
//		logger.Error("Get account db by height %d error:%s", height, err.Error())
//		return 0
//	}
//	heavyMinerCountByte := accountDB.GetData(common.MinerCountDBAddress, []byte(proposerCountKey))
//	return utility.ByteToUInt64(heavyMinerCountByte)
//
//}
//
//func (mm *MinerManager) LightMinerCount(height uint64) uint64 {
//	accountDB, err := blockChainImpl.getAccountDBByHeight(height)
//	if err != nil {
//		logger.Error("Get account db by height %d error:%s", height, err.Error())
//		return 0
//	}
//	lightMinerCountByte := accountDB.GetData(common.MinerCountDBAddress, []byte(validatorCountKey))
//	return utility.ByteToUInt64(lightMinerCountByte)
//}

//func (mm *MinerManager) loop() {
//	for {
//		<-mm.heavyMinerNetTrigger.C
//		if mm.hasNewHeavyMiner {
//			iterator := mm.minerIterator(common.MinerTypeProposer, nil)
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

func (mm *MinerManager) getMinerDatabase(minerType byte) common.Address {
	switch minerType {
	case common.MinerTypeValidator:
		return common.ValidatorDBAddress
	case common.MinerTypeProposer:
		return common.ProposerDBAddress
	}
	return common.Address{}
}

func (mm *MinerManager) AddMiner(miner *types.Miner, accountdb *account.AccountDB) int {
	logger.Debugf("Miner manager add miner, %v", miner)

	id := miner.Id
	db := mm.getMinerDatabase(miner.Type)

	if accountdb.GetData(db, id) != nil {
		return -1
	} else {
		data, _ := msgpack.Marshal(miner)
		accountdb.SetData(db, id, data)
		//if miner.Type == common.MinerTypeProposer {
		//	mm.hasNewHeavyMiner = true
		//}
		//mm.updateMinerCount(miner.Type, minerCountIncrease, accountdb)
		return 1
	}
}

func (mm *MinerManager) removeMiner(id []byte, ttype byte, accountdb *account.AccountDB) {
	logger.Debugf("Miner manager remove miner %d", ttype)
	db := mm.getMinerDatabase(ttype)
	accountdb.SetData(db, id, emptyValue[:])
}

func (mm *MinerManager) abortMiner(id []byte, ttype byte, height uint64, accountdb *account.AccountDB) bool {
	miner := mm.GetMinerById(id, ttype, accountdb)
	if miner != nil && miner.Status == common.MinerStatusNormal {
		miner.Status = common.MinerStatusAbort
		miner.AbortHeight = height

		db := mm.getMinerDatabase(ttype)
		data, _ := msgpack.Marshal(miner)
		accountdb.SetData(db, id, data)
		//mm.updateMinerCount(ttype, minerCountDecrease, accountdb)
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

//func (mm *MinerManager) updateMinerCount(minerType byte, operation MinerCountOperation, accountdb *account.AccountDB) {
//	if minerType == common.MinerTypeProposer {
//		heavyMinerCountByte := accountdb.GetData(common.MinerCountDBAddress, []byte(proposerCountKey))
//		heavyMinerCount := utility.ByteToUInt64(heavyMinerCountByte)
//		if operation == minerCountIncrease {
//			heavyMinerCount++
//		} else if operation == minerCountDecrease {
//			heavyMinerCount--
//		}
//		accountdb.SetData(common.MinerCountDBAddress, []byte(proposerCountKey), utility.UInt64ToByte(heavyMinerCount))
//		return
//	}
//
//	if minerType == types.MinerTypeLight {
//		lightMinerCountByte := accountdb.GetData(common.MinerCountDBAddress, []byte(validatorCountKey))
//		lightMinerCount := utility.ByteToUInt64(lightMinerCountByte)
//		if operation == minerCountIncrease {
//			lightMinerCount++
//		} else if operation == minerCountDecrease {
//			lightMinerCount--
//		}
//		accountdb.SetData(common.MinerCountDBAddress, []byte(validatorCountKey), utility.UInt64ToByte(lightMinerCount))
//		return
//	}
//
//	logger.Error("Unknown miner type:%d", minerType)
//}

type MinerIterator struct {
	iterator *trie.Iterator
}

func (mi *MinerIterator) Current() (*types.Miner, error) {
	var miner types.Miner
	err := msgpack.Unmarshal(mi.iterator.Value, &miner)
	if err != nil {
		logger.Debugf("MinerIterator Unmarshal Error %+v %+v %+v", mi.iterator.Key, err, mi.iterator.Value)
	}

	if len(miner.Id) == 0 {
		err = errors.New("empty miner")
	}

	if miner.Status == common.MinerStatusAbort {
		err = errors.New("abort miner")
	}

	if miner.Stake-miner.Used < common.ValidatorStake {
		err = errors.New("not enough stake")
	}

	return &miner, err
}

func (mi *MinerIterator) Next() bool {
	return mi.iterator.Next()
}

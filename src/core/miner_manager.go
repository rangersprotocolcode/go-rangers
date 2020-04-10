package core

import (
	"errors"
	"fmt"
	"sync"
	"time"
	"x/src/common"

	"github.com/vmihailenco/msgpack"
	"x/src/middleware/types"
	"x/src/service"
	"x/src/storage/account"
	"x/src/storage/trie"
	"x/src/utility"
)

const (
	heavyMinerNetTriggerInterval = time.Second * 10
)

var (
	emptyValue       [0]byte
	MinerManagerImpl *MinerManager
)

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
}

func (mm *MinerManager) GetMiner(minerId []byte, accountdb *account.AccountDB) *types.Miner {
	miner := MinerManagerImpl.GetMinerById(minerId, common.MinerTypeProposer, accountdb)
	if nil == miner {
		miner = MinerManagerImpl.GetMinerById(minerId, common.MinerTypeValidator, accountdb)
	}

	return miner
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

func (mm *MinerManager) GetValidatorsStake(members [][]byte, accountDB *account.AccountDB) (uint64, map[common.Address]uint64) {
	total := uint64(0)
	membersDetail := make(map[common.Address]uint64, len(members))
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

func (mm *MinerManager) GetProposerTotalStakeWithDetail(height uint64, accountDB *account.AccountDB) (uint64, map[common.Address]uint64) {
	if accountDB == nil {
		return 0, nil
	}

	total := uint64(0)
	membersDetail := make(map[common.Address]uint64)

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

	return total, membersDetail
}

func (mm *MinerManager) GetProposerTotalStake(height uint64) uint64 {
	accountDB, err := blockChainImpl.getAccountDBByHeight(height)
	if err != nil {
		logger.Errorf("Get account db by height %d error:%s", height, err.Error())
		return 0
	}

	total, _ := mm.GetProposerTotalStakeWithDetail(height, accountDB)

	return total
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

func (mm *MinerManager) AddStake(addr common.Address, minerId []byte, delta uint64, accountdb *account.AccountDB) (bool, string) {
	if delta == 0 {
		return true, ""
	}

	stake := utility.Float64ToBigInt(float64(delta))
	balance := accountdb.GetBalance(addr)
	if balance.Cmp(stake) < 0 {
		msg := fmt.Sprintf("not enough balance, addr: %s, balance: %d, stake: %d", addr.String(), balance, stake)
		logger.Errorf(msg)
		return false, msg
	}

	miner := mm.GetMinerById(minerId, common.MinerTypeProposer, accountdb)
	if nil == miner {
		miner = mm.GetMinerById(minerId, common.MinerTypeValidator, accountdb)
		if nil == miner {
			return false, "miner is not existed"
		}
	}

	miner.Stake = miner.Stake + delta
	if miner.Stake < 0 {
		return false, "overflow"
	}

	accountdb.SubBalance(addr, stake)
	mm.UpdateMiner(miner, accountdb)
	return true, ""
}

func (mm *MinerManager) AddMiner(addr common.Address, miner *types.Miner, accountdb *account.AccountDB) (bool, string) {
	if miner.Type != common.MinerTypeValidator && miner.Type != common.MinerTypeProposer {
		msg := fmt.Sprintf("miner type error, minerId: %s, type: %d", common.ToHex(miner.Id), miner.Type)
		logger.Errorf(msg)
		return false, msg
	}
	if (miner.Type == common.MinerTypeValidator && miner.Stake < common.ValidatorStake) ||
		(miner.Type == common.MinerTypeProposer && miner.Stake < common.ProposerStake) {
		msg := fmt.Sprintf("not enough stake, minerId: %s, stake: %d", common.ToHex(miner.Id), miner.Stake)
		logger.Errorf(msg)
		return false, msg
	}
	if isEmptyByteSlice(miner.VrfPublicKey) || isEmptyByteSlice(miner.PublicKey) {
		msg := fmt.Sprintf("VrfPublicKey or PublicKey is empty, minerId: %s, vrfPublicKey: %v,publicKey: %v", common.ToHex(miner.Id), miner.VrfPublicKey, miner.PublicKey)
		logger.Errorf(msg)
		return false, msg

	}

	stake := utility.Float64ToBigInt(float64(miner.Stake))
	balance := accountdb.GetBalance(addr)
	if balance.Cmp(stake) < 0 {
		msg := fmt.Sprintf("not enough max, addr: %s, balance: %d, stake: %d", addr.String(), balance, stake)
		logger.Errorf(msg)
		return false, msg
	}

	id := miner.Id
	if mm.GetMiner(id, accountdb) != nil {
		msg := fmt.Sprintf("miner is existed. minerId: %s", common.ToHex(id))
		logger.Errorf(msg)
		return false, msg
	}

	accountdb.SubBalance(addr, stake)
	mm.UpdateMiner(miner, accountdb)
	logger.Debugf("add miner: %v", miner)
	return true, ""
}

func (mm *MinerManager) UpdateMiner(miner *types.Miner, accountdb *account.AccountDB) {
	id := miner.Id
	db := mm.getMinerDatabase(miner.Type)
	data, _ := msgpack.Marshal(miner)

	accountdb.SetData(db, id, data)
}

func (mm *MinerManager) addMiner(miner *types.Miner, accountdb *account.AccountDB) int {
	logger.Debugf("Miner manager add miner, %v", miner)

	id := miner.Id
	db := mm.getMinerDatabase(miner.Type)

	if accountdb.GetData(db, id) != nil {
		return -1
	} else {
		mm.UpdateMiner(miner, accountdb)
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
		mm.UpdateMiner(miner, accountdb)
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

	return &miner, err
}

func (mi *MinerIterator) Next() bool {
	return mi.iterator.Next()
}

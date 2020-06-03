package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"x/src/common"
	"x/src/middleware/db"

	"x/src/middleware/types"
	"x/src/service"
	"x/src/storage/account"
	"x/src/storage/trie"
	"x/src/utility"
)

var (
	emptyValue       [0]byte
	MinerManagerImpl *MinerManager
)

type MinerManager struct {
	pkCache *db.LDBDatabase
}

func initMinerManager() {
	pkp, err := db.NewLDBDatabase("pkp"+strconv.Itoa(common.InstanceIndex), 1, 1)
	if err != nil {
		panic("newLDBDatabase fail, file=" + "" + "err=" + err.Error())
	}

	MinerManagerImpl = &MinerManager{pkCache: pkp}
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

	db := mm.getMinerDatabaseAddress(kind)
	data := accountdb.GetData(db, id)
	if data != nil && len(data) > 0 {
		var miner types.Miner
		json.Unmarshal(data, &miner)
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

	_, proposers := mm.GetProposerTotalStakeWithDetail(height, accountDB)

	return uint64(len(proposers))
}

func (mm *MinerManager) MinerIterator(minerType byte, height uint64) *MinerIterator {
	accountDB, err := blockChainImpl.getAccountDBByHeight(height)
	if err != nil {
		logger.Error("Get account db by height %d error:%s", height, err.Error())
		return nil
	}
	return mm.minerIterator(minerType, accountDB)
}

func (mm *MinerManager) getMinerDatabaseAddress(minerType byte) common.Address {
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

	mm.pkCache.Put(miner.Id, miner.PublicKey)
	return true, ""
}

func (mm *MinerManager) GetPubkey(id []byte) ([]byte, error) {
	return mm.pkCache.Get(id)
}

func (mm *MinerManager) UpdateMiner(miner *types.Miner, accountdb *account.AccountDB) {
	id := miner.Id
	db := mm.getMinerDatabaseAddress(miner.Type)
	data, _ := json.Marshal(miner)

	accountdb.SetData(db, id, data)
}

func (mm *MinerManager) addMiner(miner *types.Miner, accountdb *account.AccountDB) int {
	logger.Debugf("Miner manager add miner, %v", miner)

	id := miner.Id
	db := mm.getMinerDatabaseAddress(miner.Type)

	if accountdb.GetData(db, id) != nil {
		return -1
	} else {
		mm.pkCache.Put(miner.Id, miner.PublicKey)
		mm.UpdateMiner(miner, accountdb)
		return 1
	}
}

func (mm *MinerManager) removeMiner(id []byte, ttype byte, accountdb *account.AccountDB) {
	logger.Debugf("Miner manager remove miner %d", ttype)
	db := mm.getMinerDatabaseAddress(ttype)
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
	db := mm.getMinerDatabaseAddress(minerType)
	if accountdb == nil {
		accountdb = service.AccountDBManagerInstance.GetLatestStateDB()
	}
	iterator := &MinerIterator{iterator: accountdb.DataIterator(db, []byte(""))}
	return iterator
}

type MinerIterator struct {
	iterator *trie.Iterator
}

func (mi *MinerIterator) Current() (*types.Miner, error) {
	var miner types.Miner
	err := json.Unmarshal(mi.iterator.Value, &miner)
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

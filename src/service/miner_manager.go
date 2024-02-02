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

package service

import (
	"bytes"
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware"
	"com.tuntun.rangers/node/src/middleware/db"
	"com.tuntun.rangers/node/src/middleware/log"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/storage/account"
	"com.tuntun.rangers/node/src/storage/trie"
	"com.tuntun.rangers/node/src/utility"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

var (
	emptyValue       [0]byte
	MinerManagerImpl *MinerManager
)

type MinerManager struct {
	pkCache *db.LDBDatabase
	logger  log.Logger
}

func InitMinerManager() {
	pkp, err := db.NewLDBDatabase("pkp", 1, 1)
	if err != nil {
		panic("newLDBDatabase pkp fail, err=" + err.Error())
	}

	MinerManagerImpl = &MinerManager{pkCache: pkp}
	MinerManagerImpl.logger = log.GetLoggerByIndex(log.TxLogConfig, strconv.Itoa(common.InstanceIndex))
}

func (mm *MinerManager) GetAllMinerIdAndAccount(height uint64, accountDB *account.AccountDB) (map[string]common.Address, map[string]common.Address) {
	if accountDB == nil {
		return nil, nil
	}

	proposals, validators := make(map[string]common.Address, 0), make(map[string]common.Address, 0)

	iter := mm.minerIterator(common.MinerTypeProposer, accountDB)
	for iter.Next() {
		miner, _ := iter.Current()
		if nil == miner || common.MinerStatusNormal != miner.Status || height < miner.ApplyHeight {
			continue
		}
		proposals[common.ToHex(miner.Id)] = common.BytesToAddress(miner.Account)
	}

	iter = mm.minerIterator(common.MinerTypeValidator, accountDB)
	for iter.Next() {
		miner, _ := iter.Current()
		if nil == miner || common.MinerStatusNormal != miner.Status || height < miner.ApplyHeight {
			continue
		}
		validators[common.ToHex(miner.Id)] = common.BytesToAddress(miner.Account)
	}
	return proposals, validators
}

func (mm *MinerManager) GetMinerIdByAccount(account []byte, accountDB *account.AccountDB) types.HexBytes {
	iterator := mm.minerIterator(common.MinerTypeValidator, accountDB)
	for iterator.Next() {
		curr, _ := iterator.Current()
		if nil == curr {
			continue
		}

		if 0 == bytes.Compare(curr.Account, account) {
			return curr.Id
		}
	}

	iterator = mm.minerIterator(common.MinerTypeProposer, accountDB)
	for iterator.Next() {
		curr, _ := iterator.Current()
		if nil == curr {
			continue
		}

		if 0 == bytes.Compare(curr.Account, account) {
			return curr.Id
		}
	}

	return nil

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
		accountdb = middleware.AccountDBManagerInstance.GetLatestStateDB()
	}

	db := mm.getMinerDatabaseAddress(kind)
	data := accountdb.GetData(db, id)
	if data != nil && len(data) > 0 {
		var miner types.Miner
		err := json.Unmarshal(data, &miner)
		if nil != err {
			mm.logger.Errorf("fail to getminer: %s, type: %d", common.ToHex(id), kind)
			return nil
		}

		miner.Stake = mm.getMinerStake(id, kind, accountdb)
		miner.Account = mm.getMinerAccount(id, kind, accountdb)
		status := accountdb.GetData(db, common.Sha256(common.Sha256(common.Sha256(id))))
		if nil != status && 1 == len(status) {
			miner.Status = status[0]
		}
		return &miner
	}

	return nil
}

func (mm *MinerManager) getMinerStake(id []byte, kind byte, accountdb *account.AccountDB) uint64 {
	db := mm.getMinerDatabaseAddress(kind)
	stakeBytes := accountdb.GetData(db, common.Sha256(id))
	return utility.ByteToUInt64(stakeBytes)
}

func (mm *MinerManager) getMinerAccount(id []byte, kind byte, accountdb *account.AccountDB) []byte {
	db := mm.getMinerDatabaseAddress(kind)
	return accountdb.GetData(db, common.Sha256(common.Sha256(id)))
}

func (mm *MinerManager) GetValidatorsStake(members [][]byte, accountDB *account.AccountDB) (uint64, map[common.Address]uint64) {
	total := uint64(0)
	membersDetail := make(map[common.Address]uint64, len(members))
	for _, member := range members {
		stake := mm.getMinerStake(member, common.MinerTypeValidator, accountDB)
		account := mm.getMinerAccount(member, common.MinerTypeValidator, accountDB)
		if 0 == stake {
			mm.logger.Errorf("fail to get Member,id: %s", common.BytesToAddress(member))
			continue
		}
		addr := common.BytesToAddress(account)
		current := membersDetail[addr]
		membersDetail[addr] = stake + current
		total += stake
	}

	return total, membersDetail
}

func (mm *MinerManager) GetProposerTotalStakeWithDetail(height uint64, accountDB *account.AccountDB) (uint64, map[string]uint64) {
	if accountDB == nil {
		return 0, nil
	}

	total := uint64(0)
	membersDetail := make(map[string]uint64)

	iter := mm.minerIterator(common.MinerTypeProposer, accountDB)
	for iter.Next() {
		miner, _ := iter.Current()
		if nil == miner || common.MinerStatusNormal != miner.Status || height < miner.ApplyHeight {
			continue
		}

		total += miner.Stake
		membersDetail[common.ToHex(miner.Id)] = miner.Stake
	}

	if total == 0 {
		iter = mm.minerIterator(common.MinerTypeProposer, accountDB)
		for iter.Next() {
			miner, _ := iter.Current()
			if nil == miner {
				continue
			}
			mm.logger.Debugf("GetTotalStakeByHeight %+v", miner)
		}
	}

	return total, membersDetail
}

func (mm *MinerManager) GetProposerTotalStake(height uint64, hash common.Hash) uint64 {
	accountDB, err := middleware.AccountDBManagerInstance.GetAccountDBByHash(hash)
	if err != nil {
		mm.logger.Errorf("Get account db by height %d error:%s", height, err.Error())
		return 0
	}

	_, proposers := mm.GetProposerTotalStakeWithDetail(height, accountDB)

	return uint64(len(proposers))
}

func (mm *MinerManager) MinerIterator(minerType byte, hash common.Hash) *MinerIterator {
	accountDB, err := middleware.AccountDBManagerInstance.GetAccountDBByHash(hash)
	if err != nil {
		mm.logger.Error("Get account db by hash %s error:%s", hash.Hex(), err.Error())
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
		mm.logger.Errorf(msg)
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
	if miner.Type == common.MinerTypeProposer && miner.Stake > common.ProposerStake ||
		miner.Type == common.MinerTypeValidator && miner.Stake > common.ValidatorStake {
		miner.Status = common.MinerStatusNormal
	}

	accountdb.SubBalance(addr, stake)
	mm.UpdateMiner(miner, accountdb, false)
	return true, ""
}

func (mm *MinerManager) AddMiner(addr common.Address, miner *types.Miner, accountdb *account.AccountDB) (bool, string) {
	if miner.Type != common.MinerTypeValidator && miner.Type != common.MinerTypeProposer {
		msg := fmt.Sprintf("miner type error, minerId: %s, type: %d", common.ToHex(miner.Id), miner.Type)
		mm.logger.Errorf(msg)
		return false, msg
	}

	if (miner.Type == common.MinerTypeValidator && miner.Stake < common.ValidatorStake) ||
		(miner.Type == common.MinerTypeProposer && miner.Stake < common.ProposerStake) {
		msg := fmt.Sprintf("not enough stake, minerId: %s, stake: %d", common.ToHex(miner.Id), miner.Stake)
		mm.logger.Errorf(msg)
		return false, msg
	}

	if utility.IsEmptyByteSlice(miner.VrfPublicKey) || utility.IsEmptyByteSlice(miner.PublicKey) {
		msg := fmt.Sprintf("VrfPublicKey or PublicKey is empty, minerId: %s, vrfPublicKey: %v,publicKey: %v", common.ToHex(miner.Id), miner.VrfPublicKey, miner.PublicKey)
		mm.logger.Errorf(msg)
		return false, msg

	}

	stake := utility.Float64ToBigInt(float64(miner.Stake))
	balance := accountdb.GetBalance(addr)
	if balance.Cmp(stake) < 0 {
		msg := fmt.Sprintf("not enough max, addr: %s, balance: %d, stake: %d", addr.String(), balance, stake)
		mm.logger.Errorf(msg)
		return false, msg
	}

	id := miner.Id
	if mm.GetMiner(id, accountdb) != nil {
		msg := fmt.Sprintf("miner is existed. minerId: %s", common.ToHex(id))
		mm.logger.Errorf(msg)
		return false, msg
	}

	existed := mm.GetMinerIdByAccount(miner.Account, accountdb)
	if nil != existed {
		msg := fmt.Sprintf("miner account is existed. minerId: %s, account: %s", common.ToHex(existed), common.ToHex(miner.Account))
		mm.logger.Errorf(msg)
		return false, msg
	}

	accountdb.SubBalance(addr, stake)
	mm.UpdateMiner(miner, accountdb, true)
	mm.logger.Debugf("add miner: %v", miner)

	mm.pkCache.Put(miner.Id, miner.PublicKey)
	return true, ""
}

func (mm *MinerManager) GetPubkey(id []byte) ([]byte, error) {
	return mm.pkCache.Get(id)
}

func (mm *MinerManager) UpdateMiner(miner *types.Miner, accountdb *account.AccountDB, isNew bool) {
	id := miner.Id
	db := mm.getMinerDatabaseAddress(miner.Type)

	if isNew {
		data := miner.GetMinerInfo()
		mm.logger.Debugf("UpdateMiner, %s", utility.BytesToStr(data))
		accountdb.SetData(db, id, data)
	}

	key := common.Sha256(id)
	accountdb.SetData(db, key, utility.UInt64ToByte(miner.Stake))
	key = common.Sha256(key)
	accountdb.SetData(db, key, miner.Account)

	if common.IsProposal003() {
		key = common.Sha256(key)
		accountdb.SetData(db, key, []byte{miner.Status})
	}

}

func (mm *MinerManager) InsertMiner(miner *types.Miner, accountdb *account.AccountDB) int {
	mm.logger.Debugf("Miner manager add miner, %v", miner)

	id := miner.Id
	db := mm.getMinerDatabaseAddress(miner.Type)

	if accountdb.GetData(db, id) != nil {
		return -1
	} else {
		mm.pkCache.Put(miner.Id, miner.PublicKey)
		mm.UpdateMiner(miner, accountdb, true)
		return 1
	}
}

func (mm *MinerManager) RemoveUnusedValidator(accountDB *account.AccountDB, whitelist map[string]byte) {
	unusedList := make([]*types.Miner, 0)
	iter := mm.minerIterator(common.MinerTypeValidator, accountDB)
	for iter.Next() {
		miner, _ := iter.Current()
		if nil == miner || common.MinerStatusNormal != miner.Status {
			continue
		}

		id := common.ToHex(miner.Id)
		_, ok := whitelist[id]
		if ok {
			continue
		}

		unusedList = append(unusedList, miner)
		mm.logger.Debugf("add unused, id: %s", id)
	}

	for _, unused := range unusedList {
		if nil == unused {
			continue
		}
		mm.RemoveMiner(unused.Id, unused.Account, common.MinerTypeValidator, accountDB, 0)
	}
}

func (mm *MinerManager) RemoveMiner(id, account []byte, ttype byte, accountdb *account.AccountDB, left uint64) {
	mm.logger.Debugf("Miner manager remove miner %d", ttype)

	db := mm.getMinerDatabaseAddress(ttype)

	if left == 0 && !accountdb.IsContract(common.BytesToAddress(account)) {
		accountdb.SetData(db, id, emptyValue[:])
		key := common.Sha256(id)
		accountdb.SetData(db, key, emptyValue[:])
		key = common.Sha256(key)
		accountdb.SetData(db, key, emptyValue[:])
		key = common.Sha256(key)
		accountdb.SetData(db, key, emptyValue[:])
		return
	}

	// stake
	key := common.Sha256(id)
	accountdb.SetData(db, key, utility.UInt64ToByte(left))

	// status
	key = common.Sha256(common.Sha256(key))
	accountdb.SetData(db, key, []byte{common.MinerStatusAbort})
}

func (mm *MinerManager) minerIterator(minerType byte, accountdb *account.AccountDB) *MinerIterator {
	db := mm.getMinerDatabaseAddress(minerType)
	if accountdb == nil {
		accountdb = middleware.AccountDBManagerInstance.GetLatestStateDB()
	}
	iterator := &MinerIterator{db: db, iterator: accountdb.DataIterator(db, []byte("")), logger: mm.logger, accountdb: accountdb}
	return iterator
}

func (mm *MinerManager) Close() {
	if nil != mm.pkCache {
		mm.pkCache.Close()
	}
}

type MinerIterator struct {
	db        common.Address
	iterator  *trie.Iterator
	logger    log.Logger
	accountdb *account.AccountDB
}

func (mi *MinerIterator) Current() (*types.Miner, error) {
	var miner types.Miner
	err := json.Unmarshal(mi.iterator.Value, &miner)
	if err != nil {
		return nil, err
	}

	if len(miner.Id) == 0 {
		err = errors.New("empty miner")
		return nil, err
	}

	key := common.Sha256(miner.Id)
	miner.Stake = utility.ByteToUInt64(mi.accountdb.GetData(mi.db, key))
	key = common.Sha256(key)
	miner.Account = mi.accountdb.GetData(mi.db, key)
	key = common.Sha256(key)
	status := mi.accountdb.GetData(mi.db, key)
	if nil != status && 1 == len(status) {
		miner.Status = status[0]
	}

	if miner.Status == common.MinerStatusAbort {
		err = errors.New("abort miner")
	}

	return &miner, err
}

func (mi *MinerIterator) Next() bool {
	if nil == mi.iterator {
		return false
	}

	return mi.iterator.Next()
}

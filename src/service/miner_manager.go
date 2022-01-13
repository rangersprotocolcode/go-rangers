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

package service

import (
	"bytes"
	"com.tuntun.rocket/node/src/common"
	crypto "com.tuntun.rocket/node/src/eth_crypto"
	"com.tuntun.rocket/node/src/middleware/db"
	"com.tuntun.rocket/node/src/middleware/log"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/storage/trie"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"errors"
	"fmt"
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
	MinerManagerImpl.logger = log.GetLoggerByIndex(log.CoreLogConfig, common.GlobalConf.GetString("instance", "index", ""))
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
		accountdb = AccountDBManagerInstance.GetLatestStateDB()
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
	accountDB, err := AccountDBManagerInstance.GetAccountDBByHash(hash)
	if err != nil {
		mm.logger.Errorf("Get account db by height %d error:%s", height, err.Error())
		return 0
	}

	_, proposers := mm.GetProposerTotalStakeWithDetail(height, accountDB)

	return uint64(len(proposers))
}

func (mm *MinerManager) MinerIterator(minerType byte, hash common.Hash) *MinerIterator {
	accountDB, err := AccountDBManagerInstance.GetAccountDBByHash(hash)
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
	key = common.Sha256(key)
	accountdb.SetData(db, key, []byte{miner.Status})
}

func (mm *MinerManager) CheckContractedAddress(source []byte, miner *types.Miner, header *types.BlockHeader, accountDB *account.AccountDB) {
	// checkAccount if it's a contract
	var contractAddress common.Address
	contractAddress.SetBytes(miner.Account)
	if !accountDB.IsContract(contractAddress) {
		mm.logger.Debugf("not a contractAddress, %s", common.ToHex(miner.Account))
		return
	}

	magic := accountDB.GetData(contractAddress, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2})
	if nil == magic || 0 != bytes.Compare(magic, common.FromHex("0x0x00000000000000000000000000000000DeaDBeef")) {
		mm.logger.Debugf("no magic number: %s", common.ToHex(magic))
		return
	}

	// initial contract
	accountDB.SetData(contractAddress, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}, source)

	stake := utility.Uint64ToBigInt(miner.Stake)
	accountDB.SetData(contractAddress, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3}, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	accountDB.SetData(contractAddress, crypto.Keccak256([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3}), stake.Bytes())

	accountDB.SetData(contractAddress, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	accountDB.SetData(contractAddress, crypto.Keccak256([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}), utility.UInt64ToByte(header.Height))
	mm.logger.Debugf("set contracAddress: %s by admin: %s, stake: %s, height: %d", common.ToHex(miner.Account), common.ToHex(source), stake.String(), header.Height)
}

// 创世矿工用
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

func (mm *MinerManager) RemoveMiner(id []byte, ttype byte, accountdb *account.AccountDB, left uint64) {
	mm.logger.Debugf("Miner manager remove miner %d", ttype)
	db := mm.getMinerDatabaseAddress(ttype)

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
		accountdb = AccountDBManagerInstance.GetLatestStateDB()
	}
	iterator := &MinerIterator{db: db, iterator: accountdb.DataIterator(db, []byte("")), logger: mm.logger, accountdb: accountdb}
	return iterator
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

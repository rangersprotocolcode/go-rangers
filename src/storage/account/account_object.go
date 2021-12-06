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

package account

import (
	"bytes"
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/trie"
	"fmt"
	"golang.org/x/crypto/sha3"
	"math/big"
	"sync"
)

const (
	NFTSET_TYPE = 1
	NFT_TYPE    = 2
)

var emptyCodeHash = sha3.Sum256(nil)

type Storage map[string][]byte

func (s Storage) String() (str string) {
	for key, value := range s {
		str += fmt.Sprintf("%X : %X\n", key, value)
	}

	return
}

func (s Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range s {
		cpy[key] = value
	}

	return cpy
}

// accountObject represents an account which is being modified.
//
// The usage pattern is as follows:
// First you need to obtain a account object.
// Account values can be accessed and modified through the object.
// Finally, call CommitTrie to write the modified storage trie into a database.
type accountObject struct {
	address  common.Address
	addrHash common.Hash // hash of address of the account
	data     Account
	db       *AccountDB

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error

	trie Trie // storage trie, which becomes non-nil on first access

	nftSet      []byte
	dirtyNFTSet bool // true if the code was updated

	// SetData用
	cachedLock    sync.RWMutex
	cachedStorage Storage // Storage cache of original entries to dedup rewrites
	dirtyStorage  Storage // Storage entries that need to be flushed to disk

	suicided bool
	touched  bool
	deleted  bool
	onDirty  func(addr common.Address)
}

// empty returns whether the account is considered empty.
func (ao *accountObject) empty() bool {
	return (ao.data.NFTSetDefinitionHash == nil || 0 == bytes.Compare(ao.data.NFTSetDefinitionHash, emptyCodeHash[:])) && ao.data.Nonce == 0 && ao.data.Balance.Sign() == 0 && len(ao.cachedStorage) == 0 && len(ao.dirtyStorage) == 0
}

// Account is the consensus representation of accounts.
// These objects are stored in the main account trie.
type Account struct {
	Nonce                uint64
	Root                 common.Hash
	kind                 byte
	NFTSetDefinitionHash []byte
	Balance              *big.Int
}

// newObject creates a account object.
func newAccountObject(db *AccountDB, address common.Address, data Account, onDirty func(addr common.Address)) *accountObject {
	if data.Balance == nil {
		data.Balance = new(big.Int)
	}
	if data.NFTSetDefinitionHash == nil {
		data.NFTSetDefinitionHash = emptyCodeHash[:]
	}

	ao := &accountObject{
		db:            db,
		address:       address,
		addrHash:      sha3.Sum256(address[:]),
		data:          data,
		cachedStorage: make(Storage),
		dirtyStorage:  make(Storage),
		onDirty:       onDirty,
	}

	return ao
}

// setError remembers the first non-nil error it is called with.
func (ao *accountObject) setError(err error) {
	if ao.dbErr == nil {
		ao.dbErr = err
	}
}

// markSuicided only marked
func (ao *accountObject) markSuicided() {
	ao.suicided = true
	if ao.onDirty != nil {
		ao.onDirty(ao.Address())
		ao.onDirty = nil
	}
}

func (ao *accountObject) touch() {
	ao.db.transitions = append(ao.db.transitions, touchChange{
		account:   &ao.address,
		prev:      ao.touched,
		prevDirty: ao.onDirty == nil,
	})
	if ao.onDirty != nil {
		ao.onDirty(ao.Address())
		ao.onDirty = nil
	}
	ao.touched = true
}

func (ao *accountObject) getTrie(db AccountDatabase) Trie {
	if ao.trie == nil {
		tr, err := db.OpenStorageTrie(ao.addrHash, ao.data.Root)
		if err != nil {
			common.DefaultLogger.Debugf("OpenStorageTrie error! root:%v,err:%s", ao.data.Root, err.Error())
			tr, err = db.OpenStorageTrie(ao.addrHash, common.Hash{})
			if tr == nil {
				common.DefaultLogger.Debugf("OpenStorageTrie get nil! root:%v,err:%s", common.Hash{}, err.Error())
			}
			ao.setError(fmt.Errorf("can't create storage trie: %v", err))
		}
		ao.trie = tr
	}
	return ao.trie
}

// GetData retrieves a value from the account storage trie.
func (ao *accountObject) GetData(db AccountDatabase, key []byte) []byte {
	ao.cachedLock.RLock()
	// If we have the original value cached, return that
	value, exists := ao.cachedStorage[string(key)]
	ao.cachedLock.RUnlock()
	if exists {
		return value
	}

	// Otherwise load the value from the database
	return ao.GetCommittedData(db, key)
}

func (ao *accountObject) GetCommittedData(db AccountDatabase, key []byte) []byte {
	trie := ao.getTrie(db)
	if nil == trie {
		accountLog.Errorf("Account Obj get date nil! address:%s,key:%s", ao.address.GetHexString(), common.ToHex(key))
		return nil
	}
	value, err := trie.TryGet(key)
	if err != nil {
		ao.setError(err)
		return nil
	}

	if value != nil {
		ao.cachedLock.Lock()
		ao.cachedStorage[string(key)] = value
		ao.cachedLock.Unlock()
	}
	return value
}

// SetData updates a value in account storage.
func (ao *accountObject) SetData(db AccountDatabase, key []byte, value []byte) {
	preValue := ao.GetData(db, key)
	if 0 == bytes.Compare(value, preValue) {
		return
	}

	ao.db.transitions = append(ao.db.transitions, storageChange{
		account:  &ao.address,
		key:      key,
		prevalue: preValue,
	})
	ao.setData(key, value)
}

func (ao *accountObject) RemoveData(db AccountDatabase, key []byte) {
	ao.SetData(db, key, nil)
}

func (ao *accountObject) setData(key []byte, value []byte) {
	ao.cachedLock.Lock()
	ao.cachedStorage[string(key)] = value
	ao.cachedLock.Unlock()
	ao.dirtyStorage[string(key)] = value

	if ao.onDirty != nil {
		ao.onDirty(ao.Address())
		ao.onDirty = nil
	}
}

// updateTrie writes cached storage modifications into the object's storage trie.
func (ao *accountObject) updateTrie(db AccountDatabase) Trie {
	tr := ao.getTrie(db)

	// Update all the dirty slots in the trie
	for key, value := range ao.dirtyStorage {
		delete(ao.dirtyStorage, key)
		if value == nil {
			ao.setError(tr.TryDelete([]byte(key)))
			continue
		}

		ao.setError(tr.TryUpdate([]byte(key), value[:]))
	}

	return tr
}

// UpdateRoot sets the trie root to the current root hash of
func (ao *accountObject) updateRoot(db AccountDatabase) {
	ao.updateTrie(db)
	ao.data.Root = ao.trie.Hash()
}

// CommitTrie the storage trie of the object to db.
// This updates the trie root.
func (ao *accountObject) CommitTrie(db AccountDatabase) error {
	ao.updateTrie(db)
	if ao.dbErr != nil {
		return ao.dbErr
	}
	root, err := ao.trie.Commit(nil)

	if err == nil {
		ao.data.Root = root
		//ao.db.db.PushTrie(root, ao.trie)
	}
	return err
}

//AddBalance is used to add funds to the destination account of a transfer.
func (ao *accountObject) AddBalance(amount *big.Int) {
	// We must check emptiness for the objects such that the account
	// clearing (0,0,0 objects) can take effect.
	if amount.Sign() == 0 {
		if ao.empty() {
			ao.touch()
		}
		return
	}
	ao.SetBalance(new(big.Int).Add(ao.Balance(), amount))
}

// SubBalance is used to remove funds from the origin account of a transfer.
func (ao *accountObject) SubBalance(amount *big.Int) *big.Int {
	left := new(big.Int).Sub(ao.Balance(), amount)
	if amount.Sign() == 0 {
		return left
	}
	ao.SetBalance(left)
	return left
}

func (ao *accountObject) SetBalance(amount *big.Int) {
	ao.db.transitions = append(ao.db.transitions, balanceChange{
		account: &ao.address,
		prev:    new(big.Int).Set(ao.data.Balance),
	})
	ao.setBalance(amount)
}

func (ao *accountObject) setBalance(amount *big.Int) {
	ao.data.Balance = amount
	if ao.onDirty != nil {
		ao.onDirty(ao.Address())
		ao.onDirty = nil
	}
}

func (ao *accountObject) Balance() *big.Int {
	return ao.data.Balance
}

func (ao *accountObject) deepCopy(db *AccountDB, onDirty func(addr common.Address)) *accountObject {
	accountObject := newAccountObject(db, ao.address, ao.data, onDirty)
	if ao.trie != nil {
		accountObject.trie = db.db.CopyTrie(ao.trie)
	}

	accountObject.nftSet = ao.nftSet
	accountObject.dirtyStorage = ao.dirtyStorage.Copy()
	accountObject.cachedStorage = ao.dirtyStorage.Copy()
	accountObject.suicided = ao.suicided
	accountObject.deleted = ao.deleted
	return accountObject
}

// Returns the address of the contract/account
func (ao *accountObject) Address() common.Address {
	return ao.address
}

// DataIterator returns a new key-value iterator from a node iterator
func (ao *accountObject) DataIterator(db AccountDatabase, prefix []byte) *trie.Iterator {
	if ao.trie == nil {
		ao.getTrie(db)
	}
	return trie.NewIterator(ao.trie.NodeIterator(prefix))
}

func (ao *accountObject) IncreaseNonce() uint64 {
	ao.db.transitions = append(ao.db.transitions, nonceChange{
		account: &ao.address,
		prev:    ao.data.Nonce,
	})
	ao.setNonce(ao.data.Nonce + 1)
	return ao.data.Nonce
}

// setNFTSetDefinition update nonce in account storage.
func (ao *accountObject) SetNonce(nonce uint64) {
	ao.db.transitions = append(ao.db.transitions, nonceChange{
		account: &ao.address,
		prev:    ao.data.Nonce,
	})
	ao.setNonce(nonce)
}

func (ao *accountObject) setNonce(nonce uint64) {
	ao.data.Nonce = nonce
	if ao.onDirty != nil {
		ao.onDirty(ao.Address())
		ao.onDirty = nil
	}
}

// CodeHash returns code's hash
func (ao *accountObject) NFTSetDefinitionHash() []byte {
	return ao.data.NFTSetDefinitionHash
}

func (ao *accountObject) Nonce() uint64 {
	return ao.data.Nonce
}

func (ao *accountObject) IsNFT() bool {
	return ao.data.kind == NFT_TYPE
}

func (ao *accountObject) getOrCreateLockResource(result map[string]*types.LockResource, key string) *types.LockResource {
	lockResource := result[key]
	if nil == lockResource {
		lockResource = &types.LockResource{
			Coin: make(map[string]string),
			FT:   make(map[string]string),
			NFT:  make([]types.NFTID, 0),
		}
		result[key] = lockResource
	}

	return lockResource
}

func (ao *accountObject) SetCode(codeHash common.Hash, code []byte) {
	ao.SetNFTSetDefinition(codeHash, code)
}

func (ao *accountObject) CodeHash() []byte {
	return ao.data.NFTSetDefinitionHash
}

func (ao *accountObject) Code(db AccountDatabase) []byte {
	return ao.nftSetDefinition(db)
}

func (ao *accountObject) CodeSize(db AccountDatabase) int {
	if ao.nftSet != nil {
		return len(ao.nftSet)
	}
	if bytes.Equal(ao.CodeHash(), emptyCodeHash[:]) {
		return 0
	}
	size, err := db.ContractCodeSize(ao.addrHash, common.BytesToHash(ao.CodeHash()))
	if err != nil {
		ao.setError(fmt.Errorf("can't load code size %x: %v", ao.CodeHash(), err))
	}
	return size
}

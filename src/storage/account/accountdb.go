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
	"fmt"
	"math/big"
	"sort"
	"sync"

	"com.tuntun.rocket/node/src/storage/trie"

	"com.tuntun.rocket/node/src/common"
	"golang.org/x/crypto/sha3"

	"com.tuntun.rocket/node/src/storage/rlp"
)

type revision struct {
	id           int
	journalIndex int
}

var (
	// emptyData is the known root hash of an empty trie.
	emptyData = sha3.Sum256(nil)

	// emptyCode is the known hash of the empty TVM bytecode.
	emptyCode = sha3.Sum256(nil)
)

// AccountDB are used to store anything
// within the merkle trie. AccountDB take care of caching and storing
// nested states. It's the general query interface to retrieve:
// * Contracts
// * Accounts
type AccountDB struct {
	db   AccountDatabase
	trie Trie

	accountObjectsLock  *sync.Mutex
	accountObjects      *sync.Map
	accountObjectsDirty map[common.Address]struct{}

	// DB error.
	// Account objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by AccountDB.Commit.
	dbErr error

	refund uint64

	transitions    transition
	validRevisions []revision
	nextRevisionID int
}

// Create a new account from a given trie.
func NewAccountDB(root common.Hash, db AccountDatabase) (*AccountDB, error) {
	tr, err := db.OpenTrie(root)
	if err != nil {
		return nil, err
	}
	accountDb := &AccountDB{
		db:                  db,
		trie:                tr,
		accountObjects:      new(sync.Map),
		accountObjectsDirty: make(map[common.Address]struct{}),
		accountObjectsLock:  new(sync.Mutex),
	}
	return accountDb, nil
}

// setError remembers the first non-nil error it is called with.
func (adb *AccountDB) setError(err error) {
	if adb.dbErr == nil {
		adb.dbErr = err
	}
}

// RemoveData set data nil
func (adb *AccountDB) RemoveData(addr common.Address, key []byte) {
	adb.SetData(addr, key, nil)
}

// Error get the first non-nil error it is called with.
func (adb *AccountDB) Error() error {
	return adb.dbErr
}

// Reset clears out all ephemeral state objects from the state db, but keeps
// the underlying state trie to avoid reloading data for the next operations.
func (adb *AccountDB) Reset(root common.Hash) error {
	tr, err := adb.db.OpenTrie(root)
	if err != nil {
		return err
	}
	adb.trie = tr
	adb.accountObjects = new(sync.Map)
	adb.accountObjectsLock = new(sync.Mutex)
	adb.accountObjectsDirty = make(map[common.Address]struct{})
	adb.clearJournalAndRefund()
	return nil
}

// AddRefund adds gas to the refund counter
func (adb *AccountDB) AddRefund(gas uint64) {
	adb.transitions = append(adb.transitions, refundChange{prev: adb.refund})
	adb.refund += gas
}

// Exist reports whether the given account address exists in the state.
// Notably this also returns true for suicided accounts.
func (adb *AccountDB) Exist(addr common.Address) bool {
	return adb.getAccountObject(addr, false) != nil
}

// Empty returns whether the state object is either non-existent
// or (balance = nonce = code = 0)
func (adb *AccountDB) Empty(addr common.Address) bool {
	so := adb.getAccountObject(addr, false)
	return so == nil || so.empty()
}

// GetBalance Retrieve the balance from the given address or 0 if object not found
func (adb *AccountDB) GetBalance(addr common.Address) *big.Int {
	accountObject := adb.getAccountObject(addr, false)
	if accountObject != nil {
		return accountObject.Balance()
	}
	return common.Big0
}

// GetBalance Retrieve the nonce from the given address or 0 if object not found
func (adb *AccountDB) GetNonce(addr common.Address) uint64 {
	accountObject := adb.getAccountObject(addr, false)
	if accountObject != nil {
		return accountObject.Nonce()
	}

	return 0
}

// GetData retrieves a value from the account storage trie.
func (adb *AccountDB) GetData(a common.Address, key []byte) []byte {
	stateObject := adb.getAccountObject(a, false)
	if stateObject != nil {
		return stateObject.GetData(adb.db, key)
	}
	return nil
}

// Database retrieves the low level database supporting the lower level trie ops.
func (adb *AccountDB) Database() AccountDatabase {
	return adb.db
}

// StorageTrie returns the storage trie of an account.
// The return value is a copy and is nil for non-existent accounts.
func (adb *AccountDB) StorageTrie(a common.Address) Trie {
	stateObject := adb.getAccountObject(a, false)
	if stateObject == nil {
		return nil
	}
	cpy := stateObject.deepCopy(adb, nil)
	return cpy.updateTrie(adb.db)
}

// HasSuicided returns this account is suicided
func (adb *AccountDB) HasSuicided(addr common.Address) bool {
	stateObject := adb.getAccountObject(addr, false)
	if stateObject != nil {
		return stateObject.suicided
	}
	return false
}

// AddBalance adds amount to the account associated with addr.
func (adb *AccountDB) AddBalance(addr common.Address, amount *big.Int) {
	stateObject := adb.getOrNewAccountObject(addr)
	if stateObject != nil {
		stateObject.AddBalance(amount)
	}
}

// SubBalance subtracts amount from the account associated with addr.
func (adb *AccountDB) SubBalance(addr common.Address, amount *big.Int) (left *big.Int) {
	stateObject := adb.getOrNewAccountObject(addr)
	if stateObject != nil {
		left = stateObject.SubBalance(amount)
	}
	return
}

func (adb *AccountDB) SetBalance(addr common.Address, amount *big.Int) {
	stateObject := adb.getOrNewAccountObject(addr)
	if stateObject != nil {
		stateObject.SetBalance(amount)
	}
}

func (adb *AccountDB) SetNonce(addr common.Address, nonce uint64) {
	stateObject := adb.getOrNewAccountObject(addr)
	if stateObject != nil {
		stateObject.SetNonce(nonce)
	}
}

func (adb *AccountDB) SetData(addr common.Address, key []byte, value []byte) {
	stateObject := adb.getOrNewAccountObject(addr)
	if stateObject != nil {
		stateObject.SetData(adb.db, key, value)
	}
}

func (adb *AccountDB) Transfer(sender, recipient common.Address, amount *big.Int) {
	// Escape if amount is zero
	if amount.Sign() <= 0 {
		return
	}
	adb.SubBalance(sender, amount)
	adb.AddBalance(recipient, amount)
}

func (adb *AccountDB) CanTransfer(addr common.Address, amount *big.Int) bool {
	if amount.Sign() == -1 {
		return false
	}
	return adb.GetBalance(addr).Cmp(amount) >= 0
}

// Suicide marks the given account as suicided.
// This clears the account balance.
//
// The account's account object is still available until the account is committed,
// getAccountObject will return a non-nil account after Suicide.
func (adb *AccountDB) Suicide(addr common.Address) bool {
	stateObject := adb.getAccountObject(addr, false)
	if stateObject == nil {
		return false
	}
	adb.transitions = append(adb.transitions, suicideChange{
		account:     &addr,
		prev:        stateObject.suicided,
		prevbalance: new(big.Int).Set(stateObject.Balance()),
	})
	stateObject.markSuicided()
	stateObject.data.Balance = new(big.Int)

	return true
}

// updateStateObject writes the given object to the trie.
func (adb *AccountDB) updateAccountObject(stateObject *accountObject) {
	addr := stateObject.Address()
	data, err := rlp.EncodeToBytes(stateObject.data)
	if err != nil {
		panic(fmt.Errorf("can't serialize object at %x: %v", addr[:], err))
	}
	common.DefaultLogger.Errorf("ao: %s, size: %d", stateObject.addrHash.String(), len(data))

	adb.setError(adb.trie.TryUpdate(addr[:], data))
}

// deleteStateObject removes the given object from the state trie.
func (adb *AccountDB) deleteAccountObject(stateObject *accountObject) {
	stateObject.deleted = true
	addr := stateObject.Address()
	adb.setError(adb.trie.TryDelete(addr[:]))
}

// Retrieve a account object given by the address. Returns nil if not found.
func (adb *AccountDB) getAccountObjectFromTrie(addr common.Address) (stateObject *accountObject) {
	enc, err := adb.trie.TryGet(addr[:])
	if len(enc) == 0 {
		adb.setError(err)
		return nil
	}
	var data Account
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		return nil
	}

	obj := newAccountObject(adb, addr, data, adb.MarkAccountObjectDirty)
	return obj
}

func (adb *AccountDB) getOrNewAccountObject(addr common.Address) *accountObject {
	return adb.getAccountObject(addr, true)
}

// Retrieve a state object given by the address. Returns nil if not found.
func (adb *AccountDB) getAccountObject(addr common.Address, isCreateWhenNil bool) (stateObject *accountObject) {
	if obj, ok := adb.accountObjects.Load(addr); ok {
		obj2 := obj.(*accountObject)
		if obj2.deleted {
			return nil
		}
		return obj2
	}

	adb.accountObjectsLock.Lock()
	defer adb.accountObjectsLock.Unlock()

	if obj, ok := adb.accountObjects.Load(addr); ok {
		obj2 := obj.(*accountObject)
		if obj2.deleted {
			return nil
		}
		return obj2
	}

	obj := adb.getAccountObjectFromTrie(addr)
	if obj != nil {
		adb.setAccountObject(obj)
	}

	if !isCreateWhenNil {
		return obj
	}

	if obj == nil || obj.deleted {
		obj = adb.createObject(addr, obj)
	}
	return obj
}

func (adb *AccountDB) createObject(addr common.Address, prev *accountObject) *accountObject {
	newobj := newAccountObject(adb, addr, Account{}, adb.MarkAccountObjectDirty)
	newobj.setNonce(0) // sets the object to dirty
	if prev == nil {
		adb.transitions = append(adb.transitions, createObjectChange{account: &addr})
	} else {
		adb.transitions = append(adb.transitions, resetObjectChange{prev: prev})
	}
	adb.setAccountObject(newobj)
	return newobj
}

func (adb *AccountDB) setAccountObject(object *accountObject) {
	adb.accountObjects.LoadOrStore(object.Address(), object)
}

// MarkAccountObjectDirty Record the modified accounts
func (adb *AccountDB) MarkAccountObjectDirty(addr common.Address) {
	adb.accountObjectsDirty[addr] = struct{}{}
}

// DataIterator returns a new key-value iterator from a node iterator
func (adb *AccountDB) DataIterator(addr common.Address, prefix []byte) *trie.Iterator {
	stateObject := adb.getAccountObject(addr, false)
	if stateObject != nil {
		return stateObject.DataIterator(adb.db, prefix)
	}
	return nil
}

////DataNext returns next key-value data from iterator
//func (adb *AccountDB) DataNext(iterator uintptr) []byte {
//	iter := (*trie.Iterator)(unsafe.Pointer(iterator))
//	if iter == nil {
//		return `{"key":"","value":"","hasValue":0}`
//	}
//	hasValue := 1
//	var key string
//	var value string
//	if len(iter.Key) != 0 {
//		key = string(iter.Key)
//		value = string(iter.Value)
//	}
//
//	// Means no data
//	if !iter.Next() {
//		hasValue = 0
//	}
//	if key == "" {
//		return fmt.Sprintf(`{"key":"","value":"","hasValue":%d}`, hasValue)
//	}
//	if len(value) > 0 {
//		valueType := value[0:1]
//		if valueType == "0" { // This is map node
//			hasValue = 2
//		} else {
//			value = value[1:]
//		}
//	} else {
//		return `{"key":"","value":"","hasValue":0}`
//	}
//	return fmt.Sprintf(`{"key":"%s","value":%s,"hasValue":%d}`, key, value, hasValue)
//}

//// Snapshot returns an identifier for the current revision of the account.
func (adb *AccountDB) Snapshot() int {
	id := adb.nextRevisionID
	adb.nextRevisionID++
	adb.validRevisions = append(adb.validRevisions, revision{id, len(adb.transitions)})
	return id
}

// RevertToSnapshot reverts all state changes made since the given revision.
func (adb *AccountDB) RevertToSnapshot(revid int) {
	// Find the snapshot in the stack of valid snapshots.
	idx := sort.Search(len(adb.validRevisions), func(i int) bool {
		return adb.validRevisions[i].id >= revid
	})
	if idx == len(adb.validRevisions) || adb.validRevisions[idx].id != revid {
		panic(fmt.Errorf("revision id %v cannot be reverted", revid))
	}
	snapshot := adb.validRevisions[idx].journalIndex
	for i := len(adb.transitions) - 1; i >= snapshot; i-- {
		adb.transitions[i].undo(adb)
	}
	adb.transitions = adb.transitions[:snapshot]
	adb.validRevisions = adb.validRevisions[:idx]
}

// GetRefund returns the current value of the refund counter.
func (adb *AccountDB) GetRefund() uint64 {
	return adb.refund
}

// Finalise finalises the state by removing the self destructed objects
// and clears the journal as well as the refunds.
func (adb *AccountDB) Finalise(deleteEmptyObjects bool) {
	for addr := range adb.accountObjectsDirty {
		object, exist := adb.accountObjects.Load(addr)
		if !exist {
			continue
		}
		accountObject := object.(*accountObject)
		if accountObject.suicided || (deleteEmptyObjects && accountObject.empty()) {
			adb.deleteAccountObject(accountObject)
		} else {
			accountObject.updateRoot(adb.db)
			adb.updateAccountObject(accountObject)
		}
	}

	adb.clearJournalAndRefund()
}

// IntermediateRoot computes the current root hash of the state trie.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (adb *AccountDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	adb.Finalise(deleteEmptyObjects)
	return adb.trie.Hash()
}

func (adb *AccountDB) clearJournalAndRefund() {
	adb.transitions = nil
	adb.validRevisions = adb.validRevisions[:0]
	adb.refund = 0
}

// Commit writes the state to the underlying in-memory trie database.
func (adb *AccountDB) Commit(deleteEmptyObjects bool) (root common.Hash, err error) {
	defer adb.clearJournalAndRefund()
	var e *error
	adb.accountObjects.Range(func(key, value interface{}) bool {
		addr := key.(common.Address)
		_, isDirty := adb.accountObjectsDirty[addr]
		accountObject := value.(*accountObject)
		switch {
		case accountObject.suicided || (isDirty && deleteEmptyObjects && accountObject.empty()):
			adb.deleteAccountObject(accountObject)
		case isDirty:
			// Write any storage changes in the state object to its storage trie.
			if err := accountObject.CommitTrie(adb.db); err != nil {
				e = &err
				return false
			}
			// Update the object in the main account trie.
			adb.updateAccountObject(accountObject)
		}
		delete(adb.accountObjectsDirty, addr)
		return true
	})
	if e != nil {
		return common.Hash{}, *e
	}
	root, err = adb.trie.Commit(func(leaf []byte, parent common.Hash) error {
		var account Account
		if err := rlp.DecodeBytes(leaf, &account); err != nil {
			return nil
		}
		if account.Root != emptyData {
			adb.db.TrieDB().Reference(account.Root, parent)
		}

		return nil
	})
	return root, err
}

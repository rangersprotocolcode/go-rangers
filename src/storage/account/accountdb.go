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

package account

import (
	crypto "com.tuntun.rangers/node/src/eth_crypto"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/utility"
	"fmt"
	"math/big"
	"sort"
	"sync"

	"com.tuntun.rangers/node/src/storage/trie"

	"com.tuntun.rangers/node/src/common"
	"golang.org/x/crypto/sha3"

	"com.tuntun.rangers/node/src/storage/rlp"
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

	// Per-transaction access list
	accessList *accessList

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

	// Transient storage
	transientStorage transientStorage

	transitions    transition
	validRevisions []revision
	nextRevisionID int

	thash, bhash common.Hash
	txIndex      int
	logs         map[common.Hash][]*types.Log
	logSize      uint
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
		accessList:          newAccessList(),
		logs:                make(map[common.Hash][]*types.Log),
		transientStorage:    newTransientStorage(),
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
	adb.accessList = newAccessList()
	adb.thash = common.Hash{}
	adb.bhash = common.Hash{}
	adb.txIndex = 0
	adb.logs = make(map[common.Hash][]*types.Log)
	adb.logSize = 0
	return nil
}

func (adb *AccountDB) Clean() {
	adb.accountObjects = new(sync.Map)
	adb.accountObjectsLock = new(sync.Mutex)
	adb.accountObjectsDirty = make(map[common.Address]struct{})
	adb.clearJournalAndRefund()
}

// Prepare sets the current transaction hash and index and block hash which is
// used when the EVM emits new state logs.
func (s *AccountDB) Prepare(thash, bhash common.Hash, ti int) {
	s.thash = thash
	s.bhash = bhash
	s.txIndex = ti
	s.accessList = newAccessList()
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
	return adb.GetFT(addr, common.BLANCE_NAME)
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
// amount decimal 18
func (adb *AccountDB) AddBalance(addr common.Address, amount *big.Int) {
	adb.AddFT(addr, common.BLANCE_NAME, amount)
}

// SubBalance subtracts amount from the account associated with addr.
func (adb *AccountDB) SubBalance(addr common.Address, amount *big.Int) (left *big.Int) {
	left, _ = adb.SubFT(addr, common.BLANCE_NAME, amount)
	return
}

func (adb *AccountDB) SetBalance(addr common.Address, amount *big.Int) {
	adb.SetFT(addr, common.BLANCE_NAME, amount)
}

func (self *AccountDB) setBalance(addr common.Address, balance *big.Int) {
	found, contract, position, decimal := self.GetERC20Binding(common.BLANCE_NAME)
	if found {
		account := self.getOrNewAccountObject(contract)
		account.setData(self.GetERC20Key(addr, position), utility.FormatDecimalForERC20(balance, int64(decimal)).Bytes())
		return
	}
}

func (adb *AccountDB) SetNonce(addr common.Address, nonce uint64) {
	stateObject := adb.getOrNewAccountObject(addr)
	if stateObject != nil {
		stateObject.SetNonce(nonce)
	}
}

func (adb *AccountDB) IncreaseNonce(addr common.Address) uint64 {
	stateObject := adb.getOrNewAccountObject(addr)
	if stateObject != nil {
		result := stateObject.IncreaseNonce()
		accountLog.Debugf("addr: %s, nonce: %d", addr.GetHexString(), result)
		return result
	}
	return 0
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
		prevbalance: new(big.Int).Set(adb.GetBalance(addr)),
	})
	stateObject.markSuicided()
	adb.setBalance(addr, common.Big0)
	return true
}

// updateStateObject writes the given object to the trie.
func (adb *AccountDB) updateAccountObject(stateObject *accountObject) {
	addr := stateObject.Address()
	data, err := rlp.EncodeToBytes(stateObject.data)
	if err != nil {
		panic(fmt.Errorf("can't serialize object at %x: %v", addr[:], err))
	}
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

// // Snapshot returns an identifier for the current revision of the account.
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
			if accountObject.nftSet != nil && accountObject.dirtyNFTSet {
				adb.db.TrieDB().InsertBlob(common.BytesToHash(accountObject.NFTSetDefinitionHash()), accountObject.nftSet)
				accountObject.dirtyNFTSet = false
			}

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
		code := common.BytesToHash(account.NFTSetDefinitionHash)
		if code != emptyCode {
			adb.db.TrieDB().Reference(code, parent)
		}
		return nil
	})
	return root, err
}

// ----------------------------------------------------add interface method to implement-------------------------------
// CreateAccount explicitly creates a state object. If a state object with the address
// already exists the balance is carried over to the new account.
//
// CreateAccount is called during the EVM CREATE operation. The situation might arise that
// a contract does the following:
//
//  1. sends funds to sha(account ++ (nonce + 1))
//  2. tx_create(sha(account ++ nonce)) (note that this gets the address of 1)
//
// Carrying over the balance ensures that Ether doesn't disappear.
func (adb *AccountDB) CreateAccount(addr common.Address) {
	adb.getOrNewAccountObject(addr)
}

func (adb *AccountDB) IsContract(addr common.Address) bool {
	contract := adb.GetCode(addr)
	return nil != contract && 0 != len(contract)
}

func (adb *AccountDB) GetCode(addr common.Address) []byte {
	stateObject := adb.getAccountObject(addr, false)
	if stateObject != nil {
		return stateObject.Code(adb.db)
	}
	return nil
}

func (adb *AccountDB) GetCodeSize(addr common.Address) int {
	stateObject := adb.getAccountObject(addr, false)
	if stateObject != nil {
		return stateObject.CodeSize(adb.db)
	}
	return 0
}

func (adb *AccountDB) GetCodeHash(addr common.Address) common.Hash {
	stateObject := adb.getAccountObject(addr, false)
	if stateObject == nil {
		return common.Hash{}
	}
	return common.BytesToHash(stateObject.CodeHash())

	return common.Hash{}
}

func (adb *AccountDB) SetCode(addr common.Address, code []byte) {
	stateObject := adb.getAccountObject(addr, true)
	if stateObject != nil {
		stateObject.SetCode(crypto.Keccak256Hash(code), code)
	}

}

// GetTransientState gets transient storage for a given account.
func (adb *AccountDB) GetTransientState(addr common.Address, key common.Hash) common.Hash {
	return adb.transientStorage.Get(addr, key)
}

// SetTransientState sets transient storage for a given account. It
// adds the change to the journal so that it can be rolled back
// to its previous value if there is a revert.
func (adb *AccountDB) SetTransientState(addr common.Address, key, value common.Hash) {
	prev := adb.GetTransientState(addr, key)
	if prev == value {
		return
	}

	adb.transitions = append(adb.transitions, transientStorageChange{
		account:  &addr,
		key:      key,
		prevalue: prev,
	})
	adb.setTransientState(addr, key, value)
}

// setTransientState is a lower level setter for transient storage. It
// is called during a revert to prevent modifications to the journal.
func (adb *AccountDB) setTransientState(addr common.Address, key, value common.Hash) {
	adb.transientStorage.Set(addr, key, value)
}

// SubRefund removes gas from the refund counter.
// This method will panic if the refund counter goes below zero
func (adb *AccountDB) SubRefund(gas uint64) {
	adb.transitions = append(adb.transitions, refundChange{prev: adb.refund})
	if gas > adb.refund {
		panic(fmt.Sprintf("Refund counter below zero (gas: %d > refund: %d)", gas, adb.refund))
	}
	adb.refund -= gas

}

// GetCommittedState retrieves a value from the given account's committed storage trie.
func (adb *AccountDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	stateObject := adb.getAccountObject(addr, false)
	if stateObject != nil {
		return common.BytesToHash(stateObject.GetCommittedData(adb.db, hash.Bytes()))
	}

	return common.Hash{}
}

// GetState retrieves a value from the given account's storage trie.
func (adb *AccountDB) GetState(addr common.Address, hash common.Hash) common.Hash {
	stateObject := adb.getAccountObject(addr, false)
	if stateObject != nil {
		data := stateObject.GetData(adb.db, hash.Bytes())
		result := common.BytesToHash(data)
		return result
	}

	return common.Hash{}
}

func (adb *AccountDB) SetState(addr common.Address, key, value common.Hash) {
	stateObject := adb.getAccountObject(addr, true)
	if stateObject != nil {
		stateObject.SetData(adb.db, key.Bytes(), value.Bytes())
	}
}

// AddressInAccessList returns true if the given address is in the access list.
func (adb *AccountDB) AddressInAccessList(addr common.Address) bool {
	return adb.accessList.ContainsAddress(addr)
}

// SlotInAccessList returns true if the given (address, slot)-tuple is in the access list.
func (adb *AccountDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressPresent bool, slotPresent bool) {
	return adb.accessList.Contains(addr, slot)
}

// AddAddressToAccessList adds the given address to the access list
func (adb *AccountDB) AddAddressToAccessList(addr common.Address) {
	if adb.accessList.AddAddress(addr) {
		adb.transitions = append(adb.transitions, accessListAddAccountChange{&addr})
	}
}

// AddSlotToAccessList adds the given (address, slot)-tuple to the access list
func (adb *AccountDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	addrMod, slotMod := adb.accessList.AddSlot(addr, slot)
	if addrMod {
		// In practice, this should not happen, since there is no way to enter the
		// scope of 'address' without having the 'address' become already added
		// to the access list (via call-variant, create, etc).
		// Better safe than sorry, though
		adb.transitions = append(adb.transitions, accessListAddAccountChange{&addr})
	}
	if slotMod {
		adb.transitions = append(adb.transitions, accessListAddSlotChange{
			address: &addr,
			slot:    &slot,
		})
	}

}

func (adb *AccountDB) AddLog(log *types.Log) {
	adb.transitions = append(adb.transitions, addLogChange{txhash: adb.thash})

	log.TxHash = adb.thash
	log.BlockHash = adb.bhash
	log.TxIndex = uint(adb.txIndex)
	log.Index = adb.logSize
	adb.logs[adb.thash] = append(adb.logs[adb.thash], log)
	adb.logSize++
}

func (s *AccountDB) GetLogs(hash common.Hash) []*types.Log {
	return s.logs[hash]
}

// SetStorage replaces the entire storage for the specified account with given
// storage. This function should only be used for debugging.
func (s *AccountDB) SetStorage(addr common.Address, storage map[common.Hash]common.Hash) {
	// SetStorage needs to wipe existing storage. We achieve this by pretending
	// that the account self-destructed earlier in this block, by flagging
	// it in stateObjectsDestruct. The effect of doing so is that storage lookups
	// will not hit disk, since it is assumed that the disk-data is belonging
	// to a previous incarnation of the object.
	stateObject := s.getOrNewAccountObject(addr)
	for k, v := range storage {
		stateObject.SetData(s.db, k.Bytes(), v.Bytes())
	}
}

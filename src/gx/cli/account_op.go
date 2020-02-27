package cli

import (
	"time"
	"sync"
	"encoding/json"
	"fmt"
	"os"
	"crypto/rand"

	"x/src/common"
	"x/src/middleware/db"
	"x/src/consensus/model"
)

const accountUnLockTime = time.Second * 120

var encryptPrivateKey *common.PrivateKey
var encryptPublicKey *common.PublicKey

func init() {
	encryptPrivateKey = common.HexStringToSecKey("0x04b851c3551779125a588b2274cfa6d71604fe6ae1f0df82175bcd6e6c2b23d92a69d507023628b59c15355f3cbc0d8f74633618facd28632a0fb3e9cc8851536c4b3f1ea7c7fd3666ce8334301236c2437d9bed14e5a0793b51a9a6e7a4c46e70")
	pk := encryptPrivateKey.GetPubKey()
	encryptPublicKey = &pk
}

const (
	statusLocked   int8 = 0
	statusUnLocked      = 1

	defaultPassword = "123"
)

type AccountManager struct {
	unlockAccount *AccountInfo

	accounts sync.Map
	db       *db.LDBDatabase

	mu sync.Mutex
}

type AccountInfo struct {
	Account
	Status       int8
	UnLockExpire time.Time
}

type Account struct {
	Address  string
	Pk       string
	Sk       string
	Password string
	Miner    *MinerRaw
}

type MinerRaw struct {
	BPk   string
	BSk   string
	VrfPk string
	VrfSk string
	ID    [32]byte
}

func (am *AccountManager) NewAccount(password string, miner bool) *Result {
	privateKey := common.GenerateKey("")
	publicKey := privateKey.GetPubKey()
	address := publicKey.GetAddress()

	account := &Account{
		Address:  address.GetHexString(),
		Pk:       publicKey.GetHexString(),
		Sk:       privateKey.GetHexString(),
		Password: passwordSha(password),
	}

	if miner {
		id := publicKey.GetID()
		minerDO := model.NewSelfMinerDO(id[:])
		minerRaw := &MinerRaw{
			BPk:   minerDO.PK.GetHexString(),
			BSk:   minerDO.SK.GetHexString(),
			VrfPk: minerDO.VrfPK.GetHexString(),
			VrfSk: minerDO.VrfSK.GetHexString(),
			ID:    id,
		}
		account.Miner = minerRaw
	}
	if err := am.storeAccount(account); err != nil {
		return opError(err)
	}
	return opSuccess(address.GetHexString())
}

func (am *AccountManager) AccountList() *Result {
	iter := am.db.NewIterator()
	addrs := make([]string, 0)
	for iter.Next() {
		addrs = append(addrs, string(iter.Key()))
	}
	return opSuccess(addrs)
}

func (am *AccountManager) Lock(addr string) *Result {
	aci, err := am.getAccountInfo(addr)
	if err != nil {
		return opError(err)
	}
	aci.Status = statusLocked
	return opSuccess(nil)
}

func (am *AccountManager) UnLock(addr string, password string) *Result {
	aci, err := am.getAccountInfo(addr)
	if err != nil {
		return opError(err)
	}
	if aci.Password != passwordSha(password) {
		return opError(ErrPassword)
	}
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.unlockAccount != nil && aci.Address != am.unlockAccount.Address {
		am.unlockAccount.Status = statusLocked
	}

	aci.Status = statusUnLocked
	aci.resetExpireTime()
	am.unlockAccount = aci

	return opSuccess(nil)
}

func (am *AccountManager) AccountInfo() *Result {
	addr := am.currentUnLockedAddr()
	if addr == "" {
		return opError(ErrUnlocked)
	}
	aci, err := am.getAccountInfo(addr)
	if err != nil {
		return opError(err)
	}
	if !aci.unlocked() {
		return opError(ErrUnlocked)
	}
	aci.resetExpireTime()
	return opSuccess(&aci.Account)
}

func (am *AccountManager) DeleteAccount() *Result {
	addr := am.currentUnLockedAddr()
	if addr == "" {
		return opError(ErrUnlocked)
	}
	aci, err := am.getAccountInfo(addr)
	if err != nil {
		return opError(err)
	}
	if !aci.unlocked() {
		return opError(ErrUnlocked)
	}
	am.accounts.Delete(addr)
	am.db.Delete([]byte(addr))
	return opSuccess(nil)
}

func (am *AccountManager) Close() {
	am.db.Close()
}

func initAccountManager(keystore string, readyOnly bool) (accountOp, error) {
	if readyOnly && !dirExists(keystore) {
		accountManager, err := newAccountManager(keystore)
		if err != nil {
			panic(err)
		}
		ret := accountManager.NewAccount(defaultPassword, true)
		if !ret.IsSuccess() {
			fmt.Println(ret.Message)
			panic(ret.Message)
		}
		return accountManager, nil
	}

	if accountManager, err := newAccountManager(keystore); err != nil {
		fmt.Printf("new lelvel db error:%s\n", err.Error())
		return nil, err
	} else {
		return accountManager, nil
	}
}

func newAccountManager(ks string) (*AccountManager, error) {
	accountManagerDB, err := db.NewLDBDatabase(ks, 128, 128)
	if err != nil {
		return nil, fmt.Errorf("new ldb fail:%v", err.Error())
	}
	return &AccountManager{db: accountManagerDB}, nil
}

func (am *AccountManager) loadAccount(addr string) (*Account, error) {
	//fmt.Printf("key:%v\n",[]byte(addr))
	v, err := am.db.Get(common.FromHex(addr))
	//v, err := am.db.Get([]byte(addr))
	if err != nil {
		return nil, err
	}

	bs, err := encryptPrivateKey.Decrypt(rand.Reader, v)
	if err != nil {
		return nil, err
	}

	var acc = new(Account)
	err = json.Unmarshal(bs, acc)
	if err != nil {
		return nil, err
	}

	pk := common.HexStringToPubKey(acc.Pk)
	address := pk.GetAddress()
	acc.Address = address.String()

	bs,_=json.Marshal(acc)
	fmt.Println(string(bs))

	return acc, nil
}

func (am *AccountManager) storeAccount(account *Account) error {
	bs, err := json.Marshal(account)
	if err != nil {
		return err
	}

	ct, err := common.Encrypt(rand.Reader, encryptPublicKey, bs)
	if err != nil {
		return err
	}

	err = am.db.Put(account.Miner.ID[:], ct)
	fmt.Printf("store account:%v\n", account.Miner.ID[:])
	return err
}

func (am *AccountManager) getFirstMinerAccount() *Account {
	iter := am.db.NewIterator()
	for iter.Next() {
		if ac, err := am.getAccountInfo(string(iter.Key())); err != nil {
			panic(fmt.Sprintf("getAccountInfo err,addr=%v,err=%v", iter.Key(), err.Error()))
		} else {
			if ac.Miner != nil {
				return &ac.Account
			}
		}
	}
	return nil
}

func (am *AccountManager) resetExpireTime(addr string) {
	acc, err := am.getAccountInfo(addr)
	if err != nil {
		return
	}
	acc.resetExpireTime()
}

func (am *AccountManager) getAccountInfo(addr string) (*AccountInfo, error) {
	var aci *AccountInfo
	if v, ok := am.accounts.Load(addr); ok {
		aci = v.(*AccountInfo)
	} else {
		acc, err := am.loadAccount(addr)
		if err != nil {
			return nil, err
		}
		aci = &AccountInfo{
			Account: *acc,
		}
		am.accounts.Store(addr, aci)
	}
	return aci, nil
}

func (am *AccountManager) currentUnLockedAddr() string {
	if am.unlockAccount != nil && am.unlockAccount.unlocked() {
		return am.unlockAccount.Address
	}
	return ""
}

func passwordSha(password string) string {
	return common.ToHex(common.Sha256([]byte(password)))
}

func (ai *AccountInfo) unlocked() bool {
	return time.Now().Before(ai.UnLockExpire) && ai.Status == statusUnLocked
}

func (ai *AccountInfo) resetExpireTime() {
	ai.UnLockExpire = time.Now().Add(accountUnLockTime)
}

func dirExists(dir string) bool {
	f, err := os.Stat(dir)
	if err != nil {
		return false
	}
	return f.IsDir()
}

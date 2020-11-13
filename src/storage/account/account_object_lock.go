package account

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/utility"
	"math/big"
	"strings"
)

func (ao *accountObject) lockCoin(db AccountDatabase, key []byte, balance *big.Int) {
	if nil == balance || 0 >= balance.Sign() {
		return
	}

	existed := ao.GetData(db, key)
	value := big.NewInt(0)
	if existed != nil && 0 != len(existed) {
		value.SetBytes(existed)
	}

	value = value.Add(value, balance)
	ao.SetData(db, key, value.Bytes())
}

func (ao *accountObject) unlockCoin(db AccountDatabase, key []byte, balance *big.Int) bool {
	if nil == balance || 0 >= balance.Sign() {
		return true
	}

	existed := ao.GetData(db, key)
	value := big.NewInt(0)
	if existed != nil && 0 != len(existed) {
		value.SetBytes(existed)
	}

	if value.Cmp(balance) < 0 {
		return false
	}
	value = value.Sub(value, balance)
	ao.SetData(db, key, value.Bytes())

	return true
}

func (ao *accountObject) lockBalance(db AccountDatabase, source common.Address, balance *big.Int) {
	key := utility.StrToBytes(common.GenerateLockBalanceKey(source.String()))
	ao.lockCoin(db, key, balance)
}

func (ao *accountObject) unlockBalance(db AccountDatabase, source common.Address, balance *big.Int) bool {
	key := utility.StrToBytes(common.GenerateLockBalanceKey(source.String()))
	return ao.unlockCoin(db, key, balance)
}

func (ao *accountObject) lockBNT(db AccountDatabase, source common.Address, name string, balance *big.Int) {
	key := utility.StrToBytes(common.GenerateLockBNTKey(source.String(), name))
	ao.lockCoin(db, key, balance)
}

func (ao *accountObject) unlockBNT(db AccountDatabase, source common.Address, name string, balance *big.Int) bool {
	key := utility.StrToBytes(common.GenerateLockBNTKey(source.String(), name))
	return ao.unlockCoin(db, key, balance)
}

func (ao *accountObject) lockFT(db AccountDatabase, source common.Address, name string, balance *big.Int) {
	key := utility.StrToBytes(common.GenerateLockFTKey(source.String(), name))
	ao.lockCoin(db, key, balance)
}

func (ao *accountObject) unlockFT(db AccountDatabase, source common.Address, name string, balance *big.Int) bool {
	key := utility.StrToBytes(common.GenerateLockFTKey(source.String(), name))
	return ao.unlockCoin(db, key, balance)
}

func (ao *accountObject) lockNFT(db AccountDatabase, source common.Address, setId, id string) {
	key := utility.StrToBytes(common.GenerateLockNFTKey(source.String(), setId, id))
	ao.SetData(db, key, []byte{0})
}

func (ao *accountObject) unlockNFT(db AccountDatabase, source common.Address, setId, id string) {
	key := utility.StrToBytes(common.GenerateLockNFTKey(source.String(), setId, id))
	ao.SetData(db, key, nil)
}

func (ao *accountObject) getLockedResource(db AccountDatabase, source common.Address) *types.LockResource {
	result := &types.LockResource{}

	// balance
	balance := ao.GetData(db, utility.StrToBytes(common.GenerateLockBalanceKey(source.String())))
	if nil != balance {
		result.Balance = utility.BigIntBytesToStr(balance)
	}

	all := ao.getLockedBNTFTNFT(db)
	if nil != all {
		filtered, ok := all[source.String()]
		if ok {
			result.FT = filtered.FT
			result.NFT = filtered.NFT
			result.Coin = filtered.Coin
		}
	}
	return result
}

func (ao *accountObject) getLockedBNTFTNFT(db AccountDatabase) map[string]*types.LockResource {
	result := make(map[string]*types.LockResource)
	ao.cachedLock.Lock()
	defer ao.cachedLock.Unlock()

	// bnt/ft/nft
	for key, value := range ao.cachedStorage {
		if strings.HasPrefix(key, common.LockBNTKey) {
			source, name := ao.getSourceAndName(key[len(common.LockBNTKey)+1:])
			if 0 == len(source) {
				continue
			}
			target := ao.getOrCreateLockResource(result, source)
			target.Coin[name] = utility.BigIntBytesToStr(value)
		}
		if strings.HasPrefix(key, common.LockFTKey) {
			source, name := ao.getSourceAndName(key[len(common.LockFTKey)+1:])
			if 0 == len(source) {
				continue
			}
			target := ao.getOrCreateLockResource(result, source)
			target.FT[name] = utility.BigIntBytesToStr(value)
		}
		if strings.HasPrefix(key, common.LockNFTKey) {
			source, setId, id := ao.getSourceAndSetIdAndId(key[len(common.LockNFTKey)+1:])
			if 0 == len(source) {
				continue
			}
			target := ao.getOrCreateLockResource(result, source)
			nft := types.NFTID{
				SetId: setId,
				Id:    id,
			}
			target.NFT = append(target.NFT, nft)
		}
	}

	iterator := ao.DataIterator(db, utility.StrToBytes(comboPrefix))
	for iterator.Next() {
		key := utility.BytesToStr(iterator.Key)

		_, contains := ao.cachedStorage[key]
		if contains {
			continue
		}

		ao.cachedStorage[key] = iterator.Value

		if strings.HasPrefix(key, common.LockBNTKey) {
			source, name := ao.getSourceAndName(key[len(common.LockBNTKey)+1:])
			if 0 == len(source) {
				continue
			}
			target := ao.getOrCreateLockResource(result, source)
			target.Coin[name] = utility.BigIntBytesToStr(iterator.Value)
		}
		if strings.HasPrefix(key, common.LockFTKey) {
			source, name := ao.getSourceAndName(key[len(common.LockFTKey)+1:])
			if 0 == len(source) {
				continue
			}
			target := ao.getOrCreateLockResource(result, source)
			target.FT[name] = utility.BigIntBytesToStr(iterator.Value)
		}
		if strings.HasPrefix(key, common.LockNFTKey) {
			source, setId, id := ao.getSourceAndSetIdAndId(key[len(common.LockNFTKey)+1:])
			if 0 == len(source) {
				continue
			}
			target := ao.getOrCreateLockResource(result, source)
			nft := types.NFTID{
				SetId: setId,
				Id:    id,
			}
			target.NFT = append(target.NFT, nft)
		}
	}

	return result
}

func (ao *accountObject) getSourceAndName(mixed string) (source, name string) {
	list := strings.Split(mixed, ":")
	if 2 != len(list) {
		return "", ""
	}
	return list[0], list[1]
}

func (ao *accountObject) getSourceAndSetIdAndId(mixed string) (source, setId, id string) {
	list := strings.Split(mixed, ":")
	if 3 != len(list) {
		return "", "", ""
	}
	return list[0], list[1], list[2]
}

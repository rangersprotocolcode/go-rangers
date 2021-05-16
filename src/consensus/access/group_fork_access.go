package access

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/middleware/types"
	"encoding/hex"
	"fmt"
)

//SelectNextGroupFromChain
// SelectNextGroupFromChain determines the next verification group through the chained work-groups according to the previous random number.
// The result is random and certain, and mostly should be the same as method SelectNextGroupFromCache
//
// This method can be used to compensate when the result of the calculation through the cache(method SelectNextGroupFromCache)
// does not match the expectation
func (groupAccessor *GroupAccessor) SelectVerifyGroupFromFork(hash common.Hash, height uint64) (*types.Group, error) {
	quaulifiedGS := groupAccessor.getCastQualifiedGroupFromFork(height)
	idshort := make([]string, len(quaulifiedGS))
	for idx, g := range quaulifiedGS {
		idshort[idx] = groupsig.DeserializeID(g.Id).ShortS()
	}

	var group *types.Group
	if hash.Big().BitLen() > 0 && len(quaulifiedGS) > 0 {
		index := groupAccessor.selectIndex(len(quaulifiedGS), hash)
		group = quaulifiedGS[index]
		logger.Debugf("SelectVerifyGroupFroFork! Height:%v,qualified groups %v, index %v\n", height, idshort, index)
		return group, nil
	}
	return group, fmt.Errorf("SelectVerifyGroupFroFork failed, arg error")
}

func (groupAccessor *GroupAccessor) getCastQualifiedGroupFromFork(height uint64) []*types.Group {
	iter := groupAccessor.chain.ForkIterator()
	groups := make([]*types.Group, 0)
	for g := iter.Current(); g != nil; g = iter.MovePre() {
		if isGroupWorkQualifiedAt(g.Header, height) {
			groups = append(groups, g)
		} else if isGroupDissmisedAt(g.Header, height) {
			g = groupAccessor.chain.GetGroupByHeight(0)
			groups = append(groups, g)
			break
		}
	}
	logger.Debugf("GetCastQualifiedGroupFromChains height:%d", height)
	n := len(groups)
	reverseGroups := make([]*types.Group, n)
	for i := 0; i < n; i++ {
		reverseGroups[n-i-1] = groups[i]
		logger.Debugf("GetCastQualifiedGroupFromChains group id:%s", hex.EncodeToString(groups[i].Id))
	}
	return reverseGroups
}

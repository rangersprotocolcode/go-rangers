package access

import (
	"sync"

	"x/src/common"
	"x/src/consensus/groupsig"
	"github.com/hashicorp/golang-lru"
	"crypto/rand"
	"encoding/json"
	"os"
	"x/src/middleware/db"
	"x/src/consensus/model"
	"io/ioutil"
)

// key suffix definition when store the group infos to db
const (
	suffixSignKey = "_signKey"
	suffixGInfo   = "_gInfo"
)

//BelongGroups
// BelongGroups stores all group-related infos which is important to the members
type JoinedGroupStorage struct {
	privateKey common.PrivateKey //加密用的

	cache *lru.Cache
	db    *db.LDBDatabase

	storeDir  string
	initMutex sync.Mutex
}

//NewBelongGroups
func NewJoinedGroupStorage(filePath string, privateKey common.PrivateKey) *JoinedGroupStorage {
	return &JoinedGroupStorage{
		privateKey: privateKey,
		storeDir:   filePath,
	}
}

func (storage *JoinedGroupStorage) AddJoinedGroupInfo(joinedGroupInfo *model.JoinedGroupInfo) {
	if !storage.ready() {
		storage.initStore()
	}
	logger.Debugf("Add joined group info: group id=%v", joinedGroupInfo.GroupID.ShortS())
	storage.cache.Add(joinedGroupInfo.GroupID.GetHexString(), joinedGroupInfo)
	storage.storeJoinedGroup(joinedGroupInfo)
}

func (storage *JoinedGroupStorage) GetJoinedGroupInfo(id groupsig.ID) *model.JoinedGroupInfo {
	if !storage.ready() {
		storage.initStore()
	}
	v, ok := storage.cache.Get(id.GetHexString())
	if ok {
		return v.(*model.JoinedGroupInfo)
	}
	jg := storage.load(id)
	if jg != nil {
		storage.cache.Add(jg.GroupID.GetHexString(), jg)
	}
	return jg
}

func (storage *JoinedGroupStorage) AddMemberSignPk(minerId groupsig.ID, groupId groupsig.ID, signPK groupsig.Pubkey) (*model.JoinedGroupInfo, bool) {
	if !storage.ready() {
		storage.initStore()
	}
	jg := storage.GetJoinedGroupInfo(groupId)
	if jg == nil {
		return nil, false
	}

	if _, ok := jg.GetMemberSignPK(minerId); !ok {
		jg.AddMemberSignPK(minerId, signPK)
		storage.saveGroupInfo(jg)
		return jg, true
	}
	return jg, false
}

//joinedGroup2DBIfConfigExists
func (storage *JoinedGroupStorage) JoinedGroupFromConfig(file string) bool {
	if !fileExists(file) || isDir(file) {
		return false
	}
	logger.Debugf("load joined Groups info from %v", file)
	data, err := ioutil.ReadFile(file)
	if err != nil {
		logger.Errorf("load file %v fail, err %v", file, err.Error())
		return false
	}
	var gs []*model.JoinedGroupInfo
	err = json.Unmarshal(data, &gs)
	if err != nil {
		logger.Errorf("unmarshal joined group storage store file %v fail, err %v", file, err.Error())
		return false
	}
	n := 0
	storage.initStore()
	for _, jg := range gs {
		if storage.GetJoinedGroupInfo(jg.GroupID) == nil {
			n++
			storage.AddJoinedGroupInfo(jg)
		}
	}
	logger.Debugf("joined group info from config size %v", n)
	return true
}

func (storage *JoinedGroupStorage) LeaveGroups(gids []groupsig.ID) {
	if !storage.ready() {
		return
	}
	for _, gid := range gids {
		storage.cache.Remove(gid.GetHexString())
		storage.db.Delete(gInfoSuffix(gid))
	}
}

func (storage *JoinedGroupStorage) Close() {
	if !storage.ready() {
		return
	}
	storage.cache = nil
	storage.db.Close()
}

//IsMinerGroup
// IsMinerGroup detecting whether a group is a miner's ingot group
// (a miner can participate in multiple groups)
func (storage *JoinedGroupStorage) BelongGroup(groupId groupsig.ID) bool {
	return storage.GetJoinedGroupInfo(groupId) != nil
}

// joinGroup join a group (a miner ID can join multiple groups)
//			gid : group ID (not dummy id)
//			sk: user's group member signature private key
func (storage *JoinedGroupStorage) JoinGroup(joinedGroupInfo *model.JoinedGroupInfo, selfMinerId groupsig.ID) {
	logger.Infof("(%v):join group,group id=%v...\n", selfMinerId.GetHexString(), joinedGroupInfo.GroupID.ShortS())
	if !storage.BelongGroup(joinedGroupInfo.GroupID) {
		storage.AddJoinedGroupInfo(joinedGroupInfo)
	}
	return
}

func (storage *JoinedGroupStorage) initStore() {
	storage.initMutex.Lock()
	defer storage.initMutex.Unlock()

	if storage.ready() {
		return
	}
	db, err := db.NewLDBDatabase(storage.storeDir, 1, 1)
	if err != nil {
		panic("newLDBDatabase fail, file=" + storage.storeDir + "err=" + err.Error())
		return
	}
	storage.cache = common.CreateLRUCache(30)
	storage.db = db
}

func (storage *JoinedGroupStorage) ready() bool {
	return storage.cache != nil && storage.db != nil
}

func (storage *JoinedGroupStorage) storeJoinedGroup(joinedGroupInfo *model.JoinedGroupInfo) {
	storage.saveSignSecKey(joinedGroupInfo)
	storage.saveGroupInfo(joinedGroupInfo)
}

func (storage *JoinedGroupStorage) saveSignSecKey(joinedGroupInfo *model.JoinedGroupInfo) {
	if !storage.ready() {
		return
	}
	pubKey := storage.privateKey.GetPubKey()
	ct, err := pubKey.Encrypt(rand.Reader, joinedGroupInfo.SignSecKey.Serialize())
	//logger.Debugf("saveSignSecKey data:%v,privateKey:%v",joinedGroupInfo.SignSecKey.Serialize(),storage.privateKey.GetHexString())

	if err != nil {
		logger.Errorf("encrypt signkey fail, err=%v", err.Error())
		return
	}
	storage.db.Put(signKeySuffix(joinedGroupInfo.GroupID), ct)
}

func (storage *JoinedGroupStorage) saveGroupInfo(joinedGroupInfo *model.JoinedGroupInfo) {
	if !storage.ready() {
		return
	}
	st := joinedGroupDO{
		GroupID:             joinedGroupInfo.GroupID,
		GroupPK:             joinedGroupInfo.GroupPK,
		MemberSignPubkeyMap: joinedGroupInfo.GetMemberPKs(),
	}
	bs, err := json.Marshal(st)
	if err != nil {
		logger.Errorf("marshal joinedGroupDO fail, err=%v", err)
	} else {
		storage.db.Put(gInfoSuffix(joinedGroupInfo.GroupID), bs)
	}
}

func (storage *JoinedGroupStorage) load(gid groupsig.ID) *model.JoinedGroupInfo {
	if !storage.ready() {
		return nil
	}
	joinedGroupInfo := new(model.JoinedGroupInfo)
	joinedGroupInfo.MemberSignPubkeyMap = make(map[string]groupsig.Pubkey, 0)
	// Load signature private key
	bs, err := storage.db.Get(signKeySuffix(gid))
	if err != nil {
		logger.Errorf("get signKey fail, gid=%v, err=%v", gid.ShortS(), err.Error())
		return nil
	}
	//logger.Debugf("load bs:%v,privateKey:%v",bs,storage.privateKey.GetHexString())
	m, err := storage.privateKey.Decrypt(rand.Reader, bs)
	if err != nil {
		logger.Errorf("decrypt signKey fail, err=%v", err.Error())
		return nil
	}
	joinedGroupInfo.SignSecKey.Deserialize(m)

	// Load group information
	infoBytes, err := storage.db.Get(gInfoSuffix(gid))
	if err != nil {
		logger.Errorf("get groupInfo fail, gid=%v, err=%v", gid.ShortS(), err.Error())
		return joinedGroupInfo
	}
	if err := json.Unmarshal(infoBytes, joinedGroupInfo); err != nil {
		logger.Errorf("unmarshal groupInfo fail, gid=%v, err=%v", gid.ShortS(), err.Error())
		return joinedGroupInfo
	}
	return joinedGroupInfo
}

type joinedGroupDO struct {
	GroupID             groupsig.ID                // Group ID
	GroupPK             groupsig.Pubkey            // Group public key (backup, which can be taken from the global group)
	MemberSignPubkeyMap map[string]groupsig.Pubkey // Group member signature public key
}

func signKeySuffix(gid groupsig.ID) []byte {
	return []byte(gid.GetHexString() + suffixSignKey)
}

func gInfoSuffix(gid groupsig.ID) []byte {
	return []byte(gid.GetHexString() + suffixGInfo)
}

func fileExists(f string) bool {
	// os.Stat gets file information
	_, err := os.Stat(f)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}
func isDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

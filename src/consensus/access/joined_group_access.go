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

var joinedGroupStorageInstance *JoinedGroupStorage
//BelongGroups
// BelongGroups stores all group-related infos which is important to the members
type JoinedGroupStorage struct {
	privateKey common.PrivateKey //加密用的

	cache *lru.Cache
	db    *db.LDBDatabase

	storeDir  string
	initMutex sync.Mutex
}

func GetJoinedGroupStorageInstance() *JoinedGroupStorage {
	return joinedGroupStorageInstance
}

//NewBelongGroups
func InitJoinedGroupStorage(filePath string, privateKey common.PrivateKey) {
	if joinedGroupStorageInstance == nil {
		joinedGroupStorageInstance = &JoinedGroupStorage{
			privateKey: privateKey,
			storeDir:   filePath,
		}
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
	logger.Infof("(%v):join group,group idd=%v...\n", selfMinerId, joinedGroupInfo.GroupID.ShortS())
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

//type joinedGroupStore struct {
//	GroupID groupsig.ID        // Group ID
//	GroupPK groupsig.Pubkey    // Group public key (backup, which can be taken from the global group)
//	Members groupsig.PubkeyMap // Group member signature public key
//}

//
//func signKeySuffix(gid groupsig.ID) []byte {
//	return []byte(gid.GetHexString() + suffixSignKey)
//}
//
//func gInfoSuffix(gid groupsig.ID) []byte {
//	return []byte(gid.GetHexString() + suffixGInfo)
//}
//
//// BelongGroups stores all group-related infos which is important to the members
//type BelongGroups struct {
//	cache    *lru.Cache
//	priKey   common.PrivateKey
//	dirty    int32
//	store    *tasdb.LDBDatabase
//	storeDir string
//	initMu   sync.Mutex
//}
//
//func NewBelongGroups(file string, priKey common.PrivateKey) *BelongGroups {
//	return &BelongGroups{
//		dirty:    0,
//		priKey:   priKey,
//		storeDir: file,
//	}
//}
//
//func (bg *BelongGroups) initStore() {
//	bg.initMu.Lock()
//	defer bg.initMu.Unlock()
//
//	if bg.ready() {
//		return
//	}
//	db, err := tasdb.NewLDBDatabase(bg.storeDir, 1, 1)
//	if err != nil {
//		stdLogger.Errorf("newLDBDatabase fail, file=%v, err=%v\n", bg.storeDir, err.Error())
//		return
//	}
//
//	bg.store = db
//	bg.cache = common.MustNewLRUCache(30)
//}
//
//func (bg *BelongGroups) ready() bool {
//	return bg.cache != nil && bg.store != nil
//}
//
//func (bg *BelongGroups) storeSignKey(jg *JoinedGroup) {
//	if !bg.ready() {
//		return
//	}
//	pubKey := bg.priKey.GetPubKey()
//	ct, err := pubKey.Encrypt(rand.Reader, jg.SignKey.Serialize())
//	if err != nil {
//		stdLogger.Errorf("encrypt signkey fail, err=%v", err.Error())
//		return
//	}
//	bg.store.Put(signKeySuffix(jg.GroupID), ct)
//}
//
//func (bg *BelongGroups) storeGroupInfo(jg *JoinedGroup) {
//	if !bg.ready() {
//		return
//	}
//	st := joinedGroupStore{
//		GroupID: jg.GroupID,
//		GroupPK: jg.GroupPK,
//		Members: jg.getMemberMap(),
//	}
//	bs, err := json.Marshal(st)
//	if err != nil {
//		stdLogger.Errorf("marshal joinedGroup fail, err=%v", err)
//	} else {
//		bg.store.Put(gInfoSuffix(jg.GroupID), bs)
//	}
//}
//
//func (bg *BelongGroups) storeJoinedGroup(jg *JoinedGroup) {
//	bg.storeSignKey(jg)
//	bg.storeGroupInfo(jg)
//}
//
//func (bg *BelongGroups) loadJoinedGroup(gid groupsig.ID) *JoinedGroup {
//	if !bg.ready() {
//		return nil
//	}
//	jg := new(JoinedGroup)
//	jg.Members = make(groupsig.PubkeyMap, 0)
//	// Load signature private key
//	bs, err := bg.store.Get(signKeySuffix(gid))
//	if err != nil {
//		stdLogger.Errorf("get signKey fail, gid=%v, err=%v", gid.ShortS(), err.Error())
//		return nil
//	}
//	m, err := bg.priKey.Decrypt(rand.Reader, bs)
//	if err != nil {
//		stdLogger.Errorf("decrypt signKey fail, err=%v", err.Error())
//		return nil
//	}
//	jg.SignKey.Deserialize(m)
//
//	// Load group information
//	infoBytes, err := bg.store.Get(gInfoSuffix(gid))
//	if err != nil {
//		stdLogger.Errorf("get gInfo fail, gid=%v, err=%v", gid.ShortS(), err.Error())
//		return jg
//	}
//	if err := json.Unmarshal(infoBytes, jg); err != nil {
//		stdLogger.Errorf("unmarsal gInfo fail, gid=%v, err=%v", gid.ShortS(), err.Error())
//		return jg
//	}
//	return jg
//}
//
//func fileExists(f string) bool {
//	// os.Stat gets file information
//	_, err := os.Stat(f)
//	if err != nil {
//		if os.IsExist(err) {
//			return true
//		}
//		return false
//	}
//	return true
//}
//func isDir(path string) bool {
//	s, err := os.Stat(path)
//	if err != nil {
//		return false
//	}
//	return s.IsDir()
//}
//
//func (bg *BelongGroups) joinedGroup2DBIfConfigExists(file string) bool {
//	if !fileExists(file) || isDir(file) {
//		return false
//	}
//	stdLogger.Debugf("load belongGroups from %v", file)
//	data, err := ioutil.ReadFile(file)
//	if err != nil {
//		stdLogger.Errorf("load file %v fail, err %v", file, err.Error())
//		return false
//	}
//	var gs []*JoinedGroup
//	err = json.Unmarshal(data, &gs)
//	if err != nil {
//		stdLogger.Errorf("unmarshal belongGroup store file %v fail, err %v", file, err.Error())
//		return false
//	}
//	n := 0
//	bg.initStore()
//	for _, jg := range gs {
//		if bg.getJoinedGroup(jg.GroupID) == nil {
//			n++
//			bg.addJoinedGroup(jg)
//		}
//	}
//	stdLogger.Debugf("joinedGroup2DBIfConfigExists belongGroups size %v", n)
//	return true
//}
//
//func (bg *BelongGroups) getJoinedGroup(id groupsig.ID) *JoinedGroup {
//	if !bg.ready() {
//		bg.initStore()
//	}
//	v, ok := bg.cache.Get(id.GetHexString())
//	if ok {
//		return v.(*JoinedGroup)
//	}
//	jg := bg.loadJoinedGroup(id)
//	if jg != nil {
//		bg.cache.Add(jg.GroupID.GetHexString(), jg)
//	}
//	return jg
//}
//
//func (bg *BelongGroups) addMemSignPk(uid groupsig.ID, gid groupsig.ID, signPK groupsig.Pubkey) (*JoinedGroup, bool) {
//	if !bg.ready() {
//		bg.initStore()
//	}
//	jg := bg.getJoinedGroup(gid)
//	if jg == nil {
//		return nil, false
//	}
//
//	if _, ok := jg.getMemSignPK(uid); !ok {
//		jg.addMemSignPK(uid, signPK)
//		bg.storeGroupInfo(jg)
//		return jg, true
//	}
//	return jg, false
//}
//
//func (bg *BelongGroups) addJoinedGroup(jg *JoinedGroup) {
//	if !bg.ready() {
//		bg.initStore()
//	}
//	newBizLog("addJoinedGroup").debug("add gid=%v", jg.GroupID.ShortS())
//	bg.cache.Add(jg.GroupID.GetHexString(), jg)
//	bg.storeJoinedGroup(jg)
//}
//
//func (bg *BelongGroups) leaveGroups(gids []groupsig.ID) {
//	if !bg.ready() {
//		return
//	}
//	for _, gid := range gids {
//		bg.cache.Remove(gid.GetHexString())
//		bg.store.Delete(gInfoSuffix(gid))
//	}
//}
//
//func (bg *BelongGroups) close() {
//	if !bg.ready() {
//		return
//	}
//	bg.cache = nil
//	bg.store.Close()
//}
//
//func (p *Processor) genBelongGroupStoreFile() string {
//	storeFile := p.conf.GetString(ConsensusConfSection, "groupstore", "")
//	if strings.TrimSpace(storeFile) == "" {
//		storeFile = "groupstore" + p.conf.GetString("instance", "index", "")
//	}
//	return storeFile
//}
//
//// getMemberSignPubKey get the signature public key of the member in the group
//func (p Processor) getMemberSignPubKey(gmi *model.GroupMinerID) (pk groupsig.Pubkey, ok bool) {
//	if jg := p.belongGroups.getJoinedGroup(gmi.Gid); jg != nil {
//		pk, ok = jg.getMemSignPK(gmi.UID)
//		if !ok && !p.GetMinerID().IsEqual(gmi.UID) {
//			p.askSignPK(gmi)
//		}
//	}
//	return
//}
//
//// joinGroup join a group (a miner ID can join multiple groups)
////			gid : group ID (not dummy id)
////			sk: user's group member signature private key
//func (p *Processor) joinGroup(g *JoinedGroup) {
//	stdLogger.Infof("begin Processor(%v)::joinGroup, gid=%v...\n", p.getPrefix(), g.GroupID.ShortS())
//	if !p.IsMinerGroup(g.GroupID) {
//		p.belongGroups.addJoinedGroup(g)
//	}
//	return
//}
//
//// getSignKey get the signature private key of the miner in a certain group
//func (p Processor) getSignKey(gid groupsig.ID) groupsig.Seckey {
//	if jg := p.belongGroups.getJoinedGroup(gid); jg != nil {
//		return jg.SignKey
//	}
//	return groupsig.Seckey{}
//}
//
//// IsMinerGroup detecting whether a group is a miner's ingot group
//// (a miner can participate in multiple groups)
//func (p *Processor) IsMinerGroup(gid groupsig.ID) bool {
//	return p.belongGroups.getJoinedGroup(gid) != nil
//}
//
//func (p *Processor) askSignPK(gmi *model.GroupMinerID) {
//	if !addSignPkReq(gmi.UID) {
//		return
//	}
//	msg := &model.ConsensusSignPubkeyReqMessage{
//		GroupID: gmi.Gid,
//	}
//	ski := model.NewSecKeyInfo(p.GetMinerID(), p.mi.GetDefaultSecKey())
//	if msg.GenSign(ski, msg) {
//		newBizLog("AskSignPK").debug("ask sign pk message, receiver %v, gid %v", gmi.UID.ShortS(), gmi.Gid.ShortS())
//		p.NetServer.AskSignPkMessage(msg, gmi.UID)
//	}
//}

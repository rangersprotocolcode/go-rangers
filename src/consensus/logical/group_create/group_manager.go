package group_create







//
//func (gm *GroupManager) addGroupOnChain(sgi *StaticGroupInfo) {
//	group := convertStaticGroup2CoreGroup(sgi)
//
//	stdLogger.Infof("addGroupOnChain height:%d,id:%s\n", group.GroupHeight, sgi.GroupID.ShortS())
//
//	var err error
//	defer func() {
//		var s string
//		if err != nil {
//			s = err.Error()
//		}
//		newHashTraceLog("addGroupOnChain", sgi.GInfo.GroupHash(), groupsig.ID{}).log("gid=%v, workHeight=%v, result %v", sgi.GroupID.ShortS(), group.Header.WorkHeight, s)
//	}()
//
//	if gm.groupChain.GetGroupByID(group.ID) != nil {
//		stdLogger.Debugf("group already onchain, accept, id=%v\n", sgi.GroupID.ShortS())
//		gm.processor.acceptGroup(sgi)
//		err = fmt.Errorf("group already onchain")
//	} else {
//		top := gm.processor.MainChain.Height()
//		if !sgi.GetReadyTimeout(top) {
//			err1 := gm.groupChain.AddGroup(group)
//			if err1 != nil {
//				stdLogger.Errorf("ERROR:add group fail! hash=%v, gid=%v, err=%v\n", group.Header.Hash.ShortS(), sgi.GroupID.ShortS(), err1.Error())
//				err = err1
//				return
//			}
//			err = fmt.Errorf("success")
//			gm.checker.addHeightCreated(group.Header.CreateHeight)
//			stdLogger.Infof("addGroupOnChain success, ID=%v, height=%v\n", sgi.GroupID.ShortS(), gm.groupChain.Height())
//		} else {
//			err = fmt.Errorf("ready timeout, currentHeight %v", top)
//			stdLogger.Infof("addGroupOnChain group ready timeout, gid %v, timeout height %v, top %v\n", sgi.GroupID.ShortS(), sgi.GInfo.GI.GHeader.ReadyHeight, top)
//		}
//	}
//
//}


// GroupManager is responsible for group creation
//type GroupManager struct {
//	groupChain       *core.GroupChain
//	mainChain        core.BlockChain
//	processor        *Processor
//	creatingGroupCtx *CreatingGroupContext
//	checker          *GroupCreateChecker
//}
//
//func newGroupManager(processor *Processor) *GroupManager {
//	gm := &GroupManager{
//		processor:  processor,
//		mainChain:  processor.MainChain,
//		groupChain: processor.GroupChain,
//		checker:    newGroupCreateChecker(processor),
//	}
//	return gm
//}

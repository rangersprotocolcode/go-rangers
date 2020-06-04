package statemachine

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"x/src/utility"
)

// 获取当前状态机的存储状态
func (c *StateMachine) RefreshStorageStatus(requestId uint64) {
	c.RequestId = requestId
	if 0 == len(c.storagePath) {
		return
	}

	c.logger.Infof("start checkfiles, %s", c.Game)
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("%s RequestId: %d\n", c.Game, c.RequestId))
	for _, path := range c.storagePath {
		realPath := strings.Split(path, ":")
		pathMD5, _ := utility.CheckFolder(realPath[0])
		buffer.WriteString(fmt.Sprintf("%s, md5: %v\n", path, pathMD5))
		c.logger.Infof("checked: %s, md5: %v", realPath[0], pathMD5)
	}

	c.logger.Infof("end checkfiles, %s, detail: %s", c.Game, buffer.String())
	c.StorageStatus = md5.Sum(buffer.Bytes())
}

func (c *StateMachine) uploadStorage() string {
	zipFile := c.zipStorage()
	defer os.Remove(zipFile)

	// 上传
	if 0 != len(zipFile) {
		file, _ := os.Open(zipFile)
		defer file.Close()

		reader := bufio.NewReader(file)
		cid, err := c.ipfsShell.Add(reader)
		if err != nil {
			c.logger.Errorf("fail to add ipfs link, %s", zipFile)
			return ""
		}

		localID, err := c.ipfsShell.ID()
		if err != nil {
			c.logger.Errorf("fail to add ipfs link, %s", zipFile)
			return ""
		}
		addressList := localID.Addresses

		return fmt.Sprintf("%s:%s:%s", addressList[len(addressList)-1], cid, zipFile)
	}

	return ""
}

// 打包本地存储
func (c *StateMachine) zipStorage() string {
	zipFile := fmt.Sprintf("%s-%d-%s-%d.zip", c.Game, c.RequestId, hex.EncodeToString(c.StorageStatus[:]), utility.GetTime().UnixNano())
	err := utility.Zip(c.storageGame, zipFile)
	if err != nil {
		c.logger.Errorf("stm %s failed to zip storage, storageRoot: %s, zipFile: %s, err: %s", c.Game, c.storageRoot, zipFile, err.Error())
		return ""
	}

	c.logger.Infof("stm %s zipStorage, storageRoot: %s, zipFile: %s", c.Game, c.storageGame, zipFile)
	return zipFile
}

//更新本地存储
func (c *StateMachine) updateStorage(localID, cid, zipFile, requestId string) {
	defer os.Remove(zipFile)

	// 删除
	err := os.RemoveAll(c.storageGame)
	if err != nil {
		c.logger.Errorf("stm %s failed to remove storage, storageRoot: %s, err: %s", c.Game, c.storageRoot, err.Error())
		return
	} else {
		c.logger.Warnf("stm %s removed storage, storageRoot: %s", c.Game, c.storageRoot)
	}

	// 下载
	if !c.downloadStorage(localID, cid, zipFile) {
		c.logger.Errorf("stm %s failed to download storage: %s, storageRoot: %s, err: %s", c.Game, zipFile, c.storageRoot)
		return
	}

	//解压
	err = utility.Unzip(zipFile, c.storageRoot)
	if err != nil {
		c.logger.Errorf("stm %s failed to unzip storage, storageRoot: %s, err: %s", c.Game, c.storageRoot, err.Error())
		return
	}

	nonce, _ := strconv.Atoi(requestId)
	c.RequestId = uint64(nonce)

	c.logger.Infof("stm %s updated storage successful, storageRoot: %s, zipFile: %s, requestId: %s", c.Game, c.storageRoot, zipFile, requestId)
}

func (c *StateMachine) downloadStorage(localID, cid, zipFile string) bool {
	err := c.ipfsShell.SwarmConnect(context.Background(), localID)
	if err != nil {
		c.logger.Errorf("fail to download Storage, error: %s, appId: %s", err, c.Game)
		return false
	}
	c.logger.Debugf("connect ok %s, %s, %s", localID, cid, zipFile)

	err = c.ipfsShell.Get(cid, zipFile)
	if err != nil {
		c.logger.Errorf("fail to download Storage, error: %s, appId: %s", err, c.Game)
		return false
	}

	c.logger.Debugf("got file %s, %s, %s", localID, cid, zipFile)
	return true
}

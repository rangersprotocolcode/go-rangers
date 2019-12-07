package statemachine

import (
	"crypto/md5"
	"x/src/utility"
	"bytes"
	"fmt"
	"strings"
	"os"
	"time"
	"github.com/ipfs/go-ipfs-api"
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

func (c *StateMachine) UploadStorage() string {
	zipFile := c.zipStorage()
	defer os.Remove(zipFile)

	// 上传
	if 0 != len(zipFile) && c.uploadStorage(zipFile) {
		return zipFile
	}
	return ""
}

// 打包本地存储
func (c *StateMachine) zipStorage() string {
	zipFile := fmt.Sprintf("%s-%d-%d.zip", c.Game, c.RequestId, time.Now().UnixNano())
	err := utility.Zip(c.storageGame, zipFile)
	if err != nil {
		c.logger.Errorf("stm %s failed to zip storage, storageRoot: %s, zipFile: %s, err: %s", c.Game, c.storageRoot, zipFile, err.Error())
		return ""
	}

	c.logger.Infof("stm %s zipStorage, storageRoot: %s, zipFile: %s", c.Game, c.storageGame, zipFile)
	return zipFile
}

//更新本地存储
func (c *StateMachine) updateStorage(zipFile string) {
	// 删除
	err := os.RemoveAll(c.storageGame)
	if err != nil {
		c.logger.Errorf("stm %s failed to remove storage, storageRoot: %s, err: %s", c.Game, c.storageRoot, err.Error())
		return
	}

	// 下载
	if c.downloadStorage(zipFile) {
		c.logger.Errorf("stm %s failed to download storage: %s, storageRoot: %s, err: %s", c.Game, zipFile, c.storageRoot, err.Error())
		return
	}

	//解压
	err = utility.Unzip(zipFile, c.storageRoot)
	if err != nil {
		c.logger.Errorf("stm %s failed to unzip storage, storageRoot: %s, err: %s", c.Game, c.storageRoot, err.Error())
		return
	}

	c.logger.Infof("stm %s update storage, storageRoot: %s, zipFile: %s", c.Game, c.storageRoot, zipFile)
}

func (c *StateMachine) downloadStorage(zipFile string) bool {
	return true
}

func (c *StateMachine) uploadStorage(zipFile string) bool {
	shell.NewShell("localhost:5001")
	//localID, err := sh.ID()
	return true
}

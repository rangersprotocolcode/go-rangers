package statemachine

import (
	"crypto/md5"
	"x/src/utility"
	"bytes"
	"fmt"
)

// 获取当前状态机的存储状态
func (c *StateMachine) RefreshStorageStatus(requestId uint64) {
	c.RequestId = requestId
	if 0 == len(c.storagePath) {
		return
	}

	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("%s nonce: %d\n", c.Game, c.RequestId))
	for _, path := range c.storagePath {
		pathMD5, _ := utility.CheckFolder(path)
		buffer.WriteString(fmt.Sprintf("%s, md5: %v\n", path, pathMD5))
	}

	c.logger.Info(buffer.String())
	c.StorageStatus = md5.Sum(buffer.Bytes())
}

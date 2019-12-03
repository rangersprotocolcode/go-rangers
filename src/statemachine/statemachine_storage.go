package statemachine

import (
	"crypto/md5"
	"x/src/utility"
	"bytes"
	"fmt"
	//"github.com/aliyun/aliyun-oss-go-sdk/oss"
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

func (c *StateMachine) getStoragePathRoot() string {
	return fmt.Sprintf("%s", c.Game)
}

//
//func (c *StateMachine) UploadStorageStatus() {
//	destZip := c.generateDestZip()
//	err := utility.Zip(c.getStoragePathRoot(), destZip)
//	if err != nil {
//		c.logger.Errorf("fail to generate storage. %s", err)
//		return
//	}
//
//	bucket := c.getOSSBucket()
//	if nil == bucket {
//		return
//	}
//
//	err = bucket.PutObjectFromFile(destZip, destZip)
//	if err != nil {
//		c.logger.Errorf("fail to generate storage. %s", err)
//		return
//	}
//
//	//msg := network.Message{Code: network.STMStorageReady, Body: body}
//	//network.GetNetInstance().Broadcast(msg)
//}
//
//func (c *StateMachine) DownloadStorage() {
//	bucket := c.getOSSBucket()
//	if nil == bucket {
//		return
//	}
//	//err := bucket.GetObjectToFile(objectName, downloadedFileName)
//	//if err != nil {
//	//	c.logger.Errorf("fail to generate storage. %s", err)
//	//	return
//	//}
//}
//

//func (c *StateMachine) generateDestZip() string {
//	return fmt.Sprintf("./%s-%d.zip", c.Game, c.RequestId)
//}
//func (c *StateMachine) getOSSBucket() *oss.Bucket {
//	// Endpoint以杭州为例，其它Region请按实际情况填写。
//	endpoint := "oss-accelerate.aliyuncs.com"
//	// 阿里云主账号AccessKey拥有所有API的访问权限，风险很高。强烈建议您创建并使用RAM账号进行API访问或日常运维，请登录 https://ram.console.aliyun.com 创建RAM账号。
//	accessKeyId := "LTAI4FgUH1VQBohuGnG2qW22"
//	accessKeySecret := "bpXd7mnNYwN9zus4zXxXGclImiCpeI"
//	bucketName := "rocket-protocol-stm-file-dev"
//	// 创建OSSClient实例
//	client, err := oss.New(endpoint, accessKeyId, accessKeySecret)
//	if err != nil {
//		c.logger.Errorf("fail to generate storage. %s", err)
//		return nil
//	}
//	// 获取存储空间。
//	bucket, err := client.Bucket(bucketName)
//	if err != nil {
//		c.logger.Errorf("fail to generate storage. %s", err)
//		return nil
//	}
//
//	return bucket
//}

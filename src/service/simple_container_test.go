package service

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware/log"
	"com.tuntun.rangers/node/src/middleware/types"
	"encoding/json"
	"strconv"
	"testing"
	"time"
)

func TestSimpleContainer(t *testing.T) {
	txPoolLogger = log.GetLoggerByIndex(log.TxPoolLogConfig, strconv.Itoa(common.InstanceIndex))

	container := newSimpleContainer(100)

	hash1 := common.HexToHash("0xfa6be88da0fd27716900859a633c9085d664b4e07053f2c1361d6a5d6f911111")
	source1 := "11111"
	tx1 := &types.Transaction{Hash: hash1, Source: source1}

	hash2 := common.HexToHash("0xfa6be88da0fd27716900859a633c9085d664b4e07053f2c1361d6a5d6f922222")
	source2 := "22222"
	tx2 := &types.Transaction{Hash: hash2, Source: source2}

	container.push(tx1)
	container.push(tx2)
	//assert.Equal(t, container.Len(), 2)

	//assert.Equal(t, true, container.contains(hash1))
	//assert.Equal(t, true, container.contains(hash2))

	//gotTx1 := container.get(hash1)
	//assert.Equal(t, source1, gotTx1.Source)
	txPoolLogger.Debugf("after push tx1 and push tx2")
	container.dump()

	hash3 := common.HexToHash("0xfa6be88da0fd27716900859a633c9085d664b4e07053f2c1361d6a5d6f933333")
	source3 := "33333"
	tx3 := &types.Transaction{Hash: hash3, Source: source3}
	container.push(tx3)
	txPoolLogger.Debugf("after push tx3")
	container.dump()

	txHashList := make([]interface{}, 0)
	txHashList = append(txHashList, hash2)
	container.batchRemove(txHashList)
	txPoolLogger.Debugf("after remove tx2")
	container.dump()

	time.Sleep(time.Second * 30)
	hash4 := common.HexToHash("0xfa6be88da0fd27716900859a633c9085d664b4e07053f2c1361d6a5d6f944444")
	source4 := "44444"
	tx4 := &types.Transaction{Hash: hash4, Source: source4}
	container.push(tx4)
	txPoolLogger.Debugf("after push tx4")
	container.dump()

	time.Sleep(time.Minute * 5)
}

func (c *simpleContainer) dump() {
	txPoolLogger.Debugf("data len:%d", c.data.Size())
	data := c.asSlice()
	for _, item := range data {
		txPoolLogger.Debugf("hash:%s", item.(*types.Transaction).Hash.String())
	}

	txPoolLogger.Debugf("annual ring map:")

	dbData, _ := c.db.Get(pendingTxListKey)
	var pendingTxList []*types.Transaction
	json.Unmarshal(dbData, &pendingTxList)
	txPoolLogger.Debugf("db data len:%d", len(pendingTxList))
	for _, item := range pendingTxList {
		txPoolLogger.Debugf("hash:%s", item.Hash.String())
	}
	txPoolLogger.Debugf("\n")
}

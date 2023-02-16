package middleware

import "com.tuntun.rocket/node/src/middleware/notify"

const maxWriteSize = 100000

var DataChannel dataChannel

type dataChannel struct {
	// 客户端的writer ws请求，从tx的数据库过来
	rcvedTx chan *notify.ClientTransactionMessage
}

func InitDataChannel() {
	DataChannel = dataChannel{}
	DataChannel.rcvedTx = make(chan *notify.ClientTransactionMessage, maxWriteSize)
}

func (channel dataChannel) GetRcvedTx() chan *notify.ClientTransactionMessage {
	return channel.rcvedTx
}

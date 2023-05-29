package middleware

import "com.tuntun.rocket/node/src/middleware/notify"

const maxWriteSize = 1000

var DataChannel dataChannel

type dataChannel struct {
	rcvedTx chan *notify.ClientTransactionMessage
}

func InitDataChannel() {
	DataChannel = dataChannel{}
	DataChannel.rcvedTx = make(chan *notify.ClientTransactionMessage, maxWriteSize)
}

func (channel dataChannel) GetRcvedTx() chan *notify.ClientTransactionMessage {
	return channel.rcvedTx
}

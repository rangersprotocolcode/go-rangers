package middleware

import "com.tuntun.rocket/node/src/middleware/notify"

const maxWriteSize = 100000

var DataChannel dataChannel

type dataChannel struct {
	RcvedTx chan *notify.ClientTransactionMessage
}

func InitDataChannel() {
	DataChannel = dataChannel{}
	DataChannel.RcvedTx = make(chan *notify.ClientTransactionMessage, maxWriteSize)
}

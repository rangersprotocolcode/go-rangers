package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/notify"
	"com.tuntun.rocket/node/src/middleware/types"
)

func (chain *blockChain) notifyLogs(blockHash common.Hash, receipts types.Receipts) {
	if nil == receipts || 0 == len(receipts) {
		return
	}

	for _, receipt := range receipts {
		if receipt.Logs != nil && len(receipt.Logs) != 0 {
			logs := make([]*types.Log, 0)
			for _, log := range receipt.Logs {
				log.TxHash = receipt.TxHash
				log.BlockHash = blockHash
				logs = append(logs, log)
			}
			msg := notify.VMEventNotifyMessage{Logs: logs}
			notify.BUS.Publish(notify.VMEventNotify, &msg)
			logger.Debugf("Vm event notify publish:%v", msg.Logs)
		}
	}
}

func (chain *blockChain) notifyBlockHeader(header *types.BlockHeader) {
	if nil == header {
		return
	}
	msg := notify.BlockHeaderNotifyMessage{BlockHeader: header}
	notify.BUS.Publish(notify.BlockHeaderNotify, &msg)
	logger.Debugf("new block header notify publish:%v", msg.BlockHeader)
}

func (chain *blockChain) notifyRemovedLogs(receipts types.Receipts) {
	if nil == receipts || 0 == len(receipts) {
		return
	}

	for _, receipt := range receipts {
		if receipt.Logs != nil && len(receipt.Logs) != 0 {
			logs := make([]*types.Log, 0)
			for _, log := range receipt.Logs {
				log.Removed = true
				logs = append(logs, log)
			}

			msg := notify.VMRemovedEventNotifyMessage{Logs: logs}
			notify.BUS.Publish(notify.VMRemovedEventNotify, &msg)
			logger.Debugf("Vm removed event notify publish:%v", msg.Logs)
		}
	}
}

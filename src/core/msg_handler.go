package core

import (
	"math/big"

	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/notify"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/network"
	"com.tuntun.rocket/node/src/utility"

	"com.tuntun.rocket/node/src/middleware"
	"com.tuntun.rocket/node/src/middleware/pb"
	"com.tuntun.rocket/node/src/service"
	"github.com/golang/protobuf/proto"
	"time"
)

const blockResponseSize = 1

type ChainHandler struct{}

func initChainHandler() {
	handler := ChainHandler{}

	notify.BUS.Subscribe(notify.BlockReq, handler.blockReqHandler)
	notify.BUS.Subscribe(notify.NewBlock, handler.newBlockHandler)
	notify.BUS.Subscribe(notify.TransactionReq, handler.transactionReqHandler)
	notify.BUS.Subscribe(notify.TransactionGot, handler.transactionGotHandler)
}

func (c *ChainHandler) Handle(sourceId string, msg network.Message) error {
	return nil
}

func (ch ChainHandler) transactionReqHandler(msg notify.Message) {
	trm, ok := msg.(*notify.TransactionReqMessage)
	if !ok {
		logger.Debugf("transactionReqHandler:Message assert not ok!")
		return
	}
	m, e := unMarshalTransactionRequestMessage(trm.TransactionReqByte)
	if e != nil {
		logger.Errorf("unmarshal transaction request message error:%s", e.Error())
		return
	}

	source := trm.Peer
	logger.Debugf("receive transaction req from %s,block height:%d,block hash:%s,tx_len%d", source, m.BlockHeight, m.CurrentBlockHash.String(), len(m.TransactionHashes))
	if nil == blockChainImpl {
		return
	}
	transactions, need, _, e := blockChainImpl.queryTxsByBlockHash(m.CurrentBlockHash, m.TransactionHashes)
	if e == service.ErrNil {
		m.TransactionHashes = need
	}

	for _, tx := range transactions {
		logger.Debugf("local find tx :%s,%v", tx.Hash.String(), tx)
	}
	if nil != transactions && 0 != len(transactions) {
		sendTransactions(transactions, source)
	}
	return
}

func (ch ChainHandler) transactionGotHandler(msg notify.Message) {
	tgm, ok := msg.(*notify.TransactionGotMessage)
	if !ok {
		logger.Debugf("transactionGotHandler:Message assert not ok!")
		return
	}

	txs, e := types.UnMarshalTransactions(tgm.TransactionGotByte)
	if e != nil {
		logger.Errorf("Unmarshal got transactions error:%s", e.Error())
		return
	}
	service.GetTransactionPool().AddMissTransactions(txs)

	m := notify.TransactionGotAddSuccMessage{Transactions: txs, Peer: tgm.Peer}
	notify.BUS.Publish(notify.TransactionGotAddSucc, &m)
	return
}

func (ch ChainHandler) blockReqHandler(msg notify.Message) {

	m, ok := msg.(*notify.BlockReqMessage)
	if !ok {
		logger.Debugf("blockReqHandler:Message assert not ok!")
		return
	}
	reqHeight := utility.ByteToUInt64(m.HeightByte)
	localHeight := blockChainImpl.Height()

	logger.Debugf("Rcv block request:reqHeight:%d,localHeight:%d", reqHeight, localHeight)
	var count = 0
	for i := reqHeight; i <= localHeight; i++ {
		block := blockChainImpl.QueryBlock(i)
		if block == nil {
			continue
		}
		count++
		if count == blockResponseSize || i == localHeight {
			sendBlock(m.Peer, block, true)
		} else {
			sendBlock(m.Peer, block, false)
		}
		if count >= blockResponseSize {
			break
		}
	}
	if count == 0 {
		sendBlock(m.Peer, nil, true)
	}
}

func (ch ChainHandler) newBlockHandler(msg notify.Message) {
	m, ok := msg.(*notify.NewBlockMessage)
	if !ok {
		return
	}
	source := m.Peer
	block, e := types.UnMarshalBlock(m.BlockByte)
	if e != nil {
		logger.Debugf("UnMarshal block error:%d", e.Error())
		return
	}

	middleware.PerfLogger.Debugf("Rcv new block from %s,hash:%v,height:%d,totalQn:%d,tx len:%d, total cost: %v", source, block.Header.Hash.Hex(), block.Header.Height, block.Header.TotalQN, len(block.Transactions), time.Since(block.Header.CurTime))

	blockChainImpl.AddBlockOnChain(source, block, types.NewBlock)
}

func unMarshalTransactionRequestMessage(b []byte) (*transactionRequestMessage, error) {
	m := new(middleware_pb.TransactionRequestMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		network.Logger.Errorf("UnMarshal transaction request message error:%s", e.Error())
		return nil, e
	}

	txHashes := make([]common.Hashes, 0)
	for _, txHash := range m.TransactionHashes {
		hashes := common.Hashes{}
		hashes[0] = common.BytesToHash(txHash.Hash)
		hashes[1] = common.BytesToHash(txHash.SubHash)

		txHashes = append(txHashes, hashes)
	}

	currentBlockHash := common.BytesToHash(m.CurrentBlockHash)
	blockPv := &big.Int{}
	blockPv.SetBytes(m.BlockPv)
	message := transactionRequestMessage{TransactionHashes: txHashes, CurrentBlockHash: currentBlockHash, BlockHeight: *m.BlockHeight, BlockPv: blockPv}
	return &message, nil
}

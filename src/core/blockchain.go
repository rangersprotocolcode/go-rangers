package core

import (
	"x/src/common"
	"fmt"
	"github.com/hashicorp/golang-lru"
	"x/src/middleware/types"
	"math/big"
	"os"
	"x/src/middleware/db"

	"time"
	"x/src/utility"
	"x/src/middleware"
	"x/src/storage/account"
	"encoding/binary"
	"math"
	"x/src/service"
)

const (
	latestBlockKey = "bcurrent"

	addBlockMark    = "addBlockMark"
	removeBlockMark = "removeBlockMark"

	chainPieceLength      = 9
	chainPieceBlockLength = 6

	hashDBPrefix       = "block"
	heightDBPrefix     = "height"
	verifyHashDBPrefix = "verifyHash"

	topBlocksCacheSize = 100
)

var blockChainImpl *blockChain

type blockChain struct {
	init bool

	latestBlock *types.BlockHeader
	requestIds  map[string]uint64

	topBlocks         *lru.Cache // key：块高，value：header
	futureBlocks      *lru.Cache // key：块hash，value：block（体积很大）
	verifiedBlocks    *lru.Cache // key：块hash，value：receipts(体积大) &stateroot
	verifiedBodyCache *lru.Cache // key：块hash，value：块对应的transaction(体积大)

	hashDB       db.Database
	heightDB     db.Database
	verifyHashDB db.Database

	executor        *VMExecutor
	forkProcessor   *forkProcessor
	transactionPool service.TransactionPool

	lock middleware.Loglock
}

type castingBlock struct {
	state    *account.AccountDB
	receipts types.Receipts
}

func initBlockChain() error {
	chain := &blockChain{lock: middleware.NewLoglock("chain")}
	chain.transactionPool = service.GetTransactionPool()

	var err error
	chain.topBlocks, err = lru.New(100)
	if err != nil {
		logger.Errorf("Init cache error:%s", err.Error())
		return err
	}

	chain.futureBlocks, err = lru.New(topBlocksCacheSize)
	if err != nil {
		logger.Errorf("Init cache error:%s", err.Error())
		return err
	}

	chain.verifiedBlocks, err = lru.New(10)
	if err != nil {
		logger.Errorf("Init cache error:%s", err.Error())
		return err
	}

	chain.verifiedBodyCache, err = lru.New(10)
	if err != nil {
		logger.Errorf("Init cache error:%s", err.Error())
		return err
	}

	chain.hashDB, err = db.NewDatabase(hashDBPrefix)
	if err != nil {
		logger.Debugf("Init block chain error! Error:%s", err.Error())
		return err
	}
	chain.heightDB, err = db.NewDatabase(heightDBPrefix)
	if err != nil {
		logger.Debugf("Init block chain error! Error:%s", err.Error())
		return err
	}
	chain.verifyHashDB, err = db.NewDatabase(verifyHashDBPrefix)
	if err != nil {
		logger.Debugf("Init block chain error! Error:%s", err.Error())
		return err
	}

	chain.executor = NewVMExecutor(chain)
	chain.forkProcessor = initForkProcessor(chain)

	initMinerManager()

	chain.latestBlock = chain.queryBlockHeaderByHeight([]byte(latestBlockKey), false)
	if chain.latestBlock == nil {
		chain.insertGenesisBlock()
	} else {
		chain.ensureChainConsistency()

		state, err := service.AccountDBManagerInstance.GetAccountDBByHash(chain.latestBlock.StateTree)
		if nil != err {
			panic(err)
		}
		service.AccountDBManagerInstance.SetLatestStateDB(state)
		logger.Debugf("refreshed latestStateDB, state: %v, height: %d", chain.latestBlock.StateTree, chain.latestBlock.Height)

		if !chain.versionValidate() {
			fmt.Println("Illegal data version! Please delete the directory d0 and restart the program!")
			os.Exit(0)
		}
		chain.buildCache(topBlocksCacheSize)
	}
	chain.init = true
	blockChainImpl = chain

	return nil
}

func (chain *blockChain) CastBlock(timestamp time.Time, height uint64, proveValue *big.Int, proveRoot common.Hash, qn uint64, castor []byte, groupid []byte) *types.Block {
	chain.lock.Lock("CastBlock")
	defer chain.lock.Unlock("CastBlock")

	latestBlock := chain.latestBlock
	if latestBlock == nil {
		logger.Errorf("Block chain lastest block is nil!")
		return nil
	}
	if height <= latestBlock.Height {
		logger.Errorf("Fail to cast block: height problem. height:%d, local height:%d", height, latestBlock.Height)
		return nil
	}

	block := new(types.Block)
	block.Transactions = chain.transactionPool.PackForCast()
	block.Header = &types.BlockHeader{
		CurTime:    timestamp,
		Height:     height,
		ProveValue: proveValue, Castor: castor,
		GroupId:    groupid,
		TotalQN:    latestBlock.TotalQN + qn,
		StateTree:  common.BytesToHash(latestBlock.StateTree.Bytes()),
		//ProveRoot:  proveRoot,
		PreHash: latestBlock.Hash,
		PreTime: latestBlock.CurTime,
	}
	block.Header.RequestIds = getRequestIdFromTransactions(block.Transactions, latestBlock.RequestIds)

	middleware.PerfLogger.Infof("fin cast object. last: %v height: %v", time.Since(timestamp), height)

	preStateRoot := common.BytesToHash(latestBlock.StateTree.Bytes())
	state, err := service.AccountDBManagerInstance.GetAccountDBByHash(preStateRoot)
	if err != nil {
		logger.Errorf("Fail to new account db while casting block!Latest block height:%d,error:%s", latestBlock.Height, err.Error())
		return nil
	}
	stateRoot, evictedTxs, transactions, receipts, err, _ := chain.executor.Execute(state, block, height, "casting")
	middleware.PerfLogger.Infof("fin execute txs. last: %v height: %v", time.Since(timestamp), height)
	txLogger.Debugf("evicted tx len:%d,receipts len:%d", len(evictedTxs), len(receipts))

	transactionHashes := make([]common.Hashes, len(transactions))
	block.Transactions = transactions
	for i, transaction := range transactions {
		hashes := common.Hashes{}
		hashes[0] = transaction.Hash
		hashes[1] = transaction.SubHash
		transactionHashes[i] = hashes

	}
	block.Header.Transactions = transactionHashes
	block.Header.TxTree = calcTxTree(block.Transactions)
	block.Header.EvictedTxs = evictedTxs
	middleware.PerfLogger.Infof("fin calcTxTree. last: %v height: %v", time.Since(timestamp), height)

	block.Header.StateTree = common.BytesToHash(stateRoot.Bytes())
	block.Header.ReceiptTree = calcReceiptsTree(receipts)
	block.Header.Hash = block.Header.GenHash()
	middleware.PerfLogger.Infof("fin calcReceiptsTree. last: %v height: %v", time.Since(timestamp), height)

	chain.verifiedBlocks.Add(block.Header.Hash, &castingBlock{state: state, receipts: receipts,})
	if len(block.Transactions) != 0 {
		chain.verifiedBodyCache.Add(block.Header.Hash, block.Transactions)
	}

	logger.Debugf("Casting block %d,hash:%v,qn:%d,tx:%d,tx tree root:%v,prove value:%v,state tree root:%s,pre state tree:%s",
		height, block.Header.Hash.String(), block.Header.TotalQN, len(block.Transactions), block.Header.TxTree.Hex(),
		consensusHelper.VRFProve2Value(block.Header.ProveValue), block.Header.StateTree.String(), preStateRoot.String())
	return block
}

func getRequestIdFromTransactions(transactions []*types.Transaction, lastOne map[string]uint64) map[string]uint64 {
	result := make(map[string]uint64)
	for key, value := range lastOne {
		result[key] = value
	}

	if nil != transactions && 0 != len(transactions) {
		for _, tx := range transactions {
			if result[tx.Target] < tx.RequestId {
				result[tx.Target] = tx.RequestId
			}
		}
	}

	return result
}

func (chain *blockChain) GenerateBlock(bh types.BlockHeader) *types.Block {
	block := &types.Block{
		Header: &bh,
	}

	txs, missTxs, _ := chain.queryTxsByBlockHash(bh.Hash, bh.Transactions)

	if len(missTxs) != 0 {
		logger.Debugf("GenerateBlock can not get all txs,return nil block!")
		return nil
	}
	block.Transactions = txs
	return block
}

//验证一个铸块（如本地缺少交易，则异步网络请求该交易）
//返回值:
// 0, 验证通过
// -1，验证失败
// 1 无法验证（缺少交易，已异步向网络模块请求）
// 2 无法验证（前一块在链上不存存在）
func (chain *blockChain) VerifyBlock(bh types.BlockHeader) ([]common.Hashes, int8) {
	chain.lock.Lock("VerifyCastingBlock")
	defer chain.lock.Unlock("VerifyCastingBlock")

	return chain.verifyBlock(bh, nil)
}

func (chain *blockChain) Height() uint64 {
	if nil == chain.latestBlock {
		return math.MaxUint64
	}
	return chain.latestBlock.Height
}

func (chain *blockChain) TotalQN() uint64 {
	if nil == chain.latestBlock {
		return 0
	}
	return chain.latestBlock.TotalQN
}

//铸块成功，上链
//返回值: 0,上链成功
//       -1，验证失败
//        1, 丢弃该块(链上已存在该块）
//        2,丢弃该块（链上存在QN值更大的相同高度块)
//        3,分叉调整
func (chain *blockChain) AddBlockOnChain(source string, b *types.Block, situation types.AddBlockOnChainSituation) types.AddBlockResult {
	if validateCode, result := chain.consensusVerify(source, b); !result {
		return validateCode
	}
	chain.lock.Lock("AddBlockOnChain")
	defer chain.lock.Unlock("AddBlockOnChain")
	return chain.addBlockOnChain(source, b, situation)
}

func (chain *blockChain) QueryBlockByHash(hash common.Hash) *types.Block {
	chain.lock.RLock("QueryBlockByHash")
	defer chain.lock.RUnlock("QueryBlockByHash")
	return chain.queryBlockByHash(hash)
}

func (chain *blockChain) QueryBlock(height uint64) *types.Block {
	chain.lock.RLock("QueryBlock")
	defer chain.lock.RUnlock("QueryBlock")

	var b *types.Block
	for i := height; i <= chain.Height(); i++ {
		bh := chain.queryBlockHeaderByHeight(i, true)
		if nil == bh {
			continue
		}
		b = chain.queryBlockByHash(bh.Hash)
		if nil == b {
			continue
		}
		break
	}
	return b
}

func (chain *blockChain) Remove(block *types.Block) bool {
	chain.lock.Lock("Remove Top")
	defer chain.lock.Unlock("Remove Top")

	//if block.Header.Hash != chain.latestBlock.Hash {
	//	return false
	//}
	return chain.remove(block)
}

func (chain *blockChain) GetVerifyHash(height uint64) (common.Hash, error) {
	key := utility.UInt64ToByte(height)
	raw, err := chain.verifyHashDB.Get(key)
	logger.Debugf("Get verify hash.Height:%d,hash:%s", height, common.BytesToHash(raw).String())
	return common.BytesToHash(raw), err
}

func (chain *blockChain) TopBlock() *types.BlockHeader {
	result := chain.latestBlock
	return result
}

func (chain *blockChain) GetTransaction(txHash common.Hash) (*types.Transaction, error) {
	return chain.transactionPool.GetTransaction(txHash)
}

func (chain *blockChain) GetBalance(address common.Address) *big.Int {
	latestStateDB := service.AccountDBManagerInstance.GetLatestStateDB()
	if nil == latestStateDB {
		return nil
	}

	return latestStateDB.GetBalance(common.BytesToAddress(address.Bytes()))
}

func (chain *blockChain) Close() {
	chain.hashDB.Close()
	chain.heightDB.Close()
	chain.verifyHashDB.Close()
}

func (chain *blockChain) queryBlockHeaderByHeight(height interface{}, cache bool) *types.BlockHeader {
	var key []byte
	switch height.(type) {
	case []byte:
		key = height.([]byte)
	default:
		if cache {
			h := height.(uint64)
			result, ok := chain.topBlocks.Get(h)
			if ok && nil != result {
				return result.(*types.BlockHeader)
			}
		}

		key = generateHeightKey(height.(uint64))
	}

	result, err := chain.heightDB.Get(key)
	if result != nil {
		var header *types.BlockHeader
		header, err = types.UnMarshalBlockHeader(result)
		if err != nil {
			return nil
		}

		return header
	} else {
		return nil
	}
}

// 删除块 只删除最高块
func (chain *blockChain) remove(block *types.Block) bool {
	if nil == block {
		return true
	}
	hash := block.Header.Hash
	height := block.Header.Height
	logger.Debugf("remove hash:%s height:%d ", hash.Hex(), height)

	chain.markRemoveBlock(block)

	chain.topBlocks.Remove(height)
	chain.hashDB.Delete(hash.Bytes())
	chain.heightDB.Delete(generateHeightKey(height))
	chain.verifyHashDB.Delete(utility.UInt64ToByte(height))

	preBlock := chain.queryBlockByHash(block.Header.PreHash)
	if preBlock == nil {
		logger.Errorf("Query nil block header by hash  while removing block! Hash:%s,height:%d, preHash :%s", hash.Hex(), height, block.Header.PreHash.Hex())
		return false
	}
	preHeader := preBlock.Header
	chain.latestBlock = preHeader

	preHeaderByte, _ := types.MarshalBlockHeader(preHeader)
	chain.heightDB.Put([]byte(latestBlockKey), preHeaderByte)

	chain.transactionPool.UnMarkExecuted(block.Transactions)
	chain.eraseRemoveBlockMark()
	return true
}

func (chain *blockChain) HasBlockByHash(hash common.Hash) bool {
	result, err := chain.hashDB.Has(hash.Bytes())
	if err != nil {
		result = false
	}

	return result
}

func (chain *blockChain) queryBlockByHash(hash common.Hash) *types.Block {
	result, err := chain.hashDB.Get(hash.Bytes())

	if result != nil {
		var block *types.Block
		block, err = types.UnMarshalBlock(result)
		if err != nil || &block == nil {
			return nil
		}
		return block
	} else {
		return nil
	}
}

func (chain *blockChain) queryBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	block := chain.queryBlockByHash(hash)
	if nil == block {
		return nil
	}
	return block.Header
}

func (chain *blockChain) getAccountDBByHeight(height uint64) (*account.AccountDB, error) {
	chain.lock.RLock("getAccountDBByHeight")
	defer chain.lock.RUnlock("getAccountDBByHeight")

	header := chain.queryBlockHeaderByHeight(height, false)
	if header == nil {
		return nil, fmt.Errorf("no data at height %v", height)
	}

	return service.AccountDBManagerInstance.GetAccountDBByHash(header.StateTree)
}

func (chain *blockChain) queryTxsByBlockHash(blockHash common.Hash, txHashList []common.Hashes) ([]*types.Transaction, []common.Hashes, error) {
	if nil == txHashList || 0 == len(txHashList) {
		return nil, nil, service.ErrNil
	}

	verifiedBody, _ := chain.verifiedBodyCache.Get(blockHash)
	var verifiedTxs []*types.Transaction
	if nil != verifiedBody {
		verifiedTxs = verifiedBody.([]*types.Transaction)
	}

	txs := make([]*types.Transaction, 0)
	need := make([]common.Hashes, 0)
	var err error
	for _, hash := range txHashList {
		var tx *types.Transaction
		if verifiedTxs != nil {
			for _, verifiedTx := range verifiedTxs {
				if verifiedTx.Hash == hash[0] && verifiedTx.SubHash == hash[1] {
					tx = verifiedTx
					break
				}
			}
		}

		if tx == nil {
			tx, err = chain.transactionPool.GetTransaction(hash[0])
		}

		if tx != nil && tx.SubHash == hash[1] {
			txs = append(txs, tx)
		} else {
			need = append(need, hash)
		}
	}
	return txs, need, err
}

func (chain *blockChain) versionValidate() bool {
	genesisHeader := chain.queryBlockHeaderByHeight(uint64(0), true)
	if genesisHeader == nil {
		return false
	}
	version := genesisHeader.Nonce
	if version != ChainDataVersion {
		return false
	}
	return true
}

func (chain *blockChain) buildCache(size uint64) {
	var start uint64
	if chain.latestBlock.Height < size {
		start = 0
	} else {
		start = chain.latestBlock.Height - (size - 1)
	}

	for i := start; i < chain.latestBlock.Height; i++ {
		chain.topBlocks.Add(i, chain.queryBlockHeaderByHeight(i, false))
	}
}

func generateHeightKey(height uint64) []byte {
	h := make([]byte, 8)
	binary.BigEndian.PutUint64(h, height)
	return h
}

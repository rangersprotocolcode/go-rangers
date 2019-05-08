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
	"errors"
)

const (
	latestBlockKey = "bcurrent"

	addBlockMark    = "addBlockMark"
	removeBlockMark = "removeBlockMark"

	chainPieceLength      = 9
	chainPieceBlockLength = 6

	hashDBPrefix       = "block"
	heightDBPrefix     = "height"
	stateDBPrefix      = "state"
	verifyHashDBPrefix = "verifyHash"

	topBlocksCacheSize = 10
)

var blockChainImpl *blockChain

type blockChain struct {
	init bool

	latestBlock   *types.BlockHeader
	latestStateDB *account.AccountDB
	requestIds    map[string]uint64

	topBlocks         *lru.Cache
	futureBlocks      *lru.Cache
	verifiedBlocks    *lru.Cache
	castedBlock       *lru.Cache
	verifiedBodyCache *lru.Cache

	hashDB       db.Database
	heightDB     db.Database
	verifyHashDB db.Database
	stateDB      account.AccountDatabase

	executor        *VMExecutor
	forkProcessor   *forkProcessor
	transactionPool TransactionPool
	bonusManager    *BonusManager

	lock middleware.Loglock
}

type castingBlock struct {
	state    *account.AccountDB
	receipts types.Receipts
}

func initBlockChain() error {
	chain := &blockChain{lock: middleware.NewLoglock("chain")}

	var err error
	chain.futureBlocks, err = lru.New(10)
	if err != nil {
		logger.Errorf("Init cache error:%s", err.Error())
		return err
	}
	chain.verifiedBlocks, err = lru.New(10)
	if err != nil {
		logger.Errorf("Init cache error:%s", err.Error())
		return err
	}
	chain.topBlocks, err = lru.New(topBlocksCacheSize)
	if err != nil {
		logger.Errorf("Init cache error:%s", err.Error())
		return err
	}
	chain.castedBlock, err = lru.New(10)
	if err != nil {
		logger.Errorf("Init cache error:%s", err.Error())
		return err
	}
	chain.verifiedBodyCache, err = lru.New(50)
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

	db, err := db.NewDatabase(stateDBPrefix)
	if err != nil {
		logger.Debugf("Init block chain error! Error:%s", err.Error())
		return err
	}
	chain.stateDB = account.NewDatabase(db)

	chain.bonusManager = newBonusManager()
	chain.executor = NewVMExecutor(chain)
	chain.forkProcessor = initForkProcessor()

	initMinerManager()

	chain.latestBlock = chain.queryBlockHeaderByHeight([]byte(latestBlockKey), false)
	if chain.latestBlock == nil {
		chain.insertGenesisBlock()
	} else {
		chain.ensureChainConsistency()
		if !chain.versionValidate() {
			fmt.Println("Illegal data version! Please delete the directory d0 and restart the program!")
			os.Exit(0)
		}
		chain.buildCache(topBlocksCacheSize)
		state, err := account.NewAccountDB(common.BytesToHash(chain.latestBlock.StateTree.Bytes()), chain.stateDB)
		if err != nil {
			panic("Init blockChain new state db error:" + err.Error())
		}
		chain.latestStateDB = state
	}
	chain.init = true
	blockChainImpl = chain

	chain.transactionPool = NewTransactionPool()
	return nil
}

func (chain *blockChain) CastBlock(height uint64, proveValue *big.Int, proveRoot common.Hash, qn uint64, castor []byte, groupid []byte) *types.Block {
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
		CurTime:    time.Now(),
		Height:     height,
		ProveValue: proveValue, Castor: castor,
		GroupId:    groupid,
		TotalQN:    latestBlock.TotalQN + qn,
		StateTree:  common.BytesToHash(latestBlock.StateTree.Bytes()),
		ProveRoot:  proveRoot,
		PreHash:    latestBlock.Hash,
		PreTime:    latestBlock.CurTime,
	}
	block.Header.RequestIds = getRequestIdFromTransactions(block.Transactions, latestBlock.RequestIds)

	preStateRoot := common.BytesToHash(latestBlock.StateTree.Bytes())
	state, err := account.NewAccountDB(preStateRoot, chain.stateDB)
	if err != nil {
		logger.Errorf("Fail to new account db while casting block!Latest block height:%d,error:%s", latestBlock.Height, err.Error())
		return nil
	}
	stateRoot, evictedTxs, transactions, receipts, err, _ := chain.executor.Execute(state, block, height, "casting")

	transactionHashes := make([]common.Hash, len(transactions))
	block.Transactions = transactions
	for i, transaction := range transactions {
		transactionHashes[i] = transaction.Hash
	}
	block.Header.Transactions = transactionHashes
	block.Header.TxTree = calcTxTree(block.Transactions)
	block.Header.EvictedTxs = evictedTxs

	block.Header.StateTree = common.BytesToHash(stateRoot.Bytes())
	block.Header.ReceiptTree = calcReceiptsTree(receipts)
	block.Header.Hash = block.Header.GenHash()

	chain.verifiedBlocks.Add(block.Header.Hash, &castingBlock{state: state, receipts: receipts,})
	chain.castedBlock.Add(block.Header.Hash, block)
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
func (chain *blockChain) VerifyBlock(bh types.BlockHeader) ([]common.Hash, int8) {
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

func (chain *blockChain) Clear() error {
	chain.lock.Lock("Clear")
	defer chain.lock.Unlock("Clear")

	chain.init = false
	chain.latestBlock = nil
	chain.topBlocks, _ = lru.New(1000)

	var err error
	chain.hashDB.Close()
	chain.heightDB.Close()
	chain.verifyHashDB.Close()
	os.RemoveAll(db.DEFAULT_FILE)

	db, err := db.NewDatabase(stateDBPrefix)
	if err != nil {
		logger.Debugf("Init block chain error! Error:%s", err.Error())
		return err
	}
	chain.stateDB = account.NewDatabase(db)
	chain.executor = NewVMExecutor(chain)

	chain.insertGenesisBlock()
	chain.init = true
	chain.transactionPool.Clear()
	return err
}

func (chain *blockChain) GetVerifyHash(height uint64) (common.Hash, error) {
	key := utility.UInt64ToByte(height)
	raw, err := chain.verifyHashDB.Get(key)
	return common.BytesToHash(raw), err
}

func (chain *blockChain) TopBlock() *types.BlockHeader {
	result := chain.latestBlock
	return result
}

func (chain *blockChain) GetAccountDBByHash(hash common.Hash) (*account.AccountDB, error) {
	header := chain.queryBlockHeaderByHash(hash)
	if header != nil {
		return account.NewAccountDB(header.StateTree, chain.stateDB)
	}
	return nil, errors.New("block hash not exist")
}

func (chain *blockChain) GetTransaction(txHash common.Hash) (*types.Transaction, error) {
	return chain.transactionPool.GetTransaction(txHash)
}

func (chain *blockChain) GetTransactionPool() TransactionPool {
	return chain.transactionPool
}

func (chain *blockChain) GetBalance(address common.Address) *big.Int {
	if nil == chain.latestStateDB {
		return nil
	}

	return chain.latestStateDB.GetBalance(common.BytesToAddress(address.Bytes()))
}

func (chain *blockChain) GetAccountDB() *account.AccountDB {
	if nil == chain.latestStateDB {
		return nil
	}

	return chain.latestStateDB
}

func (chain *blockChain) GetNonce(address common.Address) uint64 {
	if nil == chain.latestStateDB {
		return 0
	}

	return chain.latestStateDB.GetNonce(common.BytesToAddress(address.Bytes()))
}

func (chain *blockChain) Close() {
	chain.hashDB.Close()
	chain.heightDB.Close()
	chain.verifyHashDB.Close()
}

func (chain *blockChain) GetBonusManager() *BonusManager {
	return chain.bonusManager
}

func (chain *blockChain) LatestStateDB() *account.AccountDB {
	return chain.latestStateDB
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
	chain.latestStateDB, _ = account.NewAccountDB(preHeader.StateTree, chain.stateDB)

	preHeaderByte, _ := types.MarshalBlockHeader(preHeader)
	chain.heightDB.Put([]byte(latestBlockKey), preHeaderByte)

	chain.transactionPool.UnMarkExecuted(block.Transactions)
	chain.eraseRemoveBlockMark()
	return true
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
	var header *types.BlockHeader
	h := height
	for {
		header = chain.queryBlockHeaderByHeight(h, true)
		if header != nil || h == 0 {
			break
		}
		h--
	}
	if header == nil {
		return nil, fmt.Errorf("no data at height %v-%v", h, height)
	}
	return account.NewAccountDB(header.StateTree, chain.stateDB)
}

func (chain *blockChain) queryTxsByBlockHash(blockHash common.Hash, txHashList []common.Hash) ([]*types.Transaction, []common.Hash, error) {
	if nil == txHashList || 0 == len(txHashList) {
		return nil, nil, ErrNil
	}

	verifiedBody, _ := chain.verifiedBodyCache.Get(blockHash)
	var verifiedTxs []*types.Transaction
	if nil != verifiedBody {
		verifiedTxs = verifiedBody.([]*types.Transaction)
	}

	txs := make([]*types.Transaction, 0)
	need := make([]common.Hash, 0)
	var err error
	for _, hash := range txHashList {
		var tx *types.Transaction
		if verifiedTxs != nil {
			for _, verifiedTx := range verifiedTxs {
				if verifiedTx.Hash == hash {
					tx = verifiedTx
					break
				}
			}
		}

		if tx == nil {
			tx, err = chain.transactionPool.GetTransaction(hash)
		}

		if tx != nil {
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

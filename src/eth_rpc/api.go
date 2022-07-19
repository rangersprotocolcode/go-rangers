package eth_rpc

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/core"
	"com.tuntun.rocket/node/src/eth_tx"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/storage/rlp"
	"com.tuntun.rocket/node/src/utility"
	"com.tuntun.rocket/node/src/vm"
	"encoding/json"
	"errors"
	"math/big"
	"sync"
)

type ethAPIService struct{}

// CallArgs represents the arguments for a call.
type CallArgs struct {
	From     *common.Address `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *utility.Uint64 `json:"gas"`
	GasPrice *utility.Big    `json:"gasPrice"`
	Value    *utility.Big    `json:"value"`
	Data     *utility.Bytes  `json:"data"`
}

// SendTxArgs represents the arguments to sumbit a new transaction into the transaction pool.
type SendTxArgs struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *utility.Uint64 `json:"gas"`
	GasPrice *utility.Big    `json:"gasPrice"`
	Value    *utility.Big    `json:"value"`
	Nonce    *utility.Uint64 `json:"nonce"`
	// We accept "data" and "input" for backwards-compatibility reasons. "input" is the
	// newer name and should be preferred by clients.
	Data  *utility.Bytes `json:"data"`
	Input *utility.Bytes `json:"input"`
}

// RPCTransaction represents a transaction that will serialize to the RPC representation of a transaction
type RPCTransaction struct {
	BlockHash        *common.Hash    `json:"blockHash"`
	BlockNumber      *utility.Big    `json:"blockNumber"`
	From             common.Address  `json:"from"`
	Gas              utility.Uint64  `json:"gas"`
	GasPrice         *utility.Big    `json:"gasPrice"`
	Hash             common.Hash     `json:"hash"`
	Input            utility.Bytes   `json:"input"`
	Nonce            utility.Uint64  `json:"nonce"`
	To               *common.Address `json:"to"`
	TransactionIndex *utility.Uint64 `json:"transactionIndex"`
	Value            *utility.Big    `json:"value"`
	V                *utility.Big    `json:"v"`
	R                *utility.Big    `json:"r"`
	S                *utility.Big    `json:"s"`
}

type RPCBlock struct {
	Difficulty       *utility.Big   `json:"difficulty"`
	ExtraData        utility.Bytes  `json:"extraData"`
	GasLimit         utility.Uint64 `json:"gasLimit"`
	GasUsed          utility.Uint64 `json:"gasUsed"`
	Hash             common.Hash    `json:"hash"`
	Bloom            string         `json:"logsBloom"`
	Miner            common.Address `json:"miner"`
	MixHash          string         `json:"mixHash"`
	Nonce            utility.Bytes  `json:"nonce"`
	Number           utility.Uint64 `json:"number"`
	ParentHash       common.Hash    `json:"parentHash"`
	ReceiptsRoot     common.Hash    `json:"receiptsRoot"`
	UncleHash        common.Hash    `json:"sha3Uncles"`
	Size             utility.Uint64 `json:"size"`
	StateRoot        common.Hash    `json:"stateRoot"`
	Timestamp        utility.Uint64 `json:"timestamp"`
	TotalDifficulty  *utility.Big   `json:"totalDifficulty"`
	Transactions     []interface{}  `json:"transactions"`
	TransactionsRoot common.Hash    `json:"transactionsRoot"`
	Uncles           []string       `json:"uncles"`
}

var (
	gasPrice               = big.NewInt(1)
	gasLimit        uint64 = 2000000
	callLock               = sync.Mutex{}
	nonce                  = []byte{1, 2, 3, 4, 5, 6, 7, 8}
	difficulty             = utility.Big(*big.NewInt(32))
	totalDifficulty        = utility.Big(*big.NewInt(180))
)

// SendRawTransaction will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (api *ethAPIService) SendRawTransaction(encodedTx utility.Bytes) (common.Hash, *types.Transaction, error) {
	tx := new(eth_tx.Transaction)
	if err := rlp.DecodeBytes(encodedTx, tx); err != nil {
		return common.Hash{}, nil, err
	}
	logger.Debugf("raw tx hash:%v", tx.Hash().String())

	signer := eth_tx.NewEIP155Signer(common.GetChainId(utility.MaxUint64))
	sender, err := eth_tx.Sender(signer, tx)
	if err != nil {
		logger.Debugf("err:%v", err.Error())
		return common.Hash{}, nil, err
	}

	rocketTx := eth_tx.ConvertTx(tx, sender, encodedTx)
	return rocketTx.Hash, rocketTx, nil
}

//Call executes the given transaction on the state for the given block number.

//Additionally, the caller can specify a batch of contract for fields overriding.

//Note, this function doesn't make and changes in the state/blockchain and is
//useful to execute and retrieve values.
func (s *ethAPIService) Call(args CallArgs, blockNrOrHash BlockNumberOrHash) (utility.Bytes, error) {

	number, _ := blockNrOrHash.Number()
	logger.Debugf("call:%v,%v", args, number)
	accountdb := getAccountDBByHashOrHeight(blockNrOrHash)
	block := getBlockByHashOrHeight(blockNrOrHash)
	if accountdb == nil || block == nil {
		return nil, errors.New("param invalid")
	}

	vmCtx := vm.Context{}
	vmCtx.CanTransfer = vm.CanTransfer
	vmCtx.Transfer = vm.Transfer
	vmCtx.GetHash = core.GetBlockChain().GetBlockHash
	if args.From != nil {
		vmCtx.Origin = *args.From
	}
	vmCtx.Coinbase = common.BytesToAddress(block.Header.Castor)
	vmCtx.BlockNumber = new(big.Int).SetUint64(block.Header.Height)
	vmCtx.Time = new(big.Int).SetUint64(uint64(block.Header.CurTime.Unix()))
	//set constant value
	vmCtx.Difficulty = new(big.Int).SetUint64(123)
	vmCtx.GasPrice = gasPrice
	vmCtx.GasLimit = gasLimit

	var transferValue *big.Int
	if args.Value != nil {
		value := args.Value.ToInt()
		transferValue = big.NewInt(0).Mod(value, big.NewInt(1000000000000000000))
	} else {
		transferValue = big.NewInt(0)
	}

	var data []byte
	if args.Data != nil {
		data = *args.Data
	}
	vmInstance := vm.NewEVMWithNFT(vmCtx, accountdb, accountdb)
	caller := vm.AccountRef(vmCtx.Origin)
	var (
		result          []byte
		leftOverGas     uint64
		contractAddress common.Address
		err             error
	)
	logger.Debugf("before vm instance")
	if args.To == nil {
		result, contractAddress, leftOverGas, _, err = vmInstance.Create(caller, data, vmCtx.GasLimit, transferValue)
		logger.Debugf("[eth_call]After execute contract create!Contract address:%s, leftOverGas: %d,error:%v", contractAddress.GetHexString(), leftOverGas, err)
	} else {
		result, leftOverGas, _, err = vmInstance.Call(caller, *args.To, data, vmCtx.GasLimit, transferValue)
		logger.Debugf("[eth_call]After execute contract call! result:%v,leftOverGas: %d,error:%v", result, leftOverGas, err)
	}

	if err != nil {
		return nil, revertError{err, common.ToHex(result)}
	}
	return result, nil
}

// ChainId returns the chainID value for transaction replay protection.
func (api *ethAPIService) ChainId() *utility.Big {
	return (*utility.Big)(common.GetChainId(utility.MaxUint64))
}

// Version returns the current ethereum protocol version.
func (s *ethAPIService) Version() string {
	return common.NetworkId()
}

// ProtocolVersion returns the current Ethereum protocol version this node supports
func (s *ethAPIService) ProtocolVersion() utility.Uint {
	return utility.Uint(common.ProtocolVersion)
}

// ClientVersion returns the current client version.
func (api *ethAPIService) ClientVersion() string {
	return "Rangers/" + common.Version + "/centos-amd64/go1.17.3"
}

// BlockNumber returns the block number of the chain head.
func (api *ethAPIService) BlockNumber() utility.Uint64 {
	blockNumber := core.GetBlockChain().Height()
	return utility.Uint64(blockNumber)
}

// GasPrice returns a suggestion for a gas price.
func (s *ethAPIService) GasPrice() (*utility.Big, error) {
	gasPrice := utility.Big(*big.NewInt(1))
	return &gasPrice, nil
}

func (s *ethAPIService) EstimateGas(args CallArgs, blockNrOrHash *BlockNumberOrHash) (utility.Uint64, error) {
	return utility.Uint64(21000), nil
}

// GetBalance returns the amount of wei for the given address in the state of the
// given block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta
// block numbers are also allowed.
func (api *ethAPIService) GetBalance(address common.Address, blockNrOrHash BlockNumberOrHash) (*utility.Big, error) {
	accountDB := getAccountDBByHashOrHeight(blockNrOrHash)
	if accountDB == nil {
		return nil, errors.New("param invalid")
	}
	balanceRaw := accountDB.GetBalance(address)
	//base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
	//balance := big.NewInt(0).Mod(balanceRaw, base)
	result := utility.Big(*balanceRaw)
	return &result, nil
}

// GetCode returns the code stored at the given address in the state for the given block number.
func (s *ethAPIService) GetCode(address common.Address, blockNrOrHash BlockNumberOrHash) (utility.Bytes, error) {
	accountDB := getAccountDBByHashOrHeight(blockNrOrHash)
	if accountDB == nil {
		return nil, errors.New("param invalid")
	}
	code := accountDB.GetCode(address)
	if code == nil {
		code = utility.Bytes{}
	}
	return code, nil
}

// GetStorageAt returns the storage from the state at the given address, key and
// block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta block
// numbers are also allowed.
func (s *ethAPIService) GetStorageAt(address common.Address, key string, blockNrOrHash BlockNumberOrHash) (utility.Bytes, error) {
	accountDB := getAccountDBByHashOrHeight(blockNrOrHash)
	if accountDB == nil {
		return nil, errors.New("param invalid")
	}
	value := accountDB.GetData(address, common.HexToHash(key).Bytes())
	if value == nil {
		value = utility.Bytes{}
	}
	return value, nil
}

// GetBlockTransactionCountByNumber returns the number of transactions in the block with the given block number.
func (s *ethAPIService) GetBlockTransactionCountByNumber(blockNr BlockNumber) *utility.Uint {
	block := getBlockByHashOrHeight(BlockNumberOrHash{BlockNumber: &blockNr})
	if block == nil {
		zero := utility.Uint(0)
		return &zero
	}
	txNum := utility.Uint(len(block.Transactions))
	return &txNum
}

// GetBlockTransactionCountByHash returns the number of transactions in the block with the given hash.
func (s *ethAPIService) GetBlockTransactionCountByHash(blockHash common.Hash) *utility.Uint {
	block := getBlockByHashOrHeight(BlockNumberOrHash{BlockHash: &blockHash})
	if block == nil {
		zero := utility.Uint(0)
		return &zero
	}
	txNum := utility.Uint(len(block.Transactions))
	return &txNum
}

// GetTransactionCount returns the number of transactions the given address has sent for the given block number
func (s *ethAPIService) GetTransactionCount(address common.Address, blockNrOrHash BlockNumberOrHash) (*utility.Uint64, error) {
	accountDB := getAccountDBByHashOrHeight(blockNrOrHash)
	if accountDB == nil {
		return nil, errors.New("param invalid")
	}
	nonce := utility.Uint64(accountDB.GetNonce(address))
	return &nonce, nil
}

// GetTransactionReceipt returns the transaction receipt for the given transaction hash.
func (s *ethAPIService) GetTransactionReceipt(hash common.Hash) (map[string]interface{}, error) {
	executedTx := service.GetTransactionPool().GetExecuted(hash)
	if executedTx == nil {
		return nil, nil
	}

	tx, err := types.UnMarshalTransaction(executedTx.Transaction)
	if err != nil {
		return nil, nil
	}

	fields := map[string]interface{}{
		"blockHash": executedTx.Receipt.BlockHash,
		//"blockNumber":       executedTx.Receipt.Height,
		"blockNumber":       (*utility.Big)(new(big.Int).SetUint64(executedTx.Receipt.Height)),
		"transactionHash":   executedTx.Receipt.TxHash,
		"from":              tx.Source,
		"gasUsed":           utility.Uint64(0),
		"cumulativeGasUsed": utility.Uint64(0),
		"contractAddress":   nil,
		"logs":              executedTx.Receipt.Logs,
		"to":                tx.Target,
		"transactionIndex":  utility.Uint64(0),
		"logsBloom":         "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
	}

	// Assign receipt status or post state.
	if len(executedTx.Receipt.PostState) > 0 {
		fields["root"] = utility.Bytes(executedTx.Receipt.PostState)
	} else {
		fields["status"] = utility.Uint(executedTx.Receipt.Status)
	}
	if executedTx.Receipt.Logs == nil {
		fields["logs"] = [][]*types.Log{}
	}
	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if executedTx.Receipt.ContractAddress != (common.Address{}) {
		fields["contractAddress"] = executedTx.Receipt.ContractAddress
	}

	if tx.Target == "" && (tx.Type == types.TransactionTypeETHTX || tx.Type == types.TransactionTypeContract) {
		fields["to"] = executedTx.Receipt.ContractAddress
	}
	receipts := types.Receipts{&executedTx.Receipt.Receipt}
	logBloom := types.CreateBloom(receipts)
	fields["logsBloom"] = logBloom
	return fields, nil
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned.
func (s *ethAPIService) GetBlockByHash(hash common.Hash, fullTx bool) (*RPCBlock, error) {
	block := core.GetBlockChain().QueryBlockByHash(hash)
	if block == nil {
		return nil, errors.New("param invalid")
	}
	return adaptRPCBlock(block, fullTx), nil
}

// GetBlockByNumber returns the requested canonical block.
// * When blockNr is -1 the chain head is returned.
// * When blockNr is -2 the pending chain head is returned.
// * When fullTx is true all transactions in the block are returned, otherwise
//   only the transaction hash is returned.
func (s *ethAPIService) GetBlockByNumber(number BlockNumber, fullTx bool) (*RPCBlock, error) {
	var block *types.Block
	if number == PendingBlockNumber || number == LatestBlockNumber || number == EarliestBlockNumber {
		height := core.GetBlockChain().Height()
		block = core.GetBlockChain().QueryBlock(height)
	} else {
		block = core.GetBlockChain().QueryBlock(uint64(number))
	}
	if block == nil {
		return nil, errors.New("param invalid")
	}
	return adaptRPCBlock(block, fullTx), nil
}

// GetTransactionByHash returns the transaction for the given hash
func (s *ethAPIService) GetTransactionByHash(hash common.Hash) (*RPCTransaction, error) {
	executedTx := service.GetTransactionPool().GetExecuted(hash)
	if executedTx != nil {
		tx, err := types.UnMarshalTransaction(executedTx.Transaction)
		if err != nil {
			return nil, nil
		}
		return newRPCTransaction(&tx, executedTx.Receipt.BlockHash, executedTx.Receipt.Height, 0), nil
	}

	tx, _ := service.GetTransactionPool().GetTransaction(hash)
	if tx != nil {
		return newRPCTransaction(tx, common.Hash{}, 0, 0), nil
	}

	// Transaction unknown, return as such
	return nil, nil
}

//// GetLogs returns logs matching the given argument that are stored within the state.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getlogs
func (s *ethAPIService) GetLogs(crit types.FilterCriteria) ([]*types.Log, error) {
	result := core.GetLogs(crit)
	return result, nil
}

// newRPCTransaction returns a transaction that will serialize to the RPC
// representation, with the given location metadata set (if available).
func newRPCTransaction(tx *types.Transaction, blockHash common.Hash, blockNumber uint64, index uint64) *RPCTransaction {
	if tx == nil || tx.Type != types.TransactionTypeETHTX {
		return nil
	}
	result := &RPCTransaction{
		From:     common.HexToAddress(tx.Source),
		Gas:      utility.Uint64(0),
		GasPrice: (*utility.Big)(gasPrice),
		Hash:     tx.Hash,
		Nonce:    utility.Uint64(tx.Nonce),
	}
	if tx.Target != "" {
		targetAddress := common.HexToAddress(tx.Target)
		result.To = &targetAddress
	}

	var data types.ContractData
	err := json.Unmarshal([]byte(tx.Data), &data)
	if err == nil {
		result.Input = utility.Bytes(data.AbiData)

		transferValue, err := utility.StrToBigInt(data.TransferValue)
		if err == nil {
			result.Value = (*utility.Big)(transferValue)
		}
	}

	if blockHash != (common.Hash{}) {
		result.BlockHash = &blockHash
		result.BlockNumber = (*utility.Big)(new(big.Int).SetUint64(blockNumber))
		result.TransactionIndex = (*utility.Uint64)(&index)
	}
	r, s, v := signatureValues(tx)
	result.R = r
	result.V = v
	result.S = s
	return result
}

func signatureValues(tx *types.Transaction) (*utility.Big, *utility.Big, *utility.Big) {
	if tx == nil || tx.Type != types.TransactionTypeETHTX {
		return nil, nil, nil
	}
	ethTx := new(eth_tx.Transaction)
	if err := rlp.DecodeBytes(common.FromHex(tx.ExtraData), ethTx); err != nil {
		return nil, nil, nil
	}
	v, r, s := ethTx.RawSignatureValues()
	return (*utility.Big)(r), (*utility.Big)(s), (*utility.Big)(v)
}

//-----------------------------------------------------------------------------
func getAccountDBByHashOrHeight(blockNrOrHash BlockNumberOrHash) *account.AccountDB {
	var accountDB *account.AccountDB
	if blockNrOrHash.BlockHash != nil {
		accountDB = getAccountDBByHash(*blockNrOrHash.BlockHash)
	} else if blockNrOrHash.BlockNumber != nil {
		var height uint64
		if *blockNrOrHash.BlockNumber == PendingBlockNumber || *blockNrOrHash.BlockNumber == LatestBlockNumber || *blockNrOrHash.BlockNumber == EarliestBlockNumber {
			accountDB = getAccountDBByHeight(0)
		} else {
			height = uint64(*blockNrOrHash.BlockNumber)
			accountDB = getAccountDBByHeight(height)
		}
	}
	return accountDB
}
func getAccountDBByHeight(height uint64) (accountDB *account.AccountDB) {
	if height == 0 {
		accountDB = service.AccountDBManagerInstance.GetLatestStateDB()
	} else {
		b := core.GetBlockChain().QueryBlockHeaderByHeight(height, true)
		if nil == b {
			return nil
		}
		accountDB, _ = service.AccountDBManagerInstance.GetAccountDBByHash(b.StateTree)
	}
	return
}

func getAccountDBByHash(hash common.Hash) (accountDB *account.AccountDB) {
	b := core.GetBlockChain().QueryBlockByHash(hash)
	if nil == b {
		return nil
	}
	accountDB, _ = service.AccountDBManagerInstance.GetAccountDBByHash(b.Header.StateTree)
	return
}

func getBlockByHashOrHeight(blockNrOrHash BlockNumberOrHash) *types.Block {
	var block *types.Block
	if blockNrOrHash.BlockHash != nil {
		block = core.GetBlockChain().QueryBlockByHash(*blockNrOrHash.BlockHash)
	} else if blockNrOrHash.BlockNumber != nil {
		var height uint64
		if *blockNrOrHash.BlockNumber == PendingBlockNumber || *blockNrOrHash.BlockNumber == LatestBlockNumber || *blockNrOrHash.BlockNumber == EarliestBlockNumber {
			header := core.GetBlockChain().TopBlock()
			block = core.GetBlockChain().QueryBlock(header.Height)
		} else {
			height = uint64(*blockNrOrHash.BlockNumber)
			block = core.GetBlockChain().QueryBlock(height)
		}
	}
	return block
}

func adaptRPCBlock(block *types.Block, fullTx bool) *RPCBlock {
	header := block.Header
	rpcBlock := RPCBlock{
		Difficulty:   &difficulty,
		ExtraData:    utility.Bytes(nonce[:]),
		GasLimit:     utility.Uint64(gasLimit),
		GasUsed:      utility.Uint64(200000),
		Hash:         header.Hash,
		Bloom:        "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
		Miner:        common.BytesToAddress(block.Header.Castor),
		MixHash:      "0x0000000000000000000000000000000000000000000000000000000000000000",
		Nonce:        utility.Bytes(nonce[:]),
		Number:       utility.Uint64(header.Height),
		ParentHash:   header.PreHash,
		ReceiptsRoot: header.ReceiptTree,
		//uncle has to be this value(rlpHash([]*Header(nil))) for pass go ethereum client verify because tx uncles is empty
		UncleHash:       common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
		Size:            utility.Uint64(1234),
		StateRoot:       header.StateTree,
		Timestamp:       utility.Uint64(header.CurTime.Unix()),
		TotalDifficulty: &totalDifficulty,
		Uncles:          []string{},
	}
	transactions := make([]interface{}, 0)
	for index, tx := range block.Transactions {
		if fullTx {
			rpcTx := newRPCTransaction(tx, header.Hash, header.Height, uint64(index))
			if rpcTx != nil {
				transactions = append(transactions, rpcTx)
			}
		} else {
			if tx.Type == types.TransactionTypeETHTX {
				transactions = append(transactions, tx.Hash)
			}
		}
	}
	rpcBlock.Transactions = transactions
	if len(transactions) == 0 {
		//transactionsRoot  has to be this value(EmptyRootHash) for pass go ethereum client verify because tx uncles is empty
		rpcBlock.TransactionsRoot = common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	} else {
		rpcBlock.TransactionsRoot = header.TxTree
	}
	return &rpcBlock
}

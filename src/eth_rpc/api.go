// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package eth_rpc

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/core"
	"com.tuntun.rangers/node/src/eth_tx"
	"com.tuntun.rangers/node/src/executor"
	"com.tuntun.rangers/node/src/middleware"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/network"
	"com.tuntun.rangers/node/src/service"
	"com.tuntun.rangers/node/src/storage/account"
	"com.tuntun.rangers/node/src/storage/rlp"
	"com.tuntun.rangers/node/src/utility"
	"com.tuntun.rangers/node/src/vm"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/boolw/go-web3/abi"
	"math/big"
	"strconv"
	"sync"
)

type EthAPIService struct{}

// CallArgs represents the arguments for a call.
type CallArgs struct {
	From     *common.Address `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *utility.Uint64 `json:"gas"`
	GasPrice *utility.Big    `json:"gasPrice"`
	Value    *utility.Big    `json:"value"`
	Data     *utility.Bytes  `json:"data"`
	Input    *utility.Bytes  `json:"input"`
}

// data retrieves the transaction calldata. Input field is preferred.
func (args *CallArgs) data() []byte {
	if args.Input != nil {
		return *args.Input
	}
	if args.Data != nil {
		return *args.Data
	}
	return nil
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

const (
	gasLimit                  uint64 = 30000000
	confirmBlockCount         uint64 = 3
	txGas                     uint64 = 21000 // Per transaction not creating a contract. NOTE: Not payable on data of calls between transactions.
	estimateExpandCoefficient        = 2.5
	generalEstimateGas        uint64 = 500000
	MaxInitCodeSize                  = 2 * 24576 // Maximum initcode to permit in a creation transaction and create instructions
)

var (
	gasPrice        = big.NewInt(1000000000)
	callLock        = sync.Mutex{}
	nonce           = []byte{1, 2, 3, 4, 5, 6, 7, 8}
	difficulty      = utility.Big(*big.NewInt(32))
	totalDifficulty = utility.Big(*big.NewInt(180))

	// ErrNegativeValue is a sanity error to ensure no one is able to specify a
	// transaction with a negative value.
	ErrNegativeValue = errors.New("negative value")
	// ErrAlreadyKnown is returned if the transactions is already contained
	// within the pool.
	ErrAlreadyKnown = errors.New("already known")
	// ErrTxPoolOverflow is returned if the transaction pool is full and can't accept
	// another remote transaction.
	ErrTxPoolOverflow = errors.New("txpool is full")

	// ErrMaxInitCodeSizeExceeded is returned if creation transaction provides the init code bigger
	// than init code size limit.
	ErrMaxInitCodeSizeExceeded = errors.New("max initcode size exceeded")
)

// SendRawTransaction will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (api *EthAPIService) SendRawTransaction(encodedTx utility.Bytes) (*types.Transaction, error) {
	tx := new(eth_tx.Transaction)
	if err := rlp.DecodeBytes(encodedTx, tx); err != nil {
		return nil, err
	}
	logger.Debugf("raw tx hash:%v", tx.Hash().String())

	sender, err := validateTx(tx)
	if err != nil {
		logger.Errorf("tx validate err:%v", err.Error())
		return nil, err
	}

	rocketTx := eth_tx.ConvertTx(tx, sender, encodedTx)
	if common.IsFullNode() {
		_, err := broadcastRawTx(encodedTx.String())
		return rocketTx, err
	}

	return rocketTx, nil
}

//Call executes the given transaction on the state for the given block number.

//Additionally, the caller can specify a batch of contract for fields overriding.

// Note, this function doesn't make and changes in the state/blockchain and is
// useful to execute and retrieve values.
func (s *EthAPIService) Call(args CallArgs, blockNrOrHash BlockNumberOrHash) (utility.Bytes, error) {
	if args.Gas == nil || uint64(*args.Gas) > gasLimit {
		defaultGasLimit := utility.Uint64(gasLimit)
		args.Gas = &defaultGasLimit
	}
	data, err, _ := doCall(args, blockNrOrHash)
	return data, err
}

func (s *EthAPIService) EstimateGas(args CallArgs, blockNrOrHash *BlockNumberOrHash) (utility.Uint64, error) {
	bNrOrHash := BlockNumberOrHashWithNumber(LatestBlockNumber)
	if blockNrOrHash != nil {
		bNrOrHash = *blockNrOrHash
	}

	if args.Gas == nil || uint64(*args.Gas) < txGas || uint64(*args.Gas) > gasLimit {
		defaultGasLimit := utility.Uint64(gasLimit)
		args.Gas = &defaultGasLimit
	}
	_, err, gasUsed := doCall(args, bNrOrHash)

	estimateGas := uint64(float64(gasUsed) * estimateExpandCoefficient)
	if gasUsed != txGas && estimateGas < generalEstimateGas {
		estimateGas = generalEstimateGas
	}
	if estimateGas > gasLimit {
		estimateGas = gasLimit
	}
	return utility.Uint64(estimateGas), err
}

func doCall(args CallArgs, blockNrOrHash BlockNumberOrHash) (utility.Bytes, error, uint64) {
	number, _ := blockNrOrHash.Number()
	logger.Debugf("doCall:%v,%v", args, number)
	accountdb := getAccountDBByHashOrHeight(blockNrOrHash)
	block := getBlockByHashOrHeight(blockNrOrHash)
	if accountdb == nil || block == nil {
		return nil, errors.New("param invalid"), 0
	}

	initialGas := uint64(*args.Gas)
	var contractCreation = false
	if args.To == nil {
		contractCreation = true
	}
	data := args.data()

	var gasErr error
	var intrinsicGas uint64
	intrinsicGas, gasErr = executor.IntrinsicGas(data, contractCreation)
	if gasErr != nil {
		logger.Errorf("IntrinsicGas error:%s", gasErr.Error())
		return nil, gasErr, 0
	}
	if initialGas < intrinsicGas {
		logger.Errorf("gas limit too low,gas limit:%d,intrinsic gas:%d", initialGas, intrinsicGas)
		return nil, errors.New("intrinsic gas too low"), 0
	}

	vmCtx := vm.Context{}
	vmCtx.CanTransfer = vm.CanTransfer
	vmCtx.Transfer = vm.Transfer
	vmCtx.GetHash = core.GetBlockChain().GetBlockHash
	if args.From != nil {
		vmCtx.Origin = *args.From
	}
	vmCtx.GasLimit = initialGas - intrinsicGas
	vmCtx.Coinbase = common.BytesToAddress(block.Header.Castor)
	vmCtx.BlockNumber = new(big.Int).SetUint64(block.Header.Height)
	vmCtx.Time = new(big.Int).SetUint64(uint64(block.Header.CurTime.Unix()))
	//set constant value
	vmCtx.Difficulty = new(big.Int).SetUint64(123)
	vmCtx.GasPrice = gasPrice

	var transferValue *big.Int
	if args.Value != nil {
		transferValue = args.Value.ToInt()
	} else {
		transferValue = big.NewInt(0)
	}

	vmInstance := vm.NewEVMWithNFT(vmCtx, accountdb, accountdb)
	caller := vm.AccountRef(vmCtx.Origin)
	var (
		result          []byte
		leftOverGas     uint64
		contractAddress common.Address
		err             error
	)
	logger.Debugf("before vm instance,intrinsicGas:%d,gasLimit:%d", intrinsicGas, vmCtx.GasLimit)
	if contractCreation {
		result, contractAddress, leftOverGas, _, err = vmInstance.Create(caller, data, vmCtx.GasLimit, transferValue)
		logger.Debugf("[eth_call]After execute contract create!Contract address:%s, leftOverGas: %d,error:%v", contractAddress.GetHexString(), leftOverGas, err)
	} else {
		result, leftOverGas, _, err = vmInstance.Call(caller, *args.To, data, vmCtx.GasLimit, transferValue)
		logger.Debugf("[eth_call]After execute contract call! result:%v,leftOverGas: %d,error:%v", result, leftOverGas, err)
	}

	gasUsed := initialGas - leftOverGas
	if gasUsed < txGas {
		gasUsed = txGas
	}
	// If the result contains a revert reason, try to unpack and return it.
	if err == vm.ErrExecutionReverted && len(result) > 0 {
		err := adaptErrorOutput(err, result)
		return nil, &revertError{err, common.ToHex(result)}, gasUsed
	}
	if err != nil {
		return nil, err, gasUsed
	}
	return result, nil, gasUsed
}

// ChainId returns the chainID value for transaction replay protection.
func (api *EthAPIService) ChainId() *utility.Big {
	return (*utility.Big)(common.GetChainId(utility.MaxUint64))
}

// ProtocolVersion returns the current Ethereum protocol version this node supports
func (s *EthAPIService) ProtocolVersion() utility.Uint {
	return utility.Uint(common.ProtocolVersion)
}

// BlockNumber returns the block number of the chain head.
func (api *EthAPIService) BlockNumber() utility.Uint64 {
	blockNumber := core.GetBlockChain().Height()
	return utility.Uint64(blockNumber)
}

// GasPrice returns a suggestion for a gas price.
func (s *EthAPIService) GasPrice() (*utility.Big, error) {
	gasPrice := utility.Big(*gasPrice)
	return &gasPrice, nil
}

// GetBalance returns the amount of wei for the given address in the state of the
// given block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta
// block numbers are also allowed.
func (api *EthAPIService) GetBalance(address common.Address, blockNrOrHash BlockNumberOrHash) (*utility.Big, error) {
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
func (s *EthAPIService) GetCode(address common.Address, blockNrOrHash BlockNumberOrHash) (utility.Bytes, error) {
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
func (s *EthAPIService) GetStorageAt(address common.Address, key string, blockNrOrHash BlockNumberOrHash) (utility.Bytes, error) {
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
func (s *EthAPIService) GetBlockTransactionCountByNumber(blockNr BlockNumber) *utility.Uint {
	block := getBlockByHashOrHeight(BlockNumberOrHash{BlockNumber: &blockNr})
	if block == nil {
		zero := utility.Uint(0)
		return &zero
	}
	txNum := utility.Uint(len(block.Transactions))
	return &txNum
}

// GetBlockTransactionCountByHash returns the number of transactions in the block with the given hash.
func (s *EthAPIService) GetBlockTransactionCountByHash(blockHash common.Hash) *utility.Uint {
	block := getBlockByHashOrHeight(BlockNumberOrHash{BlockHash: &blockHash})
	if block == nil {
		zero := utility.Uint(0)
		return &zero
	}
	txNum := utility.Uint(len(block.Transactions))
	return &txNum
}

// GetTransactionCount returns the number of transactions the given address has sent for the given block number
func (s *EthAPIService) GetTransactionCount(address common.Address, blockNrOrHash BlockNumberOrHash) (*utility.Uint64, error) {
	accountDB := getAccountDBByHashOrHeight(blockNrOrHash)
	if accountDB == nil {
		return nil, errors.New("param invalid")
	}
	nonce := utility.Uint64(accountDB.GetNonce(address))
	return &nonce, nil
}

// GetTransactionReceipt returns the transaction receipt for the given transaction hash.
func (s *EthAPIService) GetTransactionReceipt(hash common.Hash) (map[string]interface{}, error) {
	executedTx := service.GetTransactionPool().GetExecuted(hash)
	if executedTx == nil {
		return nil, nil
	}

	tx, err := types.UnMarshalTransaction(executedTx.Transaction)
	if err != nil {
		return nil, nil
	}

	topBlock := core.GetBlockChain().TopBlock()
	if topBlock == nil {
		return nil, nil
	}
	//do not return during confirm block
	if executedTx.Receipt.Height+confirmBlockCount > topBlock.Height {
		return nil, nil
	}

	fields := map[string]interface{}{
		"blockHash": executedTx.Receipt.BlockHash,
		//"blockNumber":       executedTx.Receipt.Height,
		"blockNumber":       (*utility.Big)(new(big.Int).SetUint64(executedTx.Receipt.Height)),
		"transactionHash":   executedTx.Receipt.TxHash,
		"from":              tx.Source,
		"gasUsed":           utility.Uint64(executedTx.Receipt.GasUsed),
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
func (s *EthAPIService) GetBlockByHash(hash common.Hash, fullTx bool) (*RPCBlock, error) {
	block := core.GetBlockChain().QueryBlockByHash(hash)
	if block == nil {
		return nil, errors.New("param invalid")
	}
	return adaptRPCBlock(block, fullTx), nil
}

// GetBlockByNumber returns the requested canonical block.
//   - When blockNr is -1 the chain head is returned.
//   - When blockNr is -2 the pending chain head is returned.
//   - When fullTx is true all transactions in the block are returned, otherwise
//     only the transaction hash is returned.
func (s *EthAPIService) GetBlockByNumber(number BlockNumber, fullTx bool) (*RPCBlock, error) {
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
func (s *EthAPIService) GetTransactionByHash(hash common.Hash) (*RPCTransaction, error) {
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

// // GetLogs returns logs matching the given argument that are stored within the state.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getlogs
func (s *EthAPIService) GetLogs(crit types.FilterCriteria) ([]*types.Log, error) {
	result, err := core.GetLogs(crit)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetTransactionByBlockNumberAndIndex returns the transaction for the given block number and index.
func (s *EthAPIService) GetTransactionByBlockNumberAndIndex(blockNr BlockNumber, index utility.Uint) *RPCTransaction {
	var block *types.Block
	if blockNr == PendingBlockNumber || blockNr == LatestBlockNumber || blockNr == EarliestBlockNumber {
		height := core.GetBlockChain().Height()
		block = core.GetBlockChain().QueryBlock(height)
	} else {
		block = core.GetBlockChain().QueryBlock(uint64(blockNr))
	}

	if block == nil || uint64(index) >= uint64(len(block.Transactions)) {
		return nil
	}
	return newRPCTransaction(block.Transactions[index], block.Header.Hash, block.Header.Height, uint64(index))
}

// GetTransactionByBlockHashAndIndex returns the transaction for the given block hash and index.
func (s *EthAPIService) GetTransactionByBlockHashAndIndex(blockHash common.Hash, index utility.Uint) *RPCTransaction {
	block := core.GetBlockChain().QueryBlockByHash(blockHash)
	if block == nil || uint64(index) >= uint64(len(block.Transactions)) {
		return nil
	}
	return newRPCTransaction(block.Transactions[index], block.Header.Hash, block.Header.Height, uint64(index))
}

// Version returns the current ethereum protocol version.
// net_version
func (s *EthAPIService) Version() string {
	return common.NetworkId()
}

// Listening Returns true if client is actively listening for network connections.
// net_listening
func (s *EthAPIService) Listening() bool {
	return true
}

// ClientVersion returns the current client version.
// web3_clientVersion
func (api *EthAPIService) ClientVersion() string {
	return "Rangers/" + common.Version + "/centos-amd64/go1.17.3"
}

func validateTx(tx *eth_tx.Transaction) (common.Address, error) {
	var err error
	var sender common.Address

	// Make sure the transaction is signed properly
	signer := eth_tx.NewEIP155Signer(common.GetChainId(utility.MaxUint64))
	sender, err = eth_tx.Sender(signer, tx)
	if err != nil {
		return sender, err
	}
	// Transactions can't be negative. This may never happen using RLP decoded
	// transactions but may occur for transactions created using the RPC.
	if tx.Value().Sign() < 0 {
		return sender, ErrNegativeValue
	}
	// Check whether the init code size has been exceeded.
	if len(tx.Data()) > MaxInitCodeSize {
		return sender, fmt.Errorf("%w: code size %v limit %v", ErrMaxInitCodeSizeExceeded, len(tx.Data()), MaxInitCodeSize)
	}
	// Ensure the transaction has more gas than the bare minimum needed to cover
	// the transaction metadata
	intrGas, err := executor.IntrinsicGas(tx.Data(), tx.To() == nil)
	if err != nil {
		return sender, err
	}
	if tx.Gas() < intrGas {
		return sender, fmt.Errorf("%w: needed %v, allowed %v", executor.ErrIntrinsicGas, intrGas, tx.Gas())
	}

	// Ensure the transaction adheres to nonce ordering
	stateDB, err := middleware.AccountDBManagerInstance.GetAccountDBByHash(core.GetBlockChain().TopBlock().StateTree)
	if err != nil {
		return sender, fmt.Errorf("try again")
	}

	nextNonce := stateDB.GetNonce(sender)
	if tx.Nonce() < nextNonce {
		return sender, fmt.Errorf("%w: next nonce %v, tx nonce %v", vm.ErrNonceTooLow, nextNonce, tx.Nonce())
	}
	// Ensure the transactor has enough funds to cover the transaction costs
	balance := stateDB.GetBalance(sender)
	costFee := tx.Cost()
	if balance.Cmp(costFee) < 0 {
		return sender, fmt.Errorf("%w: balance %v, tx cost %v, overshot %v", executor.ErrInsufficientFunds, balance, costFee, new(big.Int).Sub(costFee, balance))
	}

	if service.GetTransactionPool().IsFull() {
		return sender, ErrTxPoolOverflow
	}
	return sender, err
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
		result.Input = common.FromHex(data.AbiData)

		transferValue, err := utility.StrToBigInt(data.TransferValue)
		if err == nil {
			result.Value = (*utility.Big)(transferValue)
		}

		gasLimit, err := strconv.ParseUint(data.GasLimit, 10, 64)
		if err == nil {
			result.Gas = utility.Uint64(gasLimit)
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

// -----------------------------------------------------------------------------
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
	var b *types.BlockHeader
	if height == 0 {
		b = core.GetBlockChain().TopBlock()
	} else {
		b = core.GetBlockChain().QueryBlockHeaderByHeight(height, true)
	}
	if nil == b {
		return nil
	}
	accountDB, _ = middleware.AccountDBManagerInstance.GetAccountDBByHash(b.StateTree)
	return
}

func getAccountDBByHash(hash common.Hash) (accountDB *account.AccountDB) {
	b := core.GetBlockChain().QueryBlockByHash(hash)
	if nil == b {
		return nil
	}
	accountDB, _ = middleware.AccountDBManagerInstance.GetAccountDBByHash(b.Header.StateTree)
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

func broadcastRawTx(rawTx string) (common.Hash, error) {
	url := common.LocalChainConfig.JsonRPCUrl
	method := "eth_sendRawTransaction"
	params := rawTx
	responseOBJ, err := network.JSONRPCPost(url, method, params)
	if err != nil {
		logger.Error("broadcast raw tx error:%v", err)
		return common.Hash{}, err
	}
	hashStr, ok := responseOBJ.Result.(string)
	if !ok {
		return common.Hash{}, errors.New("illegal return data")
	}
	hash := common.HexToHash(hashStr)
	return hash, nil
}

func adaptErrorOutput(err error, result []byte) error {
	reason := getRevertReason(result)
	if reason != "" {
		err = fmt.Errorf("execution reverted: %v", reason)
	}
	return err
}

func getRevertReason(result []byte) string {
	if len(result) < 36 {
		return ""
	}
	typ, err := abi.NewType("string")
	if err != nil {
		return ""
	}
	decoded, err := typ.Decode(result[36:])
	if err != nil {
		return ""
	}
	return decoded.(string)
}

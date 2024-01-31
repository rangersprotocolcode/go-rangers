package core

import (
	"bytes"
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/executor"
	"com.tuntun.rangers/node/src/middleware"
	"com.tuntun.rangers/node/src/middleware/log"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/service"
	"com.tuntun.rangers/node/src/utility"
	"com.tuntun.rangers/node/src/vm"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"
)

func preTest() {
	os.RemoveAll("storage0")

	common.Init(0, "0.ini", "dev")
	middleware.InitMiddleware()
	service.InitService()

	executor.InitExecutors()
	service.InitRewardCalculator(blockChainImpl, groupChainImpl, SyncProcessor)
	service.InitRefundManager(groupChainImpl, SyncProcessor)

	common.SetBlockHeight(10000)

	logger = log.GetLoggerByIndex(log.CoreLogConfig, strconv.Itoa(common.InstanceIndex))
	txLogger = log.GetLoggerByIndex(log.TxLogConfig, strconv.Itoa(common.InstanceIndex))
	syncLogger = log.GetLoggerByIndex(log.SyncLogConfig, strconv.Itoa(common.InstanceIndex))
	syncHandleLogger = log.GetLoggerByIndex(log.SyncHandleLogConfig, strconv.Itoa(common.InstanceIndex))
	rewardLog = log.GetLoggerByIndex(log.RewardLogConfig, strconv.Itoa(common.InstanceIndex))

	vm.InitVM()
}

var (
	money, _ = utility.StrToBigInt("10661998")
	richAddr = common.HexToAddress("0x01")
)

const testContractData = "608060405234801561001057600080fd5b50610113806100206000396000f3fe6080604052348015600f57600080fd5b506004361060325760003560e01c80631003e2d21460375780631f7b6d32146048575b600080fd5b6046604236600460c5565b605d565b005b60005460405190815260200160405180910390f35b600080546001810182559080527f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563018190556040518181527fe7031cd6956b2659170d686871156b5a86ec38e9071dfc7e6863f24e5debc10f9060200160405180910390a150565b60006020828403121560d657600080fd5b503591905056fea2646970667358221220e817b443aba8374c91a43c77972eff026499557eb6382c7d874b09bc17ee81a864736f6c634300080c0033"

// one contract tx
func TestVMExecutor_Execute2(t *testing.T) {
	defer func() {
		service.Close()
		middleware.Close()
		log.Close()

		os.RemoveAll("0.ini")
		os.RemoveAll("logs")

		err := os.RemoveAll("storage0")
		if nil != err {
			t.Fatal(err)
		}
	}()
	preTest()

	var data types.ContractData
	data.AbiData = testContractData
	str, _ := json.Marshal(data)

	tx := &types.Transaction{
		Source: richAddr.String(),
		Type:   types.TransactionTypeContract,
		Data:   string(str),
	}

	block := generateBlock()
	block.Transactions = append(block.Transactions, tx)
	accountdb, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vm := newVMExecutor(accountdb, block, "casting")
	state, _, txs, receipts := vm.Execute()

	blockVerify := generateBlock()
	blockVerify.Transactions = txs
	accountdb, _ = middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vmVerify := newVMExecutor(accountdb, blockVerify, "verifying")
	stateVerify, _, txsVerify, receiptsVerify := vmVerify.Execute()

	msg := checkResult(state, txs, receipts, stateVerify, txsVerify, receiptsVerify)
	if "success" != msg {
		t.Fatalf(msg)
	}
}

// one contract
// call it
func TestVMExecutor_Execute3(t *testing.T) {
	defer func() {
		service.Close()
		middleware.Close()
		log.Close()

		os.RemoveAll("0.ini")
		os.RemoveAll("logs")

		err := os.RemoveAll("storage0")
		if nil != err {
			t.Fatal(err)
		}
	}()
	preTest()

	var data types.ContractData
	data.AbiData = testContractData
	str, _ := json.Marshal(data)
	tx := &types.Transaction{
		Source: richAddr.String(),
		Type:   types.TransactionTypeContract,
		Data:   string(str),
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx1 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeContract,
		Data:   string(str),
		Nonce:  1,
	}

	block := generateBlock()
	block.Transactions = []*types.Transaction{tx, tx1}
	accountdb, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vm := newVMExecutor(accountdb, block, "verifying")
	state, _, txs, receipts := vm.Execute()
	if 2 != len(txs) || 2 != len(receipts) {
		t.Fatalf("fail to execute")
	}
	if receipts[1].Status != 1 {
		t.Fatalf("fail to check nonce")
	}

	blockVerify := generateBlock()
	blockVerify.Transactions = txs
	accountdb, _ = middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vmVerify := newVMExecutor(accountdb, blockVerify, "verifying")
	stateVerify, _, txsVerify, receiptsVerify := vmVerify.Execute()

	msg := checkResult(state, txs, receipts, stateVerify, txsVerify, receiptsVerify)
	if "success" != msg {
		t.Fatalf(msg)
	}
}

// one contract
// call it with same nonce
func TestVMExecutor_Execute4(t *testing.T) {
	defer func() {
		service.Close()
		middleware.Close()
		log.Close()

		os.RemoveAll("0.ini")
		os.RemoveAll("logs")

		err := os.RemoveAll("storage0")
		if nil != err {
			t.Fatal(err)
		}
	}()
	preTest()

	var data types.ContractData
	data.AbiData = testContractData
	str, _ := json.Marshal(data)
	tx := &types.Transaction{
		Source: richAddr.String(),
		Type:   types.TransactionTypeContract,
		Data:   string(str),
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx1 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeContract,
		Data:   string(str),
	}

	block := generateBlock()
	block.Transactions = []*types.Transaction{tx, tx1}
	accountdb, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vm := newVMExecutor(accountdb, block, "verifying")
	state, _, txs, receipts := vm.Execute()
	if 2 != len(txs) || 2 != len(receipts) {
		t.Fatalf("fail to execute")
	}
	if receipts[1].Status != 0 {
		t.Fatalf("fail to check nonce")
	}

	blockVerify := generateBlock()
	blockVerify.Transactions = txs
	accountdb, _ = middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vmVerify := newVMExecutor(accountdb, blockVerify, "verifying")
	stateVerify, _, txsVerify, receiptsVerify := vmVerify.Execute()

	msg := checkResult(state, txs, receipts, stateVerify, txsVerify, receiptsVerify)
	if "success" != msg {
		t.Fatalf(msg)
	}
}

// jsonrpc
// one contract
// call it
func TestVMExecutor_Execute5(t *testing.T) {
	defer func() {
		service.Close()
		middleware.Close()
		log.Close()

		os.RemoveAll("0.ini")
		os.RemoveAll("logs")

		err := os.RemoveAll("storage0")
		if nil != err {
			t.Fatal(err)
		}
	}()
	preTest()

	var data types.ContractData
	data.AbiData = testContractData
	str, _ := json.Marshal(data)
	tx := &types.Transaction{
		Source: richAddr.String(),
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx1 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
		Nonce:  1,
	}

	block := generateBlock()
	block.Transactions = []*types.Transaction{tx, tx1}
	accountdb, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vm := newVMExecutor(accountdb, block, "verifying")
	state, _, txs, receipts := vm.Execute()
	if 2 != len(txs) || 2 != len(receipts) {
		t.Fatalf("fail to execute")
	}
	if receipts[1].Status != 1 {
		t.Fatalf("fail to check nonce")
	}

	blockVerify := generateBlock()
	blockVerify.Transactions = txs
	accountdb, _ = middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vmVerify := newVMExecutor(accountdb, blockVerify, "verifying")
	stateVerify, _, txsVerify, receiptsVerify := vmVerify.Execute()

	msg := checkResult(state, txs, receipts, stateVerify, txsVerify, receiptsVerify)
	if "success" != msg {
		t.Fatalf(msg)
	}
}

// jsonrpc
// one contract
// call it with same nonce
func TestVMExecutor_Execute6(t *testing.T) {
	defer func() {
		service.Close()
		middleware.Close()
		log.Close()

		os.RemoveAll("0.ini")
		os.RemoveAll("logs")

		err := os.RemoveAll("storage0")
		if nil != err {
			t.Fatal(err)
		}
	}()
	preTest()

	var data types.ContractData
	data.AbiData = testContractData
	str, _ := json.Marshal(data)
	tx := &types.Transaction{
		Source: richAddr.String(),
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx1 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
	}

	block := generateBlock()
	block.Transactions = []*types.Transaction{tx, tx1}
	accountdb, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vm := newVMExecutor(accountdb, block, "verifying")
	state, _, txs, receipts := vm.Execute()
	if 1 != len(txs) || 1 != len(receipts) {
		t.Fatalf("fail to execute")
	}

	blockVerify := generateBlock()
	blockVerify.Transactions = txs
	accountdb, _ = middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vmVerify := newVMExecutor(accountdb, blockVerify, "verifying")
	stateVerify, _, txsVerify, receiptsVerify := vmVerify.Execute()

	msg := checkResult(state, txs, receipts, stateVerify, txsVerify, receiptsVerify)
	if "success" != msg {
		t.Fatalf(msg)
	}
}

// jsonrpc one contract
// ws call it
func TestVMExecutor_Execute7(t *testing.T) {
	defer func() {
		service.Close()
		middleware.Close()
		log.Close()

		os.RemoveAll("0.ini")
		os.RemoveAll("logs")

		err := os.RemoveAll("storage0")
		if nil != err {
			t.Fatal(err)
		}
	}()
	preTest()

	var data types.ContractData
	data.AbiData = testContractData
	str, _ := json.Marshal(data)
	tx := &types.Transaction{
		Source: richAddr.String(),
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx1 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeContract,
		Data:   string(str),
		Nonce:  1,
	}

	block := generateBlock()
	block.Transactions = []*types.Transaction{tx, tx1}
	accountdb, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vm := newVMExecutor(accountdb, block, "verifying")
	state, _, txs, receipts := vm.Execute()
	if 2 != len(txs) || 2 != len(receipts) {
		t.Fatalf("fail to execute")
	}
	if receipts[1].Status != 1 {
		t.Fatalf("fail to check nonce")
	}

	blockVerify := generateBlock()
	blockVerify.Transactions = txs
	accountdb, _ = middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vmVerify := newVMExecutor(accountdb, blockVerify, "verifying")
	stateVerify, _, txsVerify, receiptsVerify := vmVerify.Execute()

	msg := checkResult(state, txs, receipts, stateVerify, txsVerify, receiptsVerify)
	if "success" != msg {
		t.Fatalf(msg)
	}
}

// jsonrpc one contract
// ws call it with same nonce
func TestVMExecutor_Execute8(t *testing.T) {
	defer func() {
		service.Close()
		middleware.Close()
		log.Close()

		os.RemoveAll("0.ini")
		os.RemoveAll("logs")

		err := os.RemoveAll("storage0")
		if nil != err {
			t.Fatal(err)
		}
	}()
	preTest()

	var data types.ContractData
	data.AbiData = testContractData
	str, _ := json.Marshal(data)
	tx := &types.Transaction{
		Source: richAddr.String(),
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx1 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeContract,
		Data:   string(str),
	}

	block := generateBlock()
	block.Transactions = []*types.Transaction{tx, tx1}
	accountdb, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vm := newVMExecutor(accountdb, block, "verifying")
	state, _, txs, receipts := vm.Execute()
	if 2 != len(txs) || 2 != len(receipts) {
		t.Fatalf("fail to execute")
	}
	if receipts[1].Status != 0 {
		t.Fatalf("fail to check nonce")
	}

	blockVerify := generateBlock()
	blockVerify.Transactions = txs
	accountdb, _ = middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vmVerify := newVMExecutor(accountdb, blockVerify, "verifying")
	stateVerify, _, txsVerify, receiptsVerify := vmVerify.Execute()

	msg := checkResult(state, txs, receipts, stateVerify, txsVerify, receiptsVerify)
	if "success" != msg {
		t.Fatalf(msg)
	}
}

// jsonrpc one contract
// ws call it
// jsonrpc call it
func TestVMExecutor_Execute9(t *testing.T) {
	defer func() {
		service.Close()
		middleware.Close()
		log.Close()

		os.RemoveAll("0.ini")
		os.RemoveAll("logs")

		err := os.RemoveAll("storage0")
		if nil != err {
			t.Fatal(err)
		}
	}()
	preTest()

	var data types.ContractData
	data.AbiData = testContractData
	str, _ := json.Marshal(data)
	tx := &types.Transaction{
		Source: richAddr.String(),
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx1 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeContract,
		Data:   string(str),
		Nonce:  1,
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx2 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
		Nonce:  2,
	}

	block := generateBlock()
	block.Transactions = []*types.Transaction{tx, tx1, tx2}
	accountdb, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vm := newVMExecutor(accountdb, block, "verifying")
	state, _, txs, receipts := vm.Execute()
	if 3 != len(txs) || 3 != len(receipts) {
		t.Fatalf("fail to execute")
	}

	blockVerify := generateBlock()
	blockVerify.Transactions = txs
	accountdb, _ = middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vmVerify := newVMExecutor(accountdb, blockVerify, "verifying")
	stateVerify, _, txsVerify, receiptsVerify := vmVerify.Execute()

	msg := checkResult(state, txs, receipts, stateVerify, txsVerify, receiptsVerify)
	if "success" != msg {
		t.Fatalf(msg)
	}
}

// jsonrpc one contract
// ws call it
// jsonrpc call it
// ws call it
func TestVMExecutor_Execute10(t *testing.T) {
	defer func() {
		service.Close()
		middleware.Close()
		log.Close()

		os.RemoveAll("0.ini")
		os.RemoveAll("logs")

		err := os.RemoveAll("storage0")
		if nil != err {
			t.Fatal(err)
		}
	}()
	preTest()

	var data types.ContractData
	data.AbiData = testContractData
	str, _ := json.Marshal(data)
	tx := &types.Transaction{
		Source: richAddr.String(),
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx1 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeContract,
		Data:   string(str),
		Nonce:  1,
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx2 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
		Nonce:  2,
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx3 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeContract,
		Data:   string(str),
		Nonce:  3,
	}

	block := generateBlock()
	block.Transactions = []*types.Transaction{tx, tx1, tx2, tx3}
	accountdb, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vm := newVMExecutor(accountdb, block, "verifying")
	state, _, txs, receipts := vm.Execute()
	if 4 != len(txs) || 4 != len(receipts) {
		t.Fatalf("fail to execute")
	}

	blockVerify := generateBlock()
	blockVerify.Transactions = txs
	accountdb, _ = middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vmVerify := newVMExecutor(accountdb, blockVerify, "verifying")
	stateVerify, _, txsVerify, receiptsVerify := vmVerify.Execute()

	msg := checkResult(state, txs, receipts, stateVerify, txsVerify, receiptsVerify)
	if "success" != msg {
		t.Fatalf(msg)
	}
}

// jsonrpc one contract
// ws call it with same nonce
// jsonrpc call it
// ws call it
func TestVMExecutor_Execute11(t *testing.T) {
	defer func() {
		service.Close()
		middleware.Close()
		log.Close()

		os.RemoveAll("0.ini")
		os.RemoveAll("logs")

		err := os.RemoveAll("storage0")
		if nil != err {
			t.Fatal(err)
		}
	}()
	preTest()

	var data types.ContractData
	data.AbiData = testContractData
	str, _ := json.Marshal(data)
	tx := &types.Transaction{
		Source: richAddr.String(),
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx1 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeContract,
		Data:   string(str),
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx2 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
		Nonce:  1,
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx3 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeContract,
		Data:   string(str),
		Nonce:  2,
	}

	block := generateBlock()
	block.Transactions = []*types.Transaction{tx, tx1, tx2, tx3}
	accountdb, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vm := newVMExecutor(accountdb, block, "verifying")
	state, evicted, txs, receipts := vm.Execute()
	if 3 != len(txs) || 3 != len(receipts) || 1 != len(evicted) {
		t.Fatalf("fail to execute")
	}

	blockVerify := generateBlock()
	blockVerify.Transactions = txs
	accountdb, _ = middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vmVerify := newVMExecutor(accountdb, blockVerify, "verifying")
	stateVerify, _, txsVerify, receiptsVerify := vmVerify.Execute()

	msg := checkResult(state, txs, receipts, stateVerify, txsVerify, receiptsVerify)
	if "success" != msg {
		t.Fatalf(msg)
	}
}

// jsonrpc one contract
// ws call it
// jsonrpc call it with same nonce
// ws call it
func TestVMExecutor_Execute12(t *testing.T) {
	defer func() {
		service.Close()
		middleware.Close()
		log.Close()

		os.RemoveAll("0.ini")
		os.RemoveAll("logs")

		err := os.RemoveAll("storage0")
		if nil != err {
			t.Fatal(err)
		}
	}()
	preTest()

	var data types.ContractData
	data.AbiData = testContractData
	str, _ := json.Marshal(data)
	tx := &types.Transaction{
		Source: richAddr.String(),
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx1 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeContract,
		Data:   string(str),
		Nonce:  1,
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx2 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
		Nonce:  1,
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx3 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeContract,
		Data:   string(str),
		Nonce:  2,
	}

	block := generateBlock()
	block.Transactions = []*types.Transaction{tx, tx1, tx2, tx3}
	accountdb, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vm := newVMExecutor(accountdb, block, "verifying")
	state, evicted, txs, receipts := vm.Execute()
	if 3 != len(txs) || 3 != len(receipts) || 1 != len(evicted) {
		t.Fatalf("fail to execute")
	}

	blockVerify := generateBlock()
	blockVerify.Transactions = txs
	accountdb, _ = middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vmVerify := newVMExecutor(accountdb, blockVerify, "verifying")
	stateVerify, _, txsVerify, receiptsVerify := vmVerify.Execute()

	msg := checkResult(state, txs, receipts, stateVerify, txsVerify, receiptsVerify)
	if "success" != msg {
		t.Fatalf(msg)
	}
}

// jsonrpc one contract
// ws call it with same nonce
// jsonrpc call it with same nonce
// ws call it
func TestVMExecutor_Execute13(t *testing.T) {
	defer func() {
		service.Close()
		middleware.Close()
		log.Close()

		os.RemoveAll("0.ini")
		os.RemoveAll("logs")

		err := os.RemoveAll("storage0")
		if nil != err {
			t.Fatal(err)
		}
	}()
	preTest()

	var data types.ContractData
	data.AbiData = testContractData
	str, _ := json.Marshal(data)
	tx := &types.Transaction{
		Source: richAddr.String(),
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx1 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeContract,
		Data:   string(str),
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx2 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx3 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeContract,
		Data:   string(str),
		Nonce:  1,
	}

	block := generateBlock()
	block.Transactions = []*types.Transaction{tx, tx1, tx2, tx3}
	accountdb, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vm := newVMExecutor(accountdb, block, "verifying")
	state, evicted, txs, receipts := vm.Execute()
	if 3 != len(txs) || 3 != len(receipts) || 1 != len(evicted) {
		t.Fatalf("fail to execute")
	}
	if receipts[1].Status != 0 {
		t.Fatalf("fail to check nonce")
	}

	blockVerify := generateBlock()
	blockVerify.Transactions = txs
	accountdb, _ = middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vmVerify := newVMExecutor(accountdb, blockVerify, "verifying")
	stateVerify, _, txsVerify, receiptsVerify := vmVerify.Execute()

	msg := checkResult(state, txs, receipts, stateVerify, txsVerify, receiptsVerify)
	if "success" != msg {
		t.Fatalf(msg)
	}
}

// jsonrpc one contract
// ws call it
// jsonrpc call it
// ws call it
// jsonrpc call it
func TestVMExecutor_Execute14(t *testing.T) {
	defer func() {
		service.Close()
		middleware.Close()
		log.Close()

		os.RemoveAll("0.ini")
		os.RemoveAll("logs")

		err := os.RemoveAll("storage0")
		if nil != err {
			t.Fatal(err)
		}
	}()
	preTest()

	var data types.ContractData
	data.AbiData = testContractData
	str, _ := json.Marshal(data)
	tx := &types.Transaction{
		Source: richAddr.String(),
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx1 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeContract,
		Data:   string(str),
		Hash:   common.HexToHash("0x01"),
		Nonce:  1,
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx2 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
		Hash:   common.HexToHash("0x02"),
		Nonce:  2,
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx3 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeContract,
		Data:   string(str),
		Nonce:  3,
		Hash:   common.HexToHash("0x03"),
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx4 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
		Hash:   common.HexToHash("0x04"),
		Nonce:  4,
	}

	block := generateBlock()
	block.Transactions = []*types.Transaction{tx, tx1, tx2, tx3, tx4}
	accountdb, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vm := newVMExecutor(accountdb, block, "verifying")
	state, _, txs, receipts := vm.Execute()
	if 5 != len(txs) || 5 != len(receipts) {
		t.Fatalf("fail to execute")
	}

	blockVerify := generateBlock()
	blockVerify.Transactions = txs
	accountdb, _ = middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vmVerify := newVMExecutor(accountdb, blockVerify, "verifying")
	stateVerify, _, txsVerify, receiptsVerify := vmVerify.Execute()

	msg := checkResult(state, txs, receipts, stateVerify, txsVerify, receiptsVerify)
	if "success" != msg {
		t.Fatalf(msg)
	}
}

// jsonrpc one contract
// ws call it
// jsonrpc call it
// ws call it
// jsonrpc call it with wrong nonce
func TestVMExecutor_Execute15(t *testing.T) {
	defer func() {
		service.Close()
		middleware.Close()
		log.Close()

		os.RemoveAll("0.ini")
		os.RemoveAll("logs")

		err := os.RemoveAll("storage0")
		if nil != err {
			t.Fatal(err)
		}
	}()
	preTest()

	var data types.ContractData
	data.AbiData = testContractData
	str, _ := json.Marshal(data)
	tx := &types.Transaction{
		Source: richAddr.String(),
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx1 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeContract,
		Data:   string(str),
		Hash:   common.HexToHash("0x01"),
		Nonce:  1,
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx2 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
		Hash:   common.HexToHash("0x02"),
		Nonce:  2,
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx3 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeContract,
		Data:   string(str),
		Nonce:  3,
		Hash:   common.HexToHash("0x03"),
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx4 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
		Hash:   common.HexToHash("0x04"),
		Nonce:  4444,
	}

	block := generateBlock()
	block.Transactions = []*types.Transaction{tx, tx1, tx2, tx3, tx4}
	accountdb, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vm := newVMExecutor(accountdb, block, "verifying")
	state, evicted, txs, receipts := vm.Execute()
	if 4 != len(txs) || 4 != len(receipts) {
		t.Fatalf("fail to execute")
	}
	if bytes.Compare(evicted[0].Bytes(), tx4.Hash.Bytes()) != 0 {
		t.Fatalf("fail to evicted")
	}

	blockVerify := generateBlock()
	blockVerify.Transactions = txs
	accountdb, _ = middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vmVerify := newVMExecutor(accountdb, blockVerify, "verifying")
	stateVerify, _, txsVerify, receiptsVerify := vmVerify.Execute()

	msg := checkResult(state, txs, receipts, stateVerify, txsVerify, receiptsVerify)
	if "success" != msg {
		t.Fatalf(msg)
	}
}

// jsonrpc one contract
// ws call it
// jsonrpc call it
// ws call it
// jsonrpc call it with wrong nonce
// abnormal data
func TestVMExecutor_Execute16(t *testing.T) {
	defer func() {
		service.Close()
		middleware.Close()
		log.Close()

		os.RemoveAll("0.ini")
		os.RemoveAll("logs")

		err := os.RemoveAll("storage0")
		if nil != err {
			t.Fatal(err)
		}
	}()
	preTest()

	var data types.ContractData
	data.AbiData = testContractData
	str, _ := json.Marshal(data)
	tx := &types.Transaction{
		Source: richAddr.String(),
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx1 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeContract,
		Data:   string(str),
		Hash:   common.HexToHash("0x01"),
		Nonce:  1,
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx2 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
		Hash:   common.HexToHash("0x02"),
		Nonce:  2,
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx3 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeContract,
		Data:   string(str),
		Nonce:  3,
		Hash:   common.HexToHash("0x03"),
	}

	data.AbiData = "0x1003e2d20000000000000000000000000000000000000000000000000000000000000462"
	str, _ = json.Marshal(data)
	tx4 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeETHTX,
		Data:   string(str),
		Hash:   common.HexToHash("0x04"),
		Nonce:  4444,
	}

	tx5 := &types.Transaction{
		Source: richAddr.String(),
		Target: "0x0742cb5613c40c305fdea246be6304dbce829c3c",
		Type:   types.TransactionTypeETHTX,
		Data:   "1",
		Hash:   common.HexToHash("0x05"),
		Nonce:  4,
	}

	block := generateBlock()
	block.Transactions = []*types.Transaction{tx, tx1, tx2, tx3, tx4, tx5}
	accountdb, _ := middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vm := newVMExecutor(accountdb, block, "verifying")
	state, evicted, txs, receipts := vm.Execute()
	if 5 != len(txs) || 5 != len(receipts) {
		t.Fatalf("fail to execute")
	}
	if bytes.Compare(evicted[0].Bytes(), tx4.Hash.Bytes()) != 0 {
		t.Fatalf("fail to evicted")
	}

	blockVerify := generateBlock()
	blockVerify.Transactions = txs
	accountdb, _ = middleware.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	accountdb.SetBalance(richAddr, money)
	vmVerify := newVMExecutor(accountdb, blockVerify, "verifying")
	stateVerify, _, txsVerify, receiptsVerify := vmVerify.Execute()

	msg := checkResult(state, txs, receipts, stateVerify, txsVerify, receiptsVerify)
	if "success" != msg {
		t.Fatalf(msg)
	}
}

func checkResult(state common.Hash, txs []*types.Transaction, receipts []*types.Receipt, stateVerify common.Hash, txsVerify []*types.Transaction, receiptsVerify []*types.Receipt) string {
	if state != stateVerify {
		return fmt.Sprintf("state error,%s %s", state.String(), stateVerify.String())
	}

	if len(txs) != len(txsVerify) {
		return fmt.Sprintf("error txs")
	}
	for i := range txs {
		s1 := txs[i].ToTxJson().ToString()
		s2 := txsVerify[i].ToTxJson().ToString()

		if s1 != s2 {
			return fmt.Sprintf("error txs, %s %s", s1, s2)
		}
	}

	if len(receipts) != len(receiptsVerify) {
		return fmt.Sprintf("error receipts")
	}
	for i := range receipts {
		s1, _ := json.Marshal(receipts[i])
		s2, _ := json.Marshal(receiptsVerify[i])
		if bytes.Compare(s1, s2) != 0 {
			return fmt.Sprintf("error receipts, %s %s", s1, s2)
		}
	}

	return "success"
}

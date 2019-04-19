package core

import (
	"time"
	"math/big"

	"x/src/common"
	"x/src/middleware/types"
	"x/src/storage/account"

	"github.com/vmihailenco/msgpack"
	"bytes"
	"x/src/statemachine"
	"strconv"
	"encoding/json"
)

const TransactionGasCost = 1000
const CodeBytePrice = 0.3814697265625
const MaxCastBlockTime = time.Second * 3

type VMExecutor struct {
	bc BlockChain
}

func NewVMExecutor(bc BlockChain) *VMExecutor {
	return &VMExecutor{
		bc: bc,
	}
}

func (executor *VMExecutor) Execute(accountdb *account.AccountDB, block *types.Block, height uint64, situation string) (common.Hash, []common.Hash, []*types.Transaction, []*types.Receipt, error, []*types.TransactionError) {
	beginTime := time.Now()
	receipts := make([]*types.Receipt, 0)
	transactions := make([]*types.Transaction, 0)
	evictedTxs := make([]common.Hash, 0)
	errs := make([]*types.TransactionError, len(block.Transactions))

	for i, transaction := range block.Transactions {
		executeTime := time.Now()
		if situation == "casting" && executeTime.Sub(beginTime) > MaxCastBlockTime {
			logger.Infof("Cast block execute tx time out!Tx hash:%s ", transaction.Hash.String())
			break
		}
		//logger.Debugf("VMExecutor Execute %v,type:%d", transaction.Hash, transaction.Type)
		var success = false
		var contractAddress common.Address
		var logs []*types.Log
		var err *types.TransactionError
		var cumulativeGasUsed uint64
		castor := common.BytesToAddress(block.Header.Castor)

		if !executor.validateNonce(accountdb, transaction) {
			evictedTxs = append(evictedTxs, transaction.Hash)
			continue
		}

		switch transaction.Type {
		case types.TransactionTypeTransfer:
			success, err, cumulativeGasUsed = executor.executeTransferTx(accountdb, transaction, castor)
			//case types.TransactionTypeContractCreate:
			//	success, err, cumulativeGasUsed, contractAddress = executor.executeContractCreateTx(accountdb, transaction, castor, block)
			//case types.TransactionTypeContractCall:
			//	success, err, cumulativeGasUsed, logs = executor.executeContractCallTx(accountdb, transaction, castor, block)
		case types.TransactionTypeBonus:
			success = executor.executeBonusTx(accountdb, transaction, castor)
		case types.TransactionTypeMinerApply:
			success = executor.executeMinerApplyTx(accountdb, transaction, height, situation, castor)
		case types.TransactionTypeMinerAbort:
			success = executor.executeMinerAbortTx(accountdb, transaction, height, castor, situation)
		case types.TransactionTypeMinerRefund:
			success = executor.executeMinerRefundTx(accountdb, transaction, height, castor, situation)

		case types.TransactionTypeOperatorEvent:

			// 已经执行过了，则直接返回true
			if nil == GetBlockChain().GetTransactionPool().GetExecuted(transaction.Hash) {
				nonce := statemachine.Docker.Nonce(transaction.Target)
				payload := string(transaction.Data)
				statemachine.Docker.Process(transaction.Target, "operator", strconv.Itoa(nonce+1), payload)

			}

			success = true
		case types.TransactionUpdateOperatorEvent:
			success = true
			data := make([]types.UserData, 0)
			if err := json.Unmarshal([]byte(transaction.Data), &data); err != nil {
				success = false

			} else {
				if nil != data && 0 != len(data) {
					snapshot := accountdb.Snapshot()
					for _, user := range data {
						if !UpdateAsset(user, transaction.Target, accountdb) {
							accountdb.RevertToSnapshot(snapshot)
							success = false
							break
						}

						if nil != common.DefaultLogger && "casting" != situation {
							sub := accountdb.GetSubAccount(common.HexToAddress(user.Address), transaction.Target)
							subData, _ := json.Marshal(sub)
							common.DefaultLogger.Errorf("%s. success to execute tx, data: %s, subAsset: %s", situation, transaction.Data, subData)
						}
					}

				}

			}

		}

		if !success {
			evictedTxs = append(evictedTxs, transaction.Hash)
		}
		if success || transaction.Type != types.TransactionTypeBonus {
			transactions = append(transactions, transaction)
			receipt := types.NewReceipt(nil, !success, cumulativeGasUsed)
			receipt.Logs = logs
			receipt.TxHash = transaction.Hash
			receipt.ContractAddress = contractAddress
			receipts = append(receipts, receipt)
			errs[i] = err
			if transaction.Source != nil {
				accountdb.SetNonce(*transaction.Source, transaction.Nonce)
			}
		}
	}
	accountdb.AddBalance(common.BytesToAddress(block.Header.Castor), consensusHelper.ProposalBonus())

	state := accountdb.IntermediateRoot(true)
	logger.Debugf("VMExecutor End Execute State %s", state.Hex())
	return state, evictedTxs, transactions, receipts, nil, errs
}

func (executor *VMExecutor) validateNonce(accountdb *account.AccountDB, transaction *types.Transaction) bool {
	//if transaction.Type == types.TransactionTypeBonus || types.IsTestTransaction(transaction) {
	//	return true
	//}
	//nonce := accountdb.GetNonce(*transaction.Source)
	//if transaction.Nonce != nonce+1 {
	//	logger.Infof("Tx nonce error! Hash:%s,Source:%s,expect nonce:%d,real nonce:%d ", transaction.Hash.String(), transaction.Source.GetHexString(), nonce+1, transaction.Nonce)
	//	return false
	//}
	return true
}

func (executor *VMExecutor) executeTransferTx(accountdb *account.AccountDB, transaction *types.Transaction, castor common.Address) (success bool, err *types.TransactionError, cumulativeGasUsed uint64) {
	//logger.Debugf("VMExecutor Execute Transfer Source:%s Target:%s Value:%d Height:%d Type:%s,Gas:%d,Success:%t", transaction.Source.GetHexString(), transaction.Target.GetHexString(), transaction.Value, height, mark,cumulativeGasUsed,success)
	return success, err, cumulativeGasUsed
}

//func (executor *VMExecutor) executeContractCreateTx(accountdb *account.AccountDB, transaction *types.Transaction, castor common.Address, block *types.Block) (success bool, err *types.TransactionError, cumulativeGasUsed uint64, contractAddress common.Address) {
//	success = true
//	txExecuteGasFee := big.NewInt(int64(transaction.GasPrice * TransactionGasCost))
//	gasLimit := transaction.GasLimit
//	gasLimitFee := new(big.Int).SetUint64(transaction.GasLimit * transaction.GasPrice)
//
//	if canTransfer(accountdb, *transaction.Source, gasLimitFee, txExecuteGasFee) {
//		accountdb.SubBalance(*transaction.Source, txExecuteGasFee)
//		accountdb.AddBalance(castor, txExecuteGasFee)
//
//		accountdb.SubBalance(*transaction.Source, gasLimitFee)
//		controller := vm.NewController(accountdb, BlockChainImpl, block.Header, transaction, common.GlobalConf.GetString("tvm", "pylib", "lib"))
//		snapshot := controller.AccountDB.Snapshot()
//		contractAddress, err = createContract(accountdb, transaction)
//		if err != nil {
//			logger.Debugf("ContractCreate tx %s execute error:%s ", transaction.Hash.String(), err.Message)
//			success = false
//			controller.AccountDB.RevertToSnapshot(snapshot)
//		} else {
//			deploySpend := uint64(float32(len(transaction.Data)) * CodeBytePrice)
//			if gasLimit < deploySpend {
//				success = false
//				err = types.TxErrorDeployGasNotEnough
//				controller.AccountDB.RevertToSnapshot(snapshot)
//			} else {
//				controller.GasLeft -= deploySpend
//				contract := tvm.LoadContract(contractAddress)
//				errorCode, errorMsg := controller.Deploy(transaction.Source, contract)
//				if errorCode != 0 {
//					success = false
//					err = types.NewTransactionError(errorCode, errorMsg)
//					controller.AccountDB.RevertToSnapshot(snapshot)
//				} else {
//					logger.Debugf("Contract create success! Tx hash:%s, contract addr:%s", transaction.Hash.String(), contractAddress.String())
//				}
//			}
//		}
//		gasLeft := controller.GetGasLeft()
//		returnFee := new(big.Int).SetUint64(gasLeft * transaction.GasPrice)
//		accountdb.AddBalance(*transaction.Source, returnFee)
//
//		cumulativeGasUsed = gasLimit - gasLeft + TransactionGasCost
//	} else {
//		success = false
//		err = types.TxErrorBalanceNotEnough
//		logger.Infof("ContractCreate balance not enough! transaction %s source %s  ", transaction.Hash.String(), transaction.Source.String())
//	}
//	//logger.Debugf("VMExecutor Execute ContractCreate Transaction %s,success:%t", transaction.Hash.Hex(),success)
//	return success, err, cumulativeGasUsed, contractAddress
//}
//
//func (executor *VMExecutor) executeContractCallTx(accountdb *account.AccountDB, transaction *types.Transaction, castor common.Address, block *types.Block) (success bool, err *types.TransactionError, cumulativeGasUsed uint64, logs []*types.Log) {
//	success = true
//	transferAmount := new(big.Int).SetUint64(transaction.Value)
//	txExecuteFee := big.NewInt(int64(transaction.GasPrice * TransactionGasCost))
//	gasLimit := transaction.GasLimit
//	gasLimitFee := new(big.Int).SetUint64(transaction.GasLimit * transaction.GasPrice)
//
//	totalAmount := new(big.Int).Add(transferAmount, gasLimitFee)
//	if canTransfer(accountdb, *transaction.Source, totalAmount, txExecuteFee) {
//		accountdb.SubBalance(*transaction.Source, txExecuteFee)
//		accountdb.AddBalance(castor, txExecuteFee)
//
//		accountdb.SubBalance(*transaction.Source, gasLimitFee)
//		controller := tvm.NewController(accountdb, BlockChainImpl, block.Header, transaction, common.GlobalConf.GetString("tvm", "pylib", "lib"))
//		contract := tvm.LoadContract(*transaction.Target)
//		if contract.Code == "" {
//			err = types.NewTransactionError(types.TxErrorCode_NO_CODE, fmt.Sprintf(types.NO_CODE_ERROR_MSG, *transaction.Target))
//			success = false
//		} else {
//			snapshot := controller.AccountDB.Snapshot()
//			var success bool
//			success, logs, err = controller.ExecuteAbi(transaction.Source, contract, string(transaction.Data))
//			if !success {
//				controller.AccountDB.RevertToSnapshot(snapshot)
//				success = false
//			} else {
//				accountdb.SubBalance(*transaction.Source, transferAmount)
//				accountdb.AddBalance(*contract.ContractAddress, transferAmount)
//			}
//		}
//		gasLeft := controller.GetGasLeft()
//		returnFee := new(big.Int).SetUint64(gasLeft * transaction.GasPrice)
//		accountdb.AddBalance(*transaction.Source, returnFee)
//
//		cumulativeGasUsed = gasLimit - gasLeft + TransactionGasCost
//	} else {
//		success = false
//		err = types.TxErrorBalanceNotEnough
//	}
//	logger.Debugf("VMExecutor Execute ContractCall Transaction %s,success:%t", transaction.Hash.Hex(), success)
//	return success, err, cumulativeGasUsed, logs
//}

func (executor *VMExecutor) executeBonusTx(accountdb *account.AccountDB, transaction *types.Transaction, castor common.Address) (success bool) {
	//logger.Debugf("VMExecutor Execute Bonus Transaction:%s Group:%s,Success:%t", common.BytesToHash(transaction.Data).Hex(), common.BytesToHash(groupId).ShortS(),success)
	return true
}

func (executor *VMExecutor) executeMinerApplyTx(accountdb *account.AccountDB, transaction *types.Transaction, height uint64, mark string, castor common.Address) (success bool) {
	logger.Debugf("Execute miner apply tx:%s,source: %v\n", transaction.Hash.String(), transaction.Source.GetHexString())
	success = false
	if len(transaction.Data) == 0 {
		logger.Debugf("VMExecutor Execute MinerApply Fail(Tx data is nil) Source:%s Height:%d Type:%s", transaction.Source.GetHexString(), height, mark)
		return success
	}

	data := common.FromHex(string(transaction.Data))
	var miner types.Miner
	msgpack.Unmarshal(data, &miner)
	mexist := MinerManagerImpl.GetMinerById(transaction.Source[:], miner.Type, accountdb)
	if mexist != nil {
		logger.Debugf("VMExecutor Execute MinerApply Fail(Already Exist) Source %s Type:%s", transaction.Source.GetHexString(), mark)
		return success
	}

	amount := big.NewInt(int64(miner.Stake))
	txExecuteFee := big.NewInt(int64(1 * TransactionGasCost))
	if canTransfer(accountdb, *transaction.Source, amount, txExecuteFee) {
		accountdb.SubBalance(*transaction.Source, txExecuteFee)
		accountdb.AddBalance(castor, txExecuteFee)

		miner.ApplyHeight = height
		if MinerManagerImpl.addMiner(transaction.Source[:], &miner, accountdb) > 0 {
			accountdb.SubBalance(*transaction.Source, amount)
			logger.Debugf("VMExecutor Execute MinerApply Success Source:%s Height:%d Type:%s", transaction.Source.GetHexString(), height, mark)
		}
		success = true
	} else {
		logger.Debugf("VMExecutor Execute MinerApply Fail(Balance Not Enough) Source:%s Height:%d Type:%s", transaction.Source.GetHexString(), height, mark)
	}
	return success
}

func (executor *VMExecutor) executeMinerAbortTx(accountdb *account.AccountDB, transaction *types.Transaction, height uint64, castor common.Address, mark string) (success bool) {
	success = false
	txExecuteFee := big.NewInt(int64(1 * TransactionGasCost))
	if canTransfer(accountdb, *transaction.Source, new(big.Int).SetUint64(0), txExecuteFee) {
		accountdb.SubBalance(*transaction.Source, txExecuteFee)
		accountdb.AddBalance(castor, txExecuteFee)
		if len(transaction.Data) != 0 {
			success = MinerManagerImpl.abortMiner(transaction.Source[:], transaction.Data[0], height, accountdb)
		}
	} else {
		logger.Debugf("VMExecutor Execute MinerAbort Fail(Balance Not Enough) Source:%s Height:%d ,Type:%s", transaction.Source.GetHexString(), height, mark)
	}
	logger.Debugf("VMExecutor Execute MinerAbort Tx %s,Source:%s, Success:%t,Type:%s", transaction.Hash.String(), transaction.Source.GetHexString(), success, mark)
	return success
}

func (executor *VMExecutor) executeMinerRefundTx(accountdb *account.AccountDB, transaction *types.Transaction, height uint64, castor common.Address, mark string) (success bool) {
	success = false
	txExecuteFee := big.NewInt(int64(1 * TransactionGasCost))
	if canTransfer(accountdb, *transaction.Source, new(big.Int).SetUint64(0), txExecuteFee) {
		accountdb.SubBalance(*transaction.Source, txExecuteFee)
		accountdb.AddBalance(castor, txExecuteFee)
	} else {
		logger.Debugf("VMExecutor Execute MinerRefund Fail(Balance Not Enough) Hash:%s,Source:%s,Type:%s", transaction.Hash.String(), transaction.Source.GetHexString(), mark)
		return success
	}

	mexist := MinerManagerImpl.GetMinerById(transaction.Source[:], transaction.Data[0], accountdb)
	if mexist != nil && mexist.Status == types.MinerStatusAbort {
		if mexist.Type == types.MinerTypeHeavy {
			if height > mexist.AbortHeight+10 {
				MinerManagerImpl.removeMiner(transaction.Source[:], mexist.Type, accountdb)
				amount := big.NewInt(int64(mexist.Stake))
				accountdb.AddBalance(*transaction.Source, amount)
				logger.Debugf("VMExecutor Execute MinerRefund Heavy Success %s,Type:%s", transaction.Source.GetHexString(), mark)
				success = true
			} else {
				logger.Debugf("VMExecutor Execute MinerRefund Heavy Fail(Refund height less than abortHeight+10) Hash%s,Type:%s", transaction.Source.GetHexString(), mark)
			}
		} else {
			if !isActive(transaction.Source[:], height) {
				MinerManagerImpl.removeMiner(transaction.Source[:], mexist.Type, accountdb)
				amount := big.NewInt(int64(mexist.Stake))
				accountdb.AddBalance(*transaction.Source, amount)
				logger.Debugf("VMExecutor Execute MinerRefund Light Success %s,Type:%s", transaction.Source.GetHexString())
				success = true
			} else {
				logger.Debugf("VMExecutor Execute MinerRefund Light Fail(Still In Active Group) %s,Type:%s", transaction.Source.GetHexString(), mark)
			}
		}
	} else {
		logger.Debugf("VMExecutor Execute MinerRefund Fail(Not Exist Or Not Abort) %s,Type:%s", transaction.Source.GetHexString(), mark)
	}
	return success
}

func createContract(accountdb *account.AccountDB, transaction *types.Transaction) (common.Address, *types.TransactionError) {
	contractAddr := common.BytesToAddress(common.Sha256(common.BytesCombine(transaction.Source[:], common.Uint64ToByte(transaction.Nonce))))

	if accountdb.GetCodeHash(contractAddr) != (common.Hash{}) {
		return common.Address{}, types.NewTransactionError(types.TxErrorCode_ContractAddressConflict, "contract address conflict")
	}
	accountdb.CreateAccount(contractAddr)
	accountdb.SetCode(contractAddr, []byte(transaction.Data))
	accountdb.SetNonce(contractAddr, 1)
	return contractAddr, nil
}

func canTransfer(db *account.AccountDB, addr common.Address, amount *big.Int, gasFee *big.Int) bool {
	totalAmount := new(big.Int).Add(amount, gasFee)
	return db.GetBalance(addr).Cmp(totalAmount) >= 0
}

func transfer(db *account.AccountDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

func isActive(minerId []byte, currentBlockHeight uint64) bool {
	iterator := groupChainImpl.Iterator()
	for g := iterator.Current(); g != nil; g = iterator.MovePre() {
		if g.Header.DismissHeight <= currentBlockHeight {
			genesisGroup := groupChainImpl.GetGroupByHeight(0)
			for _, member := range genesisGroup.Members {
				if bytes.Equal(member, minerId) {
					return true
				}
			}
			break
		} else {
			for _, member := range g.Members {
				if bytes.Equal(member, minerId) {
					return true
				}
			}
		}
	}
	return false
}

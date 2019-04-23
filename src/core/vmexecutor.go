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
const MaxCastBlockTime = time.Second * 3

type VMExecutor struct {
	bc BlockChain
}

type WithdrawInfo struct {
	Address string

	GameId string

	Amount string
}

type DepositInfo struct {
	Address string

	GameId string

	Amount string
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

						address := common.HexToAddress(user.Address)
						accountdb.SetNonce(address, accountdb.GetNonce(address)+1)
					}

				}

			}
		case types.TransactionTypeWithdraw:

		case types.TransactionTypeAssetOnChain:
		case types.TransactionTypeDepositExecute:
			success = executor.executeDepositNotify(accountdb, transaction)
		case types.TransactionTypeWithdrawExecute:
			success = executor.executeWithdrawNotify(accountdb, transaction)
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

func (executor *VMExecutor) executeWithdraw(accountdb *account.AccountDB, transaction *types.Transaction) bool {
	return true
}


func (executor *VMExecutor) executeAssetOnChain(accountdb *account.AccountDB, transaction *types.Transaction) bool {
	return true
}

func (executor *VMExecutor) executeDepositNotify(accountdb *account.AccountDB, transaction *types.Transaction) bool {
	depositInfo := DepositInfo{}
	err := json.Unmarshal([]byte(transaction.Data), depositInfo)
	if err != nil {
		logger.Errorf("Execute deposit json unmarshal err:%s", err.Error())
		return false
	}

	depositAmount, _ := new(big.Int).SetString(depositInfo.Amount, 10)
	account := accountdb.GetSubAccount(common.HexToAddress(depositInfo.Address), depositInfo.GameId)
	logger.Debugf("Execute deposit:%s,current balance:%d,deposit balance:%d", transaction.Hash.String(), account.Balance.Uint64(), depositAmount.Uint64())

	account.Balance.Add(account.Balance, depositAmount)
	logger.Debugf("After execute deposit:%s, balance:%d", transaction.Hash.String(), account.Balance.Uint64())
	accountdb.UpdateSubAccount(common.HexToAddress(depositInfo.Address), depositInfo.GameId, *account)
	return true
}

func (executor *VMExecutor) executeWithdrawNotify(accountdb *account.AccountDB, transaction *types.Transaction) bool {
	withdrawInfo := WithdrawInfo{}
	err := json.Unmarshal([]byte(transaction.Data), withdrawInfo)
	if err != nil {
		logger.Errorf("Execute withdraw json unmarshal err:%s", err.Error())
		return false
	}

	withdrawAmount, _ := new(big.Int).SetString(withdrawInfo.Amount, 10)
	account := accountdb.GetSubAccount(common.HexToAddress(withdrawInfo.Address), withdrawInfo.GameId)
	logger.Errorf("Execute withdraw:%s,current balance:%d,withdraw balance:%d", transaction.Hash.String(), account.Balance.Uint64(), withdrawAmount.Uint64())
	if account.Balance.Cmp(withdrawAmount) < 0 {
		logger.Errorf("Execute withdraw balance not enough:current balance:%d,withdraw balance:%d", account.Balance.Uint64(), withdrawAmount.Uint64())
		return false
	}
	account.Balance.Sub(account.Balance, withdrawAmount)
	logger.Debugf("After execute withdraw:%s, balance:%d", transaction.Hash.String(), account.Balance.Uint64())
	accountdb.UpdateSubAccount(common.HexToAddress(withdrawInfo.Address), withdrawInfo.GameId, *account)
	return true
}

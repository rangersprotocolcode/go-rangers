package core

import (
	"time"
	"math/big"

	"x/src/common"
	"x/src/middleware/types"
	"x/src/storage/account"

	"x/src/statemachine"
	"strconv"
	"encoding/json"
)

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

	for _, transaction := range block.Transactions {
		executeTime := time.Now()
		if situation == "casting" && executeTime.Sub(beginTime) > MaxCastBlockTime {
			logger.Infof("Cast block execute tx time out!Tx hash:%s ", transaction.Hash.String())
			break
		}
		//logger.Debugf("VMExecutor Execute %v,type:%d", transaction.Hash, transaction.Type)
		var success = false

		if !executor.validateNonce(accountdb, transaction) {
			evictedTxs = append(evictedTxs, transaction.Hash)
			continue
		}

		switch transaction.Type {
		case types.TransactionTypeOperatorEvent:

			// 已经执行过了，则直接返回true
			if nil == GetBlockChain().GetTransactionPool().GetExecuted(transaction.Hash) {
				payload := transaction.Data
				statemachine.Docker.Process(transaction.Target, "operator", strconv.FormatUint(transaction.Nonce, 10), payload)

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

	}
	accountdb.AddBalance(common.BytesToAddress(block.Header.Castor), consensusHelper.ProposalBonus())

	state := accountdb.IntermediateRoot(true)
	logger.Debugf("VMExecutor End Execute State %s", state.Hex())
	return state, evictedTxs, transactions, receipts, nil, errs
}

func (executor *VMExecutor) validateNonce(accountdb *account.AccountDB, transaction *types.Transaction) bool {
	return true
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

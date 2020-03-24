package core

import (
	"time"
	"x/src/common"
	"x/src/middleware/types"
	"x/src/storage/account"

	"x/src/middleware"
)

var executors map[int32]executor

type executor interface {
	Execute(tx *types.Transaction, header *types.BlockHeader, accountdb *account.AccountDB, context map[string]interface{}) (bool, string)
}

func initExecutors() {
	executors = make(map[int32]executor, 20)

	executors[types.TransactionTypeOperatorEvent] = &operatorExecutor{}
	executors[types.TransactionTypeWithdraw] = &withdrawExecutor{}
	executors[types.TransactionTypeCoinDepositAck] = &coinDepositExecutor{}
	executors[types.TransactionTypeFTDepositAck] = &ftDepositExecutor{}
	executors[types.TransactionTypeNFTDepositAck] = &nftDepositExecutor{}
	executors[types.TransactionTypeMinerApply] = &minerApplyExecutor{}
	executors[types.TransactionTypeMinerAdd] = &minerAddExecutor{}
	executors[types.TransactionTypeMinerRefund] = &minerRefundExecutor{}

	executors[types.TransactionTypePublishFT] = &ftExecutor{}
	executors[types.TransactionTypePublishNFTSet] = &ftExecutor{}
	executors[types.TransactionTypeMintFT] = &ftExecutor{}
	executors[types.TransactionTypeMintNFT] = &ftExecutor{}
	executors[types.TransactionTypeShuttleNFT] = &ftExecutor{}

	executors[types.TransactionTypeAddStateMachine] = &stmExecutor{}
	executors[types.TransactionTypeUpdateStorage] = &stmExecutor{}
	executors[types.TransactionTypeStartSTM] = &stmExecutor{}
	executors[types.TransactionTypeStopSTM] = &stmExecutor{}
	executors[types.TransactionTypeUpgradeSTM] = &stmExecutor{}
	executors[types.TransactionTypeQuitSTM] = &stmExecutor{}
	executors[types.TransactionTypeImportNFT] = &stmExecutor{}
}

const MaxCastBlockTime = time.Second * 3

type VMExecutor struct {
	accountdb *account.AccountDB
	block     *types.Block
	situation string
	context   map[string]interface{}
}

func newVMExecutor(accountdb *account.AccountDB, block *types.Block, situation string) *VMExecutor {
	vm := &VMExecutor{
		accountdb: accountdb,
		block:     block,
		situation: situation,
		context:   make(map[string]interface{}),
	}

	return vm
}

func (this *VMExecutor) Execute() (common.Hash, []common.Hash, []*types.Transaction, []*types.Receipt, error, []*types.TransactionError) {
	beginTime := time.Now()

	receipts := make([]*types.Receipt, 0)
	transactions := make([]*types.Transaction, 0)
	evictedTxs := make([]common.Hash, 0)
	errs := make([]*types.TransactionError, len(this.block.Transactions))

	this.prepare()

	for _, transaction := range this.block.Transactions {
		executeTime := time.Now()
		if this.situation == "casting" && executeTime.Sub(beginTime) > MaxCastBlockTime {
			logger.Infof("Cast block execute tx time out!Tx hash:%s ", transaction.Hash.String())
			break
		}
		logger.Debugf("Execute %s, type:%d", transaction.Hash.String(), transaction.Type)

		executor := executors[transaction.Type]
		snapshot := this.accountdb.Snapshot()
		success := false
		msg := ""
		if executor != nil {
			success, msg = executor.Execute(transaction, this.block.Header, this.accountdb, this.context)
		}

		if !success {
			logger.Debugf("Execute failed tx:%s, type:%d", transaction.Hash.String(), transaction.Type)
			evictedTxs = append(evictedTxs, transaction.Hash)
			this.accountdb.RevertToSnapshot(snapshot)
		} else {
			if transaction.Source != "" {
				this.accountdb.SetNonce(common.HexToAddress(transaction.Source), transaction.Nonce)
			}

			logger.Debugf("Execute success %s,type:%d", transaction.Hash.String(), transaction.Type)
		}

		transactions = append(transactions, transaction)
		receipt := types.NewReceipt(nil, !success, 0, this.block.Header.Height, msg, transaction.Source)
		receipt.TxHash = transaction.Hash
		receipts = append(receipts, receipt)
	}

	this.after()

	state := this.accountdb.IntermediateRoot(true)

	middleware.PerfLogger.Debugf("VMExecutor End. %s height: %d, cost: %v, txs: %d", this.situation, this.block.Header.Height, time.Since(beginTime), len(this.block.Transactions))
	return state, evictedTxs, transactions, receipts, nil, errs
}

func (executor *VMExecutor) validateNonce(accountdb *account.AccountDB, transaction *types.Transaction) bool {
	return true
}

func (executor *VMExecutor) prepare() {
	executor.context["refund"] = make(map[uint64]RefundInfoList)
}

func (executor *VMExecutor) after() {
	height := executor.block.Header.Height

	// 计算定时任务（冻结、退款等等）
	RefundManagerImpl.Add(getRefundInfo(executor.context), executor.accountdb)
	RefundManagerImpl.CheckAndMove(height, executor.accountdb)

	// 计算出块奖励
	RewardCalculatorImpl.CalculateReward(height, executor.accountdb)
}
func getRefundInfo(context map[string]interface{}) map[uint64]RefundInfoList {
	raw := context["refund"]
	return raw.(map[uint64]RefundInfoList)
}

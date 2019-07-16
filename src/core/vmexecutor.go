package core

import (
	"time"
	"math/big"

	"x/src/common"
	"x/src/middleware/types"
	"x/src/storage/account"

	"encoding/json"
	"x/src/utility"
	"x/src/network"
	"x/src/statemachine"
	"strconv"
)

const MaxCastBlockTime = time.Second * 3

type VMExecutor struct {
	bc BlockChain
}

type WithdrawInfo struct {
	Address string

	GameId string

	Amount string

	Hash string
}

type DepositInfo struct {
	Address string

	GameId string

	Amount string
}

type AssetOnChainInfo struct {
	Address string

	GameId string

	Assets []types.Asset

	Hash string
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
		logger.Debugf("VMExecutor Execute %v,type:%d", transaction.Hash, transaction.Type)
		var success = false
		snapshot := accountdb.Snapshot()

		switch transaction.Type {
		case types.TransactionTypeOperatorEvent:

			// 在交易池里，表示game_executor已经执行过状态机了
			// 只要处理交易里的subTransaction即可
			if nil != TxManagerInstance.BeginTransaction(transaction.Target, accountdb, false, transaction) {
				logger.Debugf("Begin transaction is not nil!")
				// 处理转账
				// 支持多人转账{"address1":"value1", "address2":"value2"}
				// 理论上这里不应该失败，nonce保证了这一点
				if 0 != len(transaction.ExtraData) {
					mm := make(map[string]string, 0)
					if err := json.Unmarshal([]byte(transaction.ExtraData), &mm); nil != err {
						TxManagerInstance.Commit(transaction.Target, transaction.Hash)
						success = false
						break
					}
					if !changeBalances(transaction.Target, transaction.Source, mm, accountdb) {
						TxManagerInstance.Commit(transaction.Target, transaction.Hash)
						success = false
						break
					}

				}

				// 本地没有执行过状态机(game_executor还没有收到消息)，则需要调用状态机
				if !GetBlockChain().GetTransactionPool().IsGameData(transaction.Hash) {
					logger.Debugf("Is game data")
					statemachine.Docker.Process(transaction.Target, "operator", strconv.FormatUint(transaction.Nonce, 10), transaction.Data)
					GetBlockChain().GetTransactionPool().PutGameData(transaction.Hash)
				} else if 0 != len(transaction.SubTransactions) {
					logger.Debugf("Is not game data")
					for _, sub := range transaction.SubTransactions {
						logger.Debugf("Execute sub tx:%v", sub)
						data := make([]types.UserData, 0)
						if err := json.Unmarshal([]byte(sub), &data); err != nil {
							logger.Error("Execute TransactionUpdateOperatorEvent tx:%s json unmarshal, err:%s", sub, err.Error())
							success = false
						} else {
							if nil != data && 0 != len(data) {
								for _, user := range data {
									if !UpdateAsset(user, transaction.Target, accountdb) {
										success = false
										break
									}

									address := common.HexToAddress(user.Address)
									accountdb.SetNonce(address, 1)
								}

							}

						}
					}
				}

				logger.Debugf("Tx manager commit!")
				TxManagerInstance.Commit(transaction.Target, transaction.Hash)
				logger.Debugf("After Tx manager commit!")
			}

			success = true

		case types.TransactionTypeWithdraw:
			success = executor.executeWithdraw(accountdb, transaction)
		case types.TransactionTypeAssetOnChain:
			success = executor.executeAssetOnChain(accountdb, transaction)
		case types.TransactionTypeDepositAck:
			success = executor.executeDepositNotify(accountdb, transaction)
		}

		if !success {
			logger.Debugf("Execute failed tx:%s", transaction.Hash.String())
			evictedTxs = append(evictedTxs, transaction.Hash)
			accountdb.RevertToSnapshot(snapshot)
		}

		if success || transaction.Type != types.TransactionTypeBonus {
			transactions = append(transactions, transaction)
			receipt := types.NewReceipt(nil, !success, 0)
			receipt.TxHash = transaction.Hash
			receipts = append(receipts, receipt)
			if transaction.Source != "" {
				accountdb.SetNonce(common.HexToAddress(transaction.Source), transaction.Nonce)
			}
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
	logger.Debugf("Execute withdraw tx:%v", transaction)
	amount, err := utility.StrToBigInt(transaction.Data)
	if err != nil {
		logger.Error("Execute withdraw bad amount!Hash:%s, err:%s", transaction.Hash.String(), err.Error())
		return false
	}
	account := accountdb.GetSubAccount(common.HexToAddress(transaction.Source), transaction.Target)
	logger.Errorf("Execute withdraw :%s,current balance:%d,withdraw balance:%d", transaction.Hash.String(), account.Balance.Uint64(), amount.Uint64())
	if account.Balance.Cmp(amount) < 0 {
		logger.Errorf("Execute withdraw balance not enough:current balance:%d,withdraw balance:%d", account.Balance.Uint64(), amount.Uint64())
		return false
	}

	account.Balance.Sub(account.Balance, amount)
	logger.Debugf("After execute withdraw:%s, balance:%d", transaction.Hash.String(), account.Balance.Uint64())
	accountdb.UpdateSubAccount(common.HexToAddress(transaction.Source), transaction.Target, *account)

	withdrawInfo := WithdrawInfo{Address: transaction.Source, GameId: transaction.Target, Amount: amount.String(), Hash: transaction.Hash.String()}
	b, err := json.Marshal(withdrawInfo)
	if err != nil {
		logger.Error("Execute withdraw tx:%s json marshal err, err:%s", transaction.Hash.String(), err.Error())
		return false
	}

	logger.Debugf("After execute withdraw.Send msg to coin proxy:%s", string(b))
	message := network.Message{Code: network.WithDraw, Body: b}
	network.GetNetInstance().SendToCoinProxy(message)
	return true
}

func (executor *VMExecutor) executeAssetOnChain(accountdb *account.AccountDB, transaction *types.Transaction) bool {
	logger.Debugf("Execute asset on chain tx:%v", transaction)
	assetOnChainInfo := AssetOnChainInfo{Address: transaction.Source, GameId: transaction.Target, Hash: transaction.Hash.String()}

	var assetIdList []string
	if err := json.Unmarshal([]byte(transaction.Data), &assetIdList); err != nil {
		logger.Errorf("Execute asset on chain tx:%s,json unmarshal error:%s", transaction.Hash.String(), err.Error())
		return false
	}

	account := accountdb.GetSubAccount(common.HexToAddress(transaction.Source), transaction.Target)

	var assets []types.Asset
	for _, assetId := range assetIdList {
		value := account.Assets[assetId]
		if 0 == len(value) {
			logger.Errorf("AssetOnChain tx:%s,unknown asset id:%s", transaction.Hash.String(), assetId)
			continue
		}

		assets = append(assets, types.Asset{Id: assetId, Value: value})

	}
	assetOnChainInfo.Assets = assets

	b, err := json.Marshal(assetOnChainInfo)
	if err != nil {
		logger.Error("Execute asset on chain tx:%s json marshal err, err:%s", transaction.Hash.String(), err.Error())
		return false
	}

	logger.Debugf("After execute asset on chain.Send msg to coin proxy:%s", string(b))
	message := network.Message{Code: network.AssetOnChain, Body: b}
	network.GetNetInstance().SendToCoinProxy(message)
	return true
}

func (executor *VMExecutor) executeDepositNotify(accountdb *account.AccountDB, transaction *types.Transaction) bool {
	depositAmount, _ := new(big.Int).SetString(transaction.Data, 10)
	account := accountdb.GetSubAccount(common.HexToAddress(transaction.Source), transaction.Target)
	logger.Debugf("Execute deposit notify:%s,address:%s,gameId:%s,current balance:%d,deposit balance:%d", transaction.Hash.String(), transaction.Source, transaction.Target, account.Balance.Uint64(), depositAmount.Uint64())

	account.Balance.Add(account.Balance, depositAmount)
	logger.Debugf("After execute deposit:%s, balance:%d", transaction.Hash.String(), account.Balance.Uint64())
	accountdb.UpdateSubAccount(common.HexToAddress(transaction.Source), transaction.Target, *account)

	b := accountdb.GetSubAccount(common.HexToAddress(transaction.Source), transaction.Target)
	logger.Debugf("After  deposit update,current balance:%d", b.Balance.Uint64())
	return true
}

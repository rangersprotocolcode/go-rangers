package core

import (
	"time"
	"x/src/common"
	"x/src/middleware/types"
	"x/src/storage/account"

	"encoding/json"
	"x/src/utility"
	"x/src/network"
	"x/src/statemachine"
	"strconv"
	"math/big"
)

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
			logger.Debugf("Begin transaction is not nil!")
			// 处理转账
			// 支持多人转账{"address1":"value1", "address2":"value2"}
			// 理论上这里不应该失败，nonce保证了这一点
			if 0 != len(transaction.ExtraData) {
				mm := make(map[string]types.TransferData, 0)
				if err := json.Unmarshal([]byte(transaction.ExtraData), &mm); nil != err {
					success = false
					break
				}
				if !changeBalances(transaction.Target, transaction.Source, mm, accountdb) {
					success = false
					break
				}

			}

			// 在交易池里，表示game_executor已经执行过状态机了
			// 只要处理交易里的subTransaction即可
			if nil != TxManagerInstance.BeginTransaction(transaction.Target, accountdb, transaction) {
				success = true
				if 0 != len(transaction.SubTransactions) {
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
			} else {
				// 本地没有执行过状态机(game_executor还没有收到消息)，则需要调用状态机
				transaction.SubTransactions = make([]string, 0)
				outputMessage := statemachine.Docker.Process(transaction.Target, "operator", strconv.FormatUint(transaction.Nonce, 10), transaction.Data)
				GetBlockChain().GetTransactionPool().PutGameData(transaction.Hash)
				result := ""
				if outputMessage != nil {
					result = outputMessage.Payload
				}

				if 0 == len(result) || result == "fail to transfer" || outputMessage == nil || outputMessage.Status == 1 {
					TxManagerInstance.RollBack(transaction.Target)
				} else {
					TxManagerInstance.Commit(transaction.Target)
					success = true
				}

			}

		case types.TransactionTypeWithdraw:
			success = executor.executeWithdraw(accountdb, transaction)
		case types.TransactionTypeDepositAck:
			success = executor.executeDepositNotify(accountdb, transaction)
		}

		if !success {
			logger.Debugf("Execute failed tx:%s", transaction.Hash.String())
			evictedTxs = append(evictedTxs, transaction.Hash)
			accountdb.RevertToSnapshot(snapshot)
			subAccount := accountdb.GetSubAccount(common.HexToAddress(transaction.Source), "0xad77feef282221a9e9cbd24cc56ef0195353a828")
			txLogger.Debugf("After roll back 0xad77feef282221a9e9cbd24cc56ef0195353a828:%v", subAccount)

			subAccount1 := accountdb.GetSubAccount(common.HexToAddress(transaction.Source), "0xad77feef282221a9e9cbd24cc56ef0195353a829")
			txLogger.Debugf("After roll back 0xad77feef282221a9e9cbd24cc56ef0195353a829:%v", subAccount1)
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
	txLogger.Debugf("Execute withdraw tx:%v", transaction)
	if transaction.Data == "" {
		return false
	}
	var withDrawReq types.WithDrawReq
	err := json.Unmarshal([]byte(transaction.Data), &withDrawReq)
	if err != nil {
		txLogger.Debugf("Unmarshal data error:%s", err.Error())
		return false
	}
	if withDrawReq.ChainType == "" || withDrawReq.Address == "" {
		return false
	}

	subAccount := accountdb.GetSubAccount(common.HexToAddress(transaction.Source), transaction.Target)
	txLogger.Debugf("Before execute withdraw tx,subAccount:%v", subAccount)

	//余额检查
	var withdrawAmount = big.NewInt(0)
	if withDrawReq.Balance != "" {
		withdrawAmount, err = utility.StrToBigInt(withDrawReq.Balance)
		if err != nil {
			txLogger.Error("Execute withdraw bad amount!Hash:%s, err:%s", transaction.Hash.String(), err.Error())
			return false
		}
		txLogger.Errorf("Execute withdraw :%s,current balance:%d,withdraw balance:%d", transaction.Hash.String(), subAccount.Balance.Uint64(), withdrawAmount.Uint64())
		if subAccount.Balance.Cmp(withdrawAmount) < 0 {
			txLogger.Errorf("Execute withdraw balance not enough:current balance:%d,withdraw balance:%d", subAccount.Balance.Uint64(), withdrawAmount.Uint64())
			return false
		}
	}

	//ft检查
	ftInfo := make(map[string]string, 0)
	if withDrawReq.FT != nil && len(withDrawReq.FT) != 0 {
		if subAccount.Ft == nil || len(subAccount.Ft) == 0 {
			return false
		}
		for k, v := range withDrawReq.FT {
			subValue := subAccount.Ft[k]
			if subValue == "" {
				return false
			}

			compareResult, sub := canWithDraw(v, subValue)
			if !compareResult {
				return false
			}
			ftInfo[k] = sub
		}
	}

	//nft检查
	nftInfo := make(map[string]string, 0)
	if withDrawReq.NFT != nil && len(withDrawReq.NFT) != 0 {
		if subAccount.Assets == nil || len(subAccount.Assets) == 0 {
			return false
		}
		for _, k := range withDrawReq.NFT {
			subValue := subAccount.Assets[k]
			if subValue == "" {
				return false
			}
			nftInfo[k] = subValue
		}
	}

	//执行提现

	//扣余额
	subAccount.Balance.Sub(subAccount.Balance, withdrawAmount)

	//FT扣钱
	if len(ftInfo) != 0 {
		for k, v := range ftInfo {
			subAccount.Ft[k] = v
		}
	}

	//删除要提现的NFT
	if len(nftInfo) != 0 {
		for k, _ := range nftInfo {
			delete(subAccount.Assets, k)
		}
	}
	txLogger.Debugf("After execute withdraw:, subAccount:%v", subAccount)
	accountdb.UpdateSubAccount(common.HexToAddress(transaction.Source), transaction.Target, *subAccount)

	//发送给Coin Connector
	withdrawData := types.WithDrawData{ChainType: withDrawReq.ChainType, Balance: withDrawReq.Balance, Address: withDrawReq.Address}
	withdrawData.FT = withDrawReq.FT
	withdrawData.NFT = nftInfo

	b, err := json.Marshal(withdrawData)
	if err != nil {
		txLogger.Error("Execute withdraw tx:%s json marshal err, err:%s", transaction.Hash.String(), err.Error())
		return false
	}
	t := types.Transaction{Source: transaction.Source, Target: transaction.Target, Data: string(b), Type: transaction.Type}
	t.Hash = t.GenHash()

	msg, err := json.Marshal(t.ToTxJson())
	if err != nil {
		txLogger.Debugf("Json marshal tx json error:%s", err.Error())
	}
	txLogger.Debugf("After execute withdraw.Send msg to coin proxy:%s", msg)
	network.GetNetInstance().SendToCoinConnector(msg)
	return true
}

/**
 充值确认
 */
func (executor *VMExecutor) executeDepositNotify(accountdb *account.AccountDB, transaction *types.Transaction) bool {
	txLogger.Debugf("Execute deposit ack tx:%v", transaction)
	if transaction.Data == "" {
		return false
	}
	var depositData types.DepositData
	err := json.Unmarshal([]byte(transaction.Data), &depositData)
	if err != nil {
		txLogger.Debugf("Unmarshal data error:%s", err.Error())
		return false
	}
	if depositData.Amount == "" {
		return false
	}

	subAccount := accountdb.GetSubAccount(common.HexToAddress(transaction.Source), transaction.Target)
	txLogger.Debugf("address:%s,gameId:%s,current balance:%d,deposit balance:%d", transaction.Source, transaction.Target, subAccount.Balance.Uint64(), depositData.Amount)
	txLogger.Debugf("Before execute deposit tx,subAccount:%v", subAccount)

	depositAmount, err := utility.StrToBigInt(depositData.Amount)
	if err != nil {
		txLogger.Debugf("deposit amount to big int error:%s", err.Error())
		return false
	}
	subAccount.Balance.Add(subAccount.Balance, depositAmount)
	txLogger.Debugf("After execute deposit:%s, balance:%d", transaction.Hash.String(), subAccount.Balance.Uint64())
	//todo for test
	if subAccount.Ft == nil {
		subAccount.Ft = make(map[string]string, 0)
	}
	if depositData.FT != nil {
		for key, value := range depositData.FT {
			b1, _ := utility.StrToBigInt(value)
			subAccount.Ft[key] = b1.String()
		}
	}

	if subAccount.Assets == nil {
		subAccount.Assets = make(map[string]string, 0)
	}
	if depositData.NFT != nil {
		for key, value := range depositData.NFT {
			subAccount.Assets[key] = value
		}
	}
	//end for test

	accountdb.UpdateSubAccount(common.HexToAddress(transaction.Source), transaction.Target, *subAccount)
	b := accountdb.GetSubAccount(common.HexToAddress(transaction.Source), transaction.Target)
	txLogger.Debugf("After  deposit update,current sub account:%v", b)
	return true
}

/**
字符串的余额比较
withDrawAmount金额大于等于ftValue 返回TRUE 否则返回FALSE
withDrawAmount 是浮点数的STRING "11.256"
ftValue 是bigInt的STRING "56631"
如果格式有错误返回FALSE
返回为true的话 返回二者的差值string
*/
func canWithDraw(withDrawAmount string, ftValue string) (bool, string) {
	b1, err1 := utility.StrToBigInt(withDrawAmount)
	if err1 != nil {
		return false, ""
	}

	b2, r2 := new(big.Int).SetString(ftValue, 10)
	if !r2 {
		return false, ""
	}

	if b1.Cmp(b2) > 0 {
		return false, ""
	}
	var sub big.Int
	sub.Sub(b2, b1)
	return true, sub.String()
}

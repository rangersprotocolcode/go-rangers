package core

import (
	"time"
	"x/src/common"
	"x/src/middleware/types"
	"x/src/storage/account"

	"encoding/json"
	"x/src/utility"
	"x/src/statemachine"
	"strconv"
	"math/big"
	"fmt"
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
		logger.Debugf("VMExecutor Execute %s,type:%d", transaction.Hash.String(), transaction.Type)
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
				_, ok := changeAssets(transaction.Source, mm, accountdb)
				if !ok {
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
					for _, user := range transaction.SubTransactions {
						logger.Debugf("Execute sub tx:%v", user)

						// 发币
						if user.Address == "StartFT" {
							createTime, _ := user.Assets["createTime"]
							_, flag := FTManagerInstance.PublishFTSet(user.Assets["name"], user.Assets["symbol"], user.Assets["gameId"], user.Assets["totalSupply"], user.Assets["owner"], createTime, 1, accountdb)
							if !flag {
								success = false
								break
							}

							continue
						}

						if user.Address == "MintFT" {
							owner := user.Assets["appId"]
							ftId := user.Assets["ftId"]
							target := user.Assets["target"]
							supply := user.Assets["balance"]
							_, flag := FTManagerInstance.MintFT(owner, ftId, target, supply, accountdb)

							if !flag {
								success = false
								break
							}

							continue
						}

						// 给用户币
						if user.Address == "TransferFT" {
							_, _, flag := FTManagerInstance.TransferFT(user.Assets["gameId"], user.Assets["symbol"], user.Assets["target"], user.Assets["supply"], accountdb)
							if !flag {
								success = false
								break
							}
							continue
						}

						// 修改NFT属性
						if user.Address == "UpdateNFT" {
							addr := common.HexToAddress(user.Assets["addr"])
							flag := NFTManagerInstance.UpdateNFT(addr, user.Assets["appId"], user.Assets["setId"], user.Assets["id"], user.Assets["data"], accountdb)
							if !flag {
								success = false
								break
							}
							continue
						}

						// NFT
						if user.Address == "TransferNFT" {
							appId := user.Assets["appId"]
							_, ok := NFTManagerInstance.Transfer(user.Assets["setId"], user.Assets["id"], common.HexToAddress(appId), common.HexToAddress(user.Assets["target"]), accountdb)
							if !ok {
								success = false
								break
							}
							continue
						}

						// 将状态机持有的NFT的使用权授予某地址
						if user.Address == "ApproveNFT" {
							appId := user.Assets["appId"]
							ok := accountdb.ApproveNFT(common.HexToAddress(appId), appId, user.Assets["setId"], user.Assets["id"], user.Assets["target"])
							if !ok {
								success = false
								break
							}
							continue
						}

						if user.Address == "changeNFTStatus" {
							appId := user.Assets["appId"]
							status, _ := strconv.Atoi(user.Assets["status"])
							ok := accountdb.ChangeNFTStatus(common.HexToAddress(appId), appId, user.Assets["setId"], user.Assets["id"], byte(status))
							if !ok {
								success = false
								break
							}
							continue
						}

						if user.Address == "PublishNFTSet" {
							maxSupply := user.Assets["maxSupply"]
							_, err := strconv.ParseInt(maxSupply, 10, 0)
							if err != nil {
								logger.Errorf("Publish nft set!MaxSupply bad format:%s", maxSupply)
								success = false
								break
							}
							appId := user.Assets["appId"]

							_, ok, _ := NFTManagerInstance.PublishNFTSet(user.Assets["setId"], user.Assets["name"], user.Assets["symbol"], appId, appId, maxSupply, user.Assets["createTime"], accountdb)
							if !ok {
								success = false
								break
							}
							continue
						}

						if user.Address == "MintNFT" {
							_, ok := NFTManagerInstance.MintNFT(user.Assets["appId"], user.Assets["setId"], user.Assets["id"], user.Assets["data"], user.Assets["createTime"], common.HexToAddress(user.Assets["target"]), accountdb)
							if !ok {
								success = false
								break
							}
							continue
						}

						// 用户之间转账
						if !UpdateAsset(user, transaction.Target, accountdb) {
							success = false
							break
						}

						//address := common.HexToAddress(user.Address)
						//accountdb.SetNonce(address, 1)

					}
				}
			} else if 0 != len(transaction.Target) {
				// 本地没有执行过状态机(game_executor还没有收到消息)，则需要调用状态机
				transaction.SubTransactions = make([]types.UserData, 0)
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
			break
		case types.TransactionTypePublishFT:
			_, success = PublishFT(accountdb, transaction)
			break
		case types.TransactionTypePublishNFTSet:
			success, _ = PublishNFTSet(accountdb, transaction)
			break
		case types.TransactionTypeMintFT:
			success, _ = MintFT(accountdb, transaction)
			break
		case types.TransactionTypeMintNFT:
			success, _ = MintNFT(accountdb, transaction)
			break
		case types.TransactionTypeWithdraw:
			_, success = Withdraw(accountdb, transaction, true)
			break
		case types.TransactionTypeCoinDepositAck:
			success = executor.executeCoinDepositNotify(accountdb, transaction)
			break
		case types.TransactionTypeFTDepositAck:
			success = executor.executeFTDepositNotify(accountdb, transaction)
			break
		case types.TransactionTypeNFTDepositAck:
			success = executor.executeNFTDepositNotify(accountdb, transaction)
			break
		case types.TransactionTypeShuttleNFT:
			success,_ = ShuttleNFT(accountdb, transaction)
			break
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

//主链币充值确认
func (executor *VMExecutor) executeCoinDepositNotify(accountdb *account.AccountDB, transaction *types.Transaction) bool {
	txLogger.Debugf("Execute coin deposit ack tx:%v", transaction)
	if transaction.Data == "" {
		return false
	}
	var depositCoinData types.DepositCoinData
	err := json.Unmarshal([]byte(transaction.Data), &depositCoinData)
	if err != nil {
		txLogger.Debugf("Deposit coin data unmarshal error:%s", err.Error())
		return false
	}
	txLogger.Debugf("deposit coin data:%v,target address:%s", depositCoinData, transaction.Source)
	if depositCoinData.Amount == "" || depositCoinData.ChainType == "" {
		return false
	}

	value, _ := utility.StrToBigInt(depositCoinData.Amount)
	return accountdb.AddFT(common.HexToAddress(transaction.Source), fmt.Sprintf("official-%s", depositCoinData.ChainType), value)
}

//FT充值确认
func (executor *VMExecutor) executeFTDepositNotify(accountdb *account.AccountDB, transaction *types.Transaction) bool {
	txLogger.Debugf("Execute ft deposit ack tx:%v", transaction)
	if transaction.Data == "" {
		return false
	}
	var depositFTData types.DepositFTData
	err := json.Unmarshal([]byte(transaction.Data), &depositFTData)
	if err != nil {
		txLogger.Debugf("Deposit ft data unmarshal error:%s", err.Error())
		return false
	}
	txLogger.Debugf("deposit ft data:%v, address:%s", depositFTData, transaction.Source)
	if depositFTData.Amount == "" || depositFTData.FTId == "" {
		return false
	}
	//todo 先不检查此ft是否存在
	value, _ := utility.StrToBigInt(depositFTData.Amount)
	return accountdb.AddFT(common.HexToAddress(transaction.Source), depositFTData.FTId, value)
}

//NFT充值确认
func (executor *VMExecutor) executeNFTDepositNotify(accountdb *account.AccountDB, transaction *types.Transaction) bool {
	txLogger.Debugf("Execute nft deposit ack tx:%v", transaction)
	if transaction.Data == "" {
		return false
	}
	var depositNFTData types.DepositNFTData
	err := json.Unmarshal([]byte(transaction.Data), &depositNFTData)
	if err != nil {
		txLogger.Debugf("Deposit nft data unmarshal error:%s", err.Error())
		return false
	}
	txLogger.Debugf("deposit nft data:%v,target address:%s", depositNFTData, transaction.Source)
	if depositNFTData.SetId == "" || depositNFTData.ID == "" || depositNFTData.Value == "" {
		return false
	}

	// 检查setId
	nftSet := NFTManagerInstance.GetNFTSet(depositNFTData.SetId, accountdb)
	if nil == nftSet {
		_, _, nftSet = NFTManagerInstance.PublishNFTSet(depositNFTData.SetId, depositNFTData.Name, depositNFTData.Symbol, depositNFTData.Creator, depositNFTData.Owner, "0", depositNFTData.CreateTime, accountdb)
	}

	appId := transaction.Target
	str, ok := NFTManagerInstance.GenerateNFT(nftSet, appId, depositNFTData.SetId, depositNFTData.ID, depositNFTData.Value, depositNFTData.Creator, depositNFTData.CreateTime, common.HexToAddress(transaction.Source), accountdb)
	txLogger.Debugf("GenerateNFT result:%s,%t", str, ok)
	return ok
}

/**
字符串的余额比较
withDrawAmount金额大于等于ftValue 返回TRUE 否则返回FALSE
withDrawAmount 是浮点数的STRING "11.256"
ftValue 是bigInt的STRING "56631"
如果格式有错误返回FALSE
返回为true的话 返回二者的差值string
*/
func canWithDraw(withDrawAmount string, ftValue *big.Int) (bool, *big.Int) {
	b1, err1 := utility.StrToBigInt(withDrawAmount)
	if err1 != nil {
		return false, nil
	}

	if b1.Cmp(ftValue) > 0 {
		return false, nil
	}

	return true, ftValue.Sub(ftValue, b1)
}

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
	"x/src/middleware"
	"x/src/service"
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
	refundInfos := make(map[uint64]RefundInfoList, 0)

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
				_, ok := service.ChangeAssets(transaction.Source, mm, accountdb)
				if !ok {
					success = false
					break
				}

				success = true
			}

			// 纯转账的场景，不用执行状态机
			if 0 == len(transaction.Target) {
				break
			}

			// 在交易池里，表示game_executor已经执行过状态机了
			// 只要处理交易里的subTransaction即可
			if nil != service.TxManagerInstance.BeginTransaction(transaction.Target, accountdb, transaction) {
				success = true
				logger.Debugf("Is not game data")
				if 0 != len(transaction.SubTransactions) {
					for _, user := range transaction.SubTransactions {
						logger.Debugf("Execute sub tx:%v", user)

						// 发币
						if user.Address == "StartFT" {
							createTime, _ := user.Assets["createTime"]
							ftSet := service.FTManagerInstance.GenerateFTSet(user.Assets["name"], user.Assets["symbol"], user.Assets["gameId"], user.Assets["totalSupply"], user.Assets["owner"], createTime, 1)
							_, flag := service.FTManagerInstance.PublishFTSet(ftSet, accountdb)
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
							_, flag := service.FTManagerInstance.MintFT(owner, ftId, target, supply, accountdb)

							if !flag {
								success = false
								break
							}

							continue
						}

						// 给用户币
						if user.Address == "TransferFT" {
							_, _, flag := service.FTManagerInstance.TransferFT(user.Assets["gameId"], user.Assets["symbol"], user.Assets["target"], user.Assets["supply"], accountdb)
							if !flag {
								success = false
								break
							}
							continue
						}

						// 修改NFT属性
						if user.Address == "UpdateNFT" {
							addr := common.HexToAddress(user.Assets["addr"])
							flag := service.NFTManagerInstance.UpdateNFT(addr, user.Assets["appId"], user.Assets["setId"], user.Assets["id"], user.Assets["data"], accountdb)
							if !flag {
								success = false
								break
							}
							continue
						}

						// NFT
						if user.Address == "TransferNFT" {
							appId := user.Assets["appId"]
							_, ok := service.NFTManagerInstance.Transfer(user.Assets["setId"], user.Assets["id"], common.HexToAddress(appId), common.HexToAddress(user.Assets["target"]), accountdb)
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
							maxSupplyString := user.Assets["maxSupply"]
							maxSupply, err := strconv.Atoi(maxSupplyString)
							if err != nil {
								logger.Errorf("Publish nft set! MaxSupply bad format:%s", maxSupplyString)
								success = false
								break
							}
							appId := user.Assets["appId"]
							nftSet := service.NFTManagerInstance.GenerateNFTSet(user.Assets["setId"], user.Assets["name"], user.Assets["symbol"], appId, appId, maxSupply, user.Assets["createTime"])
							_, ok := service.NFTManagerInstance.PublishNFTSet(nftSet, accountdb)
							if !ok {
								success = false
								break
							}
							continue
						}

						if user.Address == "MintNFT" {
							_, ok := service.NFTManagerInstance.MintNFT(user.Assets["appId"], user.Assets["setId"], user.Assets["id"], user.Assets["data"], user.Assets["createTime"], common.HexToAddress(user.Assets["target"]), accountdb)
							if !ok {
								success = false
								break
							}
							continue
						}

						// 用户之间转账
						if !service.UpdateAsset(user, transaction.Target, accountdb) {
							success = false
							break
						}

						//address := common.HexToAddress(user.Address)
						//accountdb.SetNonce(address, 1)

					}
				}
			} else if 0 != len(transaction.Target) {
				// 本地没有执行过状态机(game_executor还没有收到消息)，则需要调用状态机
				// todo: RequestId 排序问题
				transaction.SubTransactions = make([]types.UserData, 0)
				outputMessage := statemachine.STMManger.Process(transaction.Target, "operator", transaction.RequestId, transaction.Data, transaction)
				service.GetTransactionPool().PutGameData(transaction.Hash)
				result := ""
				if outputMessage != nil {
					result = outputMessage.Payload
				}

				if 0 == len(result) || result == "fail to transfer" || outputMessage == nil || outputMessage.Status == 1 {
					success = false
					service.TxManagerInstance.RollBack(transaction.Target)
				} else {
					service.TxManagerInstance.Commit(transaction.Target)
					success = true
				}

			}
			break
		case types.TransactionTypePublishFT:
			_, success = service.PublishFT(accountdb, transaction)
			break
		case types.TransactionTypePublishNFTSet:
			success, _ = service.PublishNFTSet(accountdb, transaction)
			break
		case types.TransactionTypeMintFT:
			success, _ = service.MintFT(accountdb, transaction)
			break
		case types.TransactionTypeMintNFT:
			success, _ = service.MintNFT(accountdb, transaction)
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
			success, _ = service.ShuttleNFT(accountdb, transaction)
			break
		case types.TransactionTypeImportNFT:
			appId := transaction.Source
			success = statemachine.STMManger.IsAppId(appId)
			break
		case types.TransactionTypeAddStateMachine:
			// todo: 经济模型，新增状态机应该要付费
			go statemachine.STMManger.AddStatemachine(transaction.Source, transaction.Data)
			success = true
			break
		case types.TransactionTypeUpdateStorage:
			// todo: 经济模型，更新状态机应该要付费
			go statemachine.STMManger.UpdateSTMStorage(transaction.Source, transaction.Data)
			success = true
			break
		case types.TransactionTypeStartSTM:
			// todo: 经济模型，重启状态机应该要付费
			go statemachine.STMManger.StartSTM(transaction.Source)
			success = true
			break
		case types.TransactionTypeStopSTM:
			// todo: 经济模型，重启状态机应该要付费
			go statemachine.STMManger.StopSTM(transaction.Source)
			success = true
			break
		case types.TransactionTypeUpgradeSTM:
			// todo: 经济模型，重启状态机应该要付费
			go statemachine.STMManger.UpgradeSTM(transaction.Source, transaction.Data)
			success = true
			break
		case types.TransactionTypeQuitSTM:
			// todo: 经济模型，重启状态机应该要付费
			go statemachine.STMManger.QuitSTM(transaction.Source)
			success = true
			break
		case types.TransactionTypeMinerApply:
			data := transaction.Data
			var miner types.Miner
			err := json.Unmarshal([]byte(data), &miner)
			if err != nil {
				logger.Errorf("json Unmarshal error, %s", err.Error())
				success = false
			} else {
				miner.ApplyHeight = height + common.HeightAfterStake
				success = MinerManagerImpl.AddMiner(common.HexToAddress(transaction.Source), &miner, accountdb)
			}
			break
		case types.TransactionTypeMinerAdd:
			data := transaction.Data
			var miner types.Miner
			err := json.Unmarshal([]byte(data), &miner)
			if err != nil {
				logger.Errorf("json Unmarshal error, %s", err.Error())
				success = false
			} else {
				success = MinerManagerImpl.AddStake(common.HexToAddress(transaction.Source), miner.Id, miner.Stake, accountdb)
			}
		case types.TransactionTypeMinerRefund:
			value, err := strconv.ParseUint(transaction.Data, 10, 64)
			if err != nil {
				logger.Errorf("fail to refund %s", transaction.Data)
				success = false
			} else {
				minerId := common.Hex2Bytes(transaction.Source)
				refundHeight, money, refundErr := RefundManagerImpl.GetRefundStake(height, minerId, value, accountdb)
				if refundErr != nil {
					logger.Errorf("fail to refund %s, err: %s", transaction.Data, refundErr.Error())
					success = false
				} else {
					success = true
					logger.Infof("add refund, minerId: %s, height: %d, money: %d", transaction.Source, refundHeight, money)
					refundInfo, ok := refundInfos[refundHeight]
					if ok {
						refundInfo.AddRefundInfo(minerId, money)
					} else {
						refundInfo = RefundInfoList{}
						refundInfo.AddRefundInfo(minerId, money)
						refundInfos[refundHeight] = refundInfo
					}
				}
			}
			break
		}

		if !success {
			logger.Debugf("Execute failed tx:%s", transaction.Hash.String())
			evictedTxs = append(evictedTxs, transaction.Hash)
			accountdb.RevertToSnapshot(snapshot)
		} else {
			if transaction.Source != "" {
				accountdb.SetNonce(common.HexToAddress(transaction.Source), transaction.Nonce)
			}

			logger.Debugf("VMExecutor Execute success %s,type:%d", transaction.Hash.String(), transaction.Type)
		}
		transactions = append(transactions, transaction)
		receipt := types.NewReceipt(nil, !success, 0)
		receipt.TxHash = transaction.Hash
		receipts = append(receipts, receipt)
	}

	// 计算定时任务（冻结退款等等）
	RefundManagerImpl.Add(refundInfos, accountdb)
	RefundManagerImpl.CheckAndMove(height, accountdb)

	// 计算出块奖励
	RewardCalculatorImpl.CalculateReward(height, accountdb)

	state := accountdb.IntermediateRoot(true)

	middleware.PerfLogger.Debugf("VMExecutor End. %s height: %d, cost: %v, txs: %d", situation, block.Header.Height, time.Since(beginTime), len(block.Transactions))
	return state, evictedTxs, transactions, receipts, nil, errs
}

func (executor *VMExecutor) validateNonce(accountdb *account.AccountDB, transaction *types.Transaction) bool {
	return true
}

//主链币充值确认
func (executor *VMExecutor) executeCoinDepositNotify(accountdb *account.AccountDB, transaction *types.Transaction) bool {
	txLogger.Tracef("Execute coin deposit ack tx:%s", transaction.ToTxJson().ToString())
	if transaction.Data == "" {
		return false
	}
	var depositCoinData types.DepositCoinData
	err := json.Unmarshal([]byte(transaction.Data), &depositCoinData)
	if err != nil {
		txLogger.Errorf("Deposit coin data unmarshal error:%s", err.Error())
		return false
	}
	txLogger.Tracef("deposit coin data:%v,target address:%s", depositCoinData, transaction.Source)
	if depositCoinData.Amount == "" || depositCoinData.ChainType == "" {
		return false
	}

	value, _ := utility.StrToBigInt(depositCoinData.Amount)
	return accountdb.AddFT(common.HexToAddress(transaction.Source), fmt.Sprintf("official-%s", depositCoinData.ChainType), value)
}

//FT充值确认
func (executor *VMExecutor) executeFTDepositNotify(accountdb *account.AccountDB, transaction *types.Transaction) bool {
	txLogger.Tracef("Execute ft deposit ack tx:%s", transaction.ToTxJson().ToString())
	if transaction.Data == "" {
		return false
	}
	var depositFTData types.DepositFTData
	err := json.Unmarshal([]byte(transaction.Data), &depositFTData)
	if err != nil {
		txLogger.Errorf("Deposit ft data unmarshal error:%s", err.Error())
		return false
	}
	txLogger.Tracef("deposit ft data:%v, address:%s", depositFTData, transaction.Source)
	if depositFTData.Amount == "" || depositFTData.FTId == "" {
		return false
	}
	//todo 先不检查此ft是否存在
	value, _ := utility.StrToBigInt(depositFTData.Amount)
	return accountdb.AddFT(common.HexToAddress(transaction.Source), depositFTData.FTId, value)
}

//NFT充值确认
func (executor *VMExecutor) executeNFTDepositNotify(accountdb *account.AccountDB, transaction *types.Transaction) bool {
	txLogger.Tracef("Execute nft deposit ack tx:%s", transaction.ToTxJson().ToString())
	if transaction.Data == "" {
		return false
	}
	var depositNFTData types.DepositNFTData
	err := json.Unmarshal([]byte(transaction.Data), &depositNFTData)
	if err != nil {
		txLogger.Errorf("Deposit nft data unmarshal error:%s", err.Error())
		return false
	}
	//todo 这里需要重写
	txLogger.Tracef("deposit nft data:%v,target address:%s", depositNFTData, transaction.Source)
	if depositNFTData.SetId == "" || depositNFTData.ID == "" {
		return false
	}

	// 检查setId
	nftSet := service.NFTManagerInstance.GetNFTSet(depositNFTData.SetId, accountdb)
	if nil == nftSet {
		nftSet = service.NFTManagerInstance.GenerateNFTSet(depositNFTData.SetId, depositNFTData.Name, depositNFTData.Symbol, depositNFTData.Creator, depositNFTData.Owner, 0, depositNFTData.CreateTime, )
		service.NFTManagerInstance.PublishNFTSet(nftSet, accountdb)
	}

	appId := transaction.Target
	str, ok := service.NFTManagerInstance.GenerateNFT(nftSet, appId, depositNFTData.SetId, depositNFTData.ID, "", depositNFTData.Creator, depositNFTData.CreateTime, "official", common.HexToAddress(transaction.Source), depositNFTData.Data, accountdb)
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

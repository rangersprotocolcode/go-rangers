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
				if !changeAssets(transaction.Target, transaction.Source, mm, accountdb) {
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
							break
						} else {
							if nil != data && 0 != len(data) {
								for _, user := range data {
									// 发币
									if user.Address == "StartFT" {
										_, flag := FTManagerInstance.PublishFTSet(user.Assets["name"], user.Assets["symbol"], user.Assets["gameId"], user.Assets["totalSupply"], 1, accountdb)
										if !flag {
											success = false
											break
										}

										continue
									}

									// 给用户币
									if user.Address == "TransferFT" {
										_, flag := TransferFT(user.Assets["gameId"], user.Assets["symbol"], user.Assets["target"], user.Assets["supply"], accountdb)
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
										_, ok := NFTManagerInstance.Transfer(appId, user.Assets["setId"], user.Assets["id"], common.HexToAddress(appId), common.HexToAddress(user.Assets["target"]), accountdb)
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
										totalSupply, _ := strconv.Atoi(user.Assets["totalSupply"])
										_, ok, _ := NFTManagerInstance.PublishNFTSet(user.Assets["setId"], user.Assets["name"], user.Assets["symbol"], uint(totalSupply), accountdb)
										if !ok {
											success = false
											break
										}
										continue
									}

									if user.Address == "MintNFT" {
										_, ok := NFTManagerInstance.MintNFT(user.Assets["appId"], user.Assets["setId"], user.Assets["id"], user.Assets["data"], common.HexToAddress(user.Assets["target"]), accountdb)
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
		case types.TransactionTypeCoinDepositAck:
			success = executor.executeCoinDepositNotify(accountdb, transaction)
		case types.TransactionTypeFTDepositAck:
			success = executor.executeFTDepositNotify(accountdb, transaction)
		case types.TransactionTypeNFTDepositAck:
			success = executor.executeNFTDepositNotify(accountdb, transaction)
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

	state := accountdb.IntermediateRoot(false)
	logger.Debugf("VMExecutor End Execute State %s", state.Hex())
	return state, evictedTxs, transactions, receipts, nil, errs
}

func (executor *VMExecutor) validateNonce(accountdb *account.AccountDB, transaction *types.Transaction) bool {
	return true
}

// 提现
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

	source := common.HexToAddress(transaction.Source)
	gameId := transaction.Target

	//余额检查
	var withdrawAmount = big.NewInt(0)
	subAccountBalance := accountdb.GetBalance(source)
	if withDrawReq.Balance != "" {
		withdrawAmount, err = utility.StrToBigInt(withDrawReq.Balance)
		if err != nil {
			txLogger.Error("Execute withdraw bad amount!Hash:%s, err:%s", transaction.Hash.String(), err.Error())
			return false
		}

		txLogger.Errorf("Execute withdraw :%s,current balance:%d,withdraw balance:%d", transaction.Hash.String(), subAccountBalance.Uint64(), withdrawAmount.Uint64())
		if subAccountBalance.Cmp(withdrawAmount) < 0 {
			txLogger.Errorf("Execute withdraw balance not enough:current balance:%d,withdraw balance:%d", subAccountBalance.Uint64(), withdrawAmount.Uint64())
			return false
		}

		//扣余额
		subAccountBalance.Sub(subAccountBalance, withdrawAmount)
		accountdb.SetBalance(source, subAccountBalance)
	}

	//ft
	if withDrawReq.FT != nil && len(withDrawReq.FT) != 0 {
		for k, v := range withDrawReq.FT {
			subValue := accountdb.GetFT(source, k)
			compareResult, sub := canWithDraw(v, subValue)
			if !compareResult {
				return false
			}

			// 扣ft
			accountdb.SetFT(source, k, sub)
		}
	}

	//nft
	nftInfo := make([]types.NFTID, 0)
	if withDrawReq.NFT != nil && len(withDrawReq.NFT) != 0 {
		for _, k := range withDrawReq.NFT {
			subValue := accountdb.GetNFTByGameId(source, gameId, k.SetId, k.Id)
			if 0 == len(subValue) {
				return false
			}

			nftInfo = append(nftInfo, types.NFTID{SetId: k.SetId, Id: k.Id, Data: subValue})

			//删除要提现的NFT
			accountdb.RemoveNFTByGameId(source, gameId, k.SetId, k.Id)
		}
	}

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
	txLogger.Debugf("deposit coin value:%s", value.String())
	accountdb.AddFT(common.HexToAddress(transaction.Source), fmt.Sprintf("official-%s", depositCoinData.ChainType), value)

	balance := accountdb.GetFT(common.HexToAddress("0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54"), "official-eth")
	txLogger.Debugf("After deposit coin,balance:%v",balance)

	balance = accountdb.GetFT(common.HexToAddress("0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54"), "official-eth")
	common.DefaultLogger.Debugf("Query Balance again:%v", balance)
	return true
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
	accountdb.AddFT(common.HexToAddress(transaction.Source), depositFTData.FTId, value)
	return true
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
	//todo 这里NFT充值的必须字段检查可能不止Value这一个字段
	if depositNFTData.Value == "" {
		return false
	}

	// 检查setId
	nftSet := NFTManagerInstance.GetNFTSet(depositNFTData.SetId, accountdb)
	if nil == nftSet {
		_, _, nftSet = NFTManagerInstance.PublishNFTSet(depositNFTData.SetId, depositNFTData.Name, depositNFTData.Symbol, 0, accountdb)
	}
	timestamp, _ := time.Parse("2006-01-02 15:04:05", depositNFTData.CreateTime)
	appId := transaction.Target
	_, ok := NFTManagerInstance.GenerateNFT(nftSet, appId, depositNFTData.SetId, depositNFTData.ID, depositNFTData.Value, depositNFTData.Creator, timestamp, common.HexToAddress(transaction.Source), accountdb)
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

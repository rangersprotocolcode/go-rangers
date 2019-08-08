package core

import (
	"x/src/common"
	"x/src/middleware/types"
	"strconv"
	"math/big"
	"encoding/json"
	"x/src/storage/account"
)

func StartFT(gameId string, name string, symbol string, totalSupply string, accountDB *account.AccountDB) (string, bool) {
	if 0 == len(gameId) || 0 == len(name) || 0 == len(symbol) || 0 == len(totalSupply) {
		return "wrong params", false
	}

	gameAddress := common.HexToAddress(gameId)
	ftList := accountDB.GetSubAccount(gameAddress, "ft")
	// 相同名字的已经存在了
	if 0 != len(ftList.Ft[symbol]) {
		return "wrong symbol", false
	}

	data := &types.FtInitialization{
		TotalSupply: totalSupply,
		Name:        name,
	}
	supply, _ := strconv.ParseFloat(totalSupply, 64)
	remain := big.NewInt(int64(supply * 1000000000))
	if -1 == remain.Sign() {
		return "wrong supply", false
	}
	data.Remain = remain.String()

	ftBytes, _ := json.Marshal(data)
	ftList.Ft[symbol] = string(ftBytes)

	accountDB.UpdateSubAccount(gameAddress, "ft", *ftList)

	return "ft start successful", true
}

// todo: 经济模型，转币的费用问题
// 状态机转币给玩家
func TransferFT(gameId string, symbol string, target string, supply string, accountDB *account.AccountDB) (string, bool) {
	if 0 == len(gameId) || 0 == len(target) || 0 == len(symbol) || 0 == len(supply) {
		return "wrong params", false
	}

	gameAddress := common.HexToAddress(gameId)
	ftList := accountDB.GetSubAccount(gameAddress, "ft")
	ftString := ftList.Ft[symbol]
	if 0 == len(ftString) {
		return "wrong symbol", false
	}

	var ftInitialization types.FtInitialization
	json.Unmarshal([]byte(ftString), &ftInitialization)

	remain := ftInitialization.ConvertWithoutBase(ftInitialization.Remain)
	want := ftInitialization.Convert(supply)
	if -1 == remain.Cmp(want) {
		return "not enough", false
	}

	targetAddress := common.HexToAddress(target)
	targetAccount := accountDB.GetSubAccount(targetAddress, gameId)

	targetValue := ftInitialization.ConvertWithoutBase(targetAccount.Ft[symbol])
	targetLeft := targetValue.Add(targetValue, want)
	targetAccount.Ft[symbol] = targetLeft.String()

	left := remain.Add(remain, want.Mul(want, minusOne))
	ftInitialization.Remain = left.String()
	ftBytes, _ := json.Marshal(ftInitialization)
	ftList.Ft[symbol] = string(ftBytes)

	accountDB.UpdateSubAccount(gameAddress, "ft", *ftList)

	return "TransferFT successful", true
}

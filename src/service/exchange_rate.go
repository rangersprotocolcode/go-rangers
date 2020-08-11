package service

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"encoding/json"
	"fmt"
	"strings"
)

func SetExchangeRate(accountdb *account.AccountDB, transaction *types.Transaction) (bool, string) {
	if nil == transaction || nil == accountdb {
		return false, ""
	}

	if !checkAuth(transaction.Source) {
		return false, fmt.Sprintf("%s not allowed", transaction.Source)
	}

	var rates map[string]string
	err := json.Unmarshal([]byte(transaction.Data), &rates)
	if err != nil {
		return false, err.Error()
	}

	if 0 == len(rates) {
		return true, ""
	}

	for key, value := range rates {
		if 0 == len(key) {
			continue
		}

		if 0 == len(value) || 0 == strings.Compare("0", value) {
			accountdb.RemoveData(common.ExchangeRateAddress, []byte(key))
			continue
		}
		accountdb.SetData(common.ExchangeRateAddress, []byte(key), []byte(value))
	}
	return true, ""
}

func checkAuth(source string) bool {
	return 0 == strings.Compare(source, "0")
}

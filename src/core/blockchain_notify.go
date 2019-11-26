package core

import (
	"x/src/middleware/types"
	"x/src/network"
	"encoding/json"
	"strconv"
)

func (chain *blockChain) notifyWallet(remoteBlock *types.Block) {
	txs := remoteBlock.Transactions
	if nil == txs || 0 == len(txs) {
		return
	}

	for _, tx := range txs {
		switch tx.Type {
		case types.TransactionTypeCoinDepositAck:
			var depositCoinData types.DepositCoinData
			json.Unmarshal([]byte(tx.Data), &depositCoinData)

			data := make(map[string]interface{})
			data["from"] = depositCoinData.MainChainAddress
			data["to"] = tx.Source
			data["token"] = depositCoinData.ChainType
			value, _ := strconv.ParseFloat(depositCoinData.Amount, 64)
			data["value"] = value
			data["hash"] = depositCoinData.TxId

			chain.doNotify("deposit_bnt", data)
			break
		case types.TransactionTypeFTDepositAck:
			var depositFTData types.DepositFTData
			json.Unmarshal([]byte(tx.Data), &depositFTData)

			data := make(map[string]interface{})
			data["from"] = depositFTData.MainChainAddress
			data["to"] = tx.Source
			data["setId"] = depositFTData.FTId
			value, _ := strconv.ParseFloat(depositFTData.Amount, 64)
			data["value"] = value
			data["contract"] = depositFTData.ContractAddress
			data["hash"] = depositFTData.TxId

			chain.doNotify("deposit_ft", data)
			break
		case types.TransactionTypeNFTDepositAck:
			var depositNFTData types.DepositNFTData
			json.Unmarshal([]byte(tx.Data), &depositNFTData)

			data := make(map[string]interface{})
			data["from"] = depositNFTData.MainChainAddress
			data["to"] = tx.Source
			data["setId"] = depositNFTData.SetId
			data["tokenId"] = depositNFTData.ID
			data["contract"] = depositNFTData.ContractAddress
			data["hash"] = depositNFTData.TxId

			chain.doNotify("deposit_nft", data)
			break
		case types.TransactionTypeOperatorEvent:
			if nil != tx.SubTransactions && 0 != len(tx.SubTransactions) {
				for _, sub := range tx.SubTransactions {
					if sub.Address != "UpdateNFT" {
						continue
					}

					data := make(map[string]interface{})
					data["appId"] = sub.Assets["appId"]
					data["owner"] = sub.Assets["addr"]
					data["setId"] = sub.Assets["setId"]
					data["tokenId"] = sub.Assets["id"]
					data["data"] = sub.Assets["data"]

					chain.doNotify("nft_update", data)
				}
			}
			break

		}

	}
}

func (chain *blockChain) doNotify(method string, data map[string]interface{}) {
	var notify types.DepositNotify
	notify.Method = method
	notify.Data = data

	result, _ := json.Marshal(notify)
	network.GetNetInstance().Notify(false, "wallet", "wallet", string(result))
}
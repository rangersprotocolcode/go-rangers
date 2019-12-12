package core

import (
	"x/src/middleware/types"
	"x/src/network"
	"encoding/json"
	"strconv"
	"strings"
)

func (chain *blockChain) notifyWallet(remoteBlock *types.Block) {
	txs := remoteBlock.Transactions
	if nil == txs || 0 == len(txs) {
		return
	}

	block := remoteBlock.Header.Height
	events := make([]types.DepositNotify, 0)

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

			events = append(events, chain.generateWalletNotify("deposit_bnt", data))
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

			events = append(events, chain.generateWalletNotify("deposit_ft", data))
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

			events = append(events, chain.generateWalletNotify("deposit_nft", data))
			break
		case types.TransactionTypeWithdraw:
			chain.notifyWithDrawInfo(tx, &events)

		case types.TransactionTypeShuttleNFT:
			chain.notifyShuttleNFT(tx, &events)

		case types.TransactionTypeOperatorEvent:
			if 0 != len(tx.ExtraData) {
				chain.notifyTransferInfo(tx, &events)
			}

			if nil != tx.SubTransactions && 0 != len(tx.SubTransactions) {
				for _, sub := range tx.SubTransactions {
					if sub.Address == "UpdateNFT" {
						data := make(map[string]interface{})
						data["appId"] = sub.Assets["appId"]
						data["owner"] = sub.Assets["addr"]
						data["setId"] = sub.Assets["setId"]
						data["tokenId"] = sub.Assets["id"]
						data["data"] = sub.Assets["data"]

						events = append(events, chain.generateWalletNotify("nft_update", data))
						continue
					}

					if sub.Address == "TransferNFT" {
						data := make(map[string]interface{})
						data["from"] = sub.Assets["appId"]
						data["to"] = sub.Assets["target"]
						data["setId"] = sub.Assets["setId"]
						data["tokenId"] = sub.Assets["id"]
						events = append(events, chain.generateWalletNotify("transfer_nft", data))
						continue
					}

					if sub.Address == "TransferFT" && sub.Assets["symbol"] != "" {
						data := make(map[string]interface{})
						data["from"] = sub.Assets["gameId"]
						data["to"] = sub.Assets["target"]
						data["value"], _ = strconv.ParseFloat(sub.Assets["supply"], 64)
						if strings.HasPrefix(sub.Assets["symbol"], "official-") {
							data["token"] = strings.Split(sub.Assets["symbol"], "-")[1]
							events = append(events, chain.generateWalletNotify("transfer_bnt", data))
						} else {
							data["setId"] = sub.Assets["symbol"]
							events = append(events, chain.generateWalletNotify("transfer_ft", data))
						}
						continue
					}

				}
			}
			break

		}

	}

	if 0 != len(events) {
		notify := make(map[string]interface{})
		notify["block"] = block
		notify["events"] = events
		result, _ := json.Marshal(notify)
		network.GetNetInstance().Notify(false, "wallet", "wallet", string(result))
	}
}

func (chain *blockChain) generateWalletNotify(method string, data map[string]interface{}) types.DepositNotify {
	var notify types.DepositNotify
	notify.Method = method
	notify.Data = data

	return notify

}

func (chain *blockChain) notifyTransferInfo(tx *types.Transaction, events *[]types.DepositNotify) {
	transferDataMap := make(map[string]types.TransferData, 0)
	if err := json.Unmarshal([]byte(tx.ExtraData), &transferDataMap); nil != err {
		logger.Debugf("json unmarshal transfer data map error:%s", err.Error())
		return
	}

	for targetAddress, transferData := range transferDataMap {
		//BNT
		if transferData.Coin != nil && len(transferData.Coin) > 0 {
			for bntType, bntValue := range transferData.Coin {
				data := make(map[string]interface{})
				data["from"] = tx.Source
				data["to"] = targetAddress
				data["token"] = bntType
				data["value"], _ = strconv.ParseFloat(bntValue, 64)
				*events = append(*events, chain.generateWalletNotify("transfer_bnt", data))
			}
		}

		//FT
		if transferData.FT != nil && len(transferData.FT) > 0 {
			for ftSetId, ftValue := range transferData.FT {
				data := make(map[string]interface{})
				data["from"] = tx.Source
				data["to"] = targetAddress
				data["setId"] = ftSetId
				data["value"], _ = strconv.ParseFloat(ftValue, 64)
				*events = append(*events, chain.generateWalletNotify("transfer_ft", data))
			}
		}

		//NFT
		if transferData.NFT != nil && len(transferData.NFT) > 0 {
			for _, nft := range transferData.NFT {
				data := make(map[string]interface{})
				data["from"] = tx.Source
				data["to"] = targetAddress
				data["setId"] = nft.SetId
				data["tokenId"] = nft.Id
				*events = append(*events, chain.generateWalletNotify("transfer_nft", data))
			}
		}
	}

}

func (chain *blockChain) notifyWithDrawInfo(tx *types.Transaction, events *[]types.DepositNotify) {
	var withDrawReq types.WithDrawReq
	err := json.Unmarshal([]byte(tx.Data), &withDrawReq)
	if err != nil {
		return
	}

	//BNT
	if withDrawReq.BNT.TokenType != "" {
		data := make(map[string]interface{})
		data["from"] = tx.Source
		data["to"] = withDrawReq.Address
		data["token"] = withDrawReq.BNT.TokenType
		data["value"], _ = strconv.ParseFloat(withDrawReq.BNT.Value, 64)
		data["status"] = 0
		*events = append(*events, chain.generateWalletNotify("withdraw_bnt", data))
	}

	//ft
	if withDrawReq.FT != nil && len(withDrawReq.FT) != 0 {
		for k, v := range withDrawReq.FT {
			data := make(map[string]interface{})
			data["from"] = tx.Source
			data["to"] = withDrawReq.Address
			data["chainType"] = withDrawReq.ChainType
			data["setId"] = k
			data["value"], _ = strconv.ParseFloat(v, 64)
			data["status"] = 0
			*events = append(*events, chain.generateWalletNotify("withdraw_ft", data))
		}
	}

	//nft
	if withDrawReq.NFT != nil && len(withDrawReq.NFT) != 0 {
		for _, k := range withDrawReq.NFT {
			data := make(map[string]interface{})
			data["from"] = tx.Source
			data["to"] = withDrawReq.Address
			data["chainType"] = withDrawReq.ChainType
			data["setId"] = k.SetId
			data["tokenId"] = k.Id
			data["status"] = 0
			*events = append(*events, chain.generateWalletNotify("withdraw_nft", data))
		}
	}
}

func (chain *blockChain) notifyShuttleNFT(tx *types.Transaction, events *[]types.DepositNotify) {
	shuttleData := make(map[string]string)
	json.Unmarshal([]byte(tx.Data), &shuttleData)

	data := make(map[string]interface{})
	data["setId"] = shuttleData["setId"]
	data["tokenId"] = shuttleData["id"]
	data["toAppId"] = shuttleData["newAppId"]

	data["owner"] = tx.Source
	//这两个字段没有
	data["fromAppId"] = ""
	data["data"] = ""
	*events = append(*events, chain.generateWalletNotify("shuttle", data))
}

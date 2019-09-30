package types

import (
	"testing"
	"fmt"
	"encoding/json"
)

//func TestAssetOnChainTransactionHash(t *testing.T) {
//	a := []string{"drogon"}
//	b, err := json.Marshal(a)
//	if err != nil {
//		fmt.Printf("Json marshal []string err:%s", err.Error())
//		return
//	}
//	str := string(b)
//
//	txJson := TxJson{Source: "dragonMother", Target: "tuntunbiu", Type: 202, Data: str, Nonce: 3, Time: "1556076659050692000"}
//	tx := txJson.ToTransaction()
//	tx.Hash = tx.GenHash()
//
//	j, _ := json.Marshal(tx.ToTxJson())
//	fmt.Printf("TX JSON:\n%s\n", string(j))
//
//}
//
//func TestUpgradeDragonTransactionHash(t *testing.T) {
//	txJson := TxJson{Target: "tuntunbiu", Type: 8, Data: "u", Nonce: 6,}
//	tx := txJson.ToTransaction()
//	tx.Hash = tx.GenHash()
//
//	j, _ := json.Marshal(tx.ToTxJson())
//	fmt.Printf("TX JSON:\n%s\n", string(j))
//}
//
//func TestWithdrawTransactionHash(t *testing.T) {
//	txJson := TxJson{Source: "dragonMother", Target: "tuntunbiu", Type: 201, Data: "1.35", Nonce: 1, Time: "1556076659050692000"}
//	tx := txJson.ToTransaction()
//	tx.Hash = tx.GenHash()
//
//	j, _ := json.Marshal(tx.ToTxJson())
//	fmt.Printf("TX JSON:\n%s\n", string(j))
//}
//
//func TestTransactionHash(t *testing.T) {
//	txJson := TxJson{Source: "dragonMother", Target: "tuntunbiu", Type: 11, Time: "1556076659050692000"}
//	tx := txJson.ToTransaction()
//	tx.Hash = tx.GenHash()
//
//	j, _ := json.Marshal(tx.ToTxJson())
//	fmt.Printf("TX JSON:\n%s\n", string(j))
//}
//
//func TestNonceTransaction(t *testing.T) {
//	txJson := TxJson{Source: "0x00113898717aafe49f28ca587219e1188550edfb", Target: "appid_demo_1", Type: 14, Time: "1556076659050692000"}
//	tx := txJson.ToTransaction()
//	tx.Hash = tx.GenHash()
//
//	j, _ := json.Marshal(tx.ToTxJson())
//	fmt.Printf("TX JSON:\n%s\n", string(j))
//}
//
///**
//layer2 web socket 接口测试交易生成
// */
//
//func TestGetBalanceTx(t *testing.T) {
//	tx := Transaction{Source: "0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54", Target: "0xf677e4051eeff7a60598cc6419b982cdeef60b01", Type: TransactionTypeGetCoin, Time: "1556076659050692000", SocketRequestId: "12134"}
//	tx.Hash = tx.GenHash()
//
//	j, _ := json.Marshal(tx.ToTxJson())
//	fmt.Printf("TX JSON:\n%s\n", string(j))
//}
//
//func TestOneAssetTx(t *testing.T) {
//	tx := Transaction{Source: "0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54", Target: "0xf677e4051eeff7a60598cc6419b982cdeef60b01", Type: TransactionTypeNFT, Time: "1556076659050692000", SocketRequestId: "12135", Data: "xxx"}
//	tx.Hash = tx.GenHash()
//
//	j, _ := json.Marshal(tx.ToTxJson())
//	fmt.Printf("TX JSON:\n%s\n", string(j))
//}
//
//func TestGetStateMachineNonceTx(t *testing.T) {
//	tx := Transaction{Source: "0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54", Target: "0xf677e4051eeff7a60598cc6419b982cdeef60b01", Type: TransactionTypeStateMachineNonce, Time: "1556076659050692000", SocketRequestId: "12138"}
//	tx.Hash = tx.GenHash()
//
//	j, _ := json.Marshal(tx.ToTxJson())
//	fmt.Printf("TX JSON:\n%s\n", string(j))
//}
//
//func TestWithdrawTx(t *testing.T) {
//	tx := Transaction{Source: "0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54", Target: "0xf677e4051eeff7a60598cc6419b982cdeef60b01", Type: TransactionTypeWithdraw, Time: "1556076659050692000", SocketRequestId: "12139"}
//
//	ft := make(map[string]string)
//	ft["ftId1"] = "2.56"
//	ft["ftId2"] = "5.99"
//
//	nft := make([]NFTID, 0)
//	nft = append(nft, NFTID{SetId: "test1", Id: "test2"})
//
//	req := WithDrawReq{Balance: "11.12", ChainType: "ETH", Address: "0xf3426Ae90e962f49D71307DB309535815e16808f", FT: ft, NFT: nft}
//	b, _ := json.Marshal(req)
//	tx.Data = string(b)
//	fmt.Printf("data:\n%s\n", tx.Data)
//
//	tx.Hash = tx.GenHash()
//
//	j, _ := json.Marshal(tx.ToTxJson())
//	fmt.Printf("TX JSON:\n%s\n", string(j))
//}
//
//func TestOperateTx(t *testing.T) {
//	tx := Transaction{Source: "0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54", Target: "0xf677e4051eeff7a60598cc6419b982cdeef60b01", Type: TransactionTypeOperatorEvent, Time: "1556076659050692000", SocketRequestId: "12140"}
//	tx.ExtraData = "{\"msg_name\":\"lottery_balance\"}"
//
//	ft := make(map[string]string)
//	ft["ftId1"] = "2.56"
//	ft["ftId2"] = "5.99"
//
//	nft := make([]NFTID, 0)
//	nft = append(nft, NFTID{SetId: "abc", Id: "123"})
//
//	req := TransferData{Balance: "11.12", FT: ft, NFT: nft}
//	b, _ := json.Marshal(req)
//	tx.Data = string(b)
//	fmt.Printf("data:\n%s\n", tx.Data)
//
//	tx.Hash = tx.GenHash()
//
//	j, _ := json.Marshal(tx.ToTxJson())
//	fmt.Printf("TX JSON:\n%s\n", string(j))
//}

func TestMintNFTTx(t *testing.T) {
	tx := Transaction{Source: "0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54", Target: "0xb0da465fbc3eab96e68151625d504ef1946b9446", Type: TransactionTypeMintNFT, Time: "1556076659050692000", SocketRequestId: "12140"}

	mintNFTInfo:= make(map[string]string)
	mintNFTInfo["setId"] = "23c4233d-5407-4ed1-a342-16a13cbb33a1"
	mintNFTInfo["id"] = "123456"
	mintNFTInfo["data"] = "5.99"
	mintNFTInfo["createTime"] = "1569736452602"
	mintNFTInfo["target"] = "0xb0da465fbc3eab96e68151625d504ef1946b9446"

	b, _ := json.Marshal(mintNFTInfo)
	tx.Data = string(b)
	fmt.Printf("data:\n%s\n", tx.Data)

	tx.Hash = tx.GenHash()

	j, _ := json.Marshal(tx.ToTxJson())
	fmt.Printf("TX JSON:\n%s\n", string(j))
}

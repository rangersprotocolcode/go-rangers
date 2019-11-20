package types

import (
	"testing"
	"fmt"
	"encoding/json"
)
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


func TestQueryBNTBalanceTx(t *testing.T) {
	tx := Transaction{Source: "0x6ed3a2ea39e1774096de4d920b4fb5b32d37fa98", Target: "0x6ed3a2ea39e1774096de4d920b4fb5b32d37fa98", Type: TransactionTypeGetCoin, Time: "1556076659050692000", SocketRequestId: "12140"}
	tx.Data = string("ETH.ETH")
	fmt.Printf("data:\n%s\n", tx.Data)

	tx.Hash = tx.GenHash()

	j, _ := json.Marshal(tx.ToTxJson())
	fmt.Printf("TX JSON:\n%s\n", string(j))
}

func TestMintNFTTx(t *testing.T) {
	tx := Transaction{Source: "0x945dbcff35562688388c74ad5084746abc9c8341", Target: "0x945dbcff35562688388c74ad5084746abc9c8341", Type: TransactionTypeMintNFT, Time: "1556076659050692000", SocketRequestId: "12140"}

	mintNFTInfo:= make(map[string]string)
	mintNFTInfo["setId"] = "75d8b6d6-6763-49b5-a5fc-b9a22ca5be5a"
	mintNFTInfo["id"] = "12347"
	mintNFTInfo["data"] = "5.99"
	mintNFTInfo["createTime"] = "1569736452603"
	mintNFTInfo["target"] = "0x945dbcff35562688388c74ad5084746abc9c8341"

	b, _ := json.Marshal(mintNFTInfo)
	tx.Data = string(b)
	fmt.Printf("data:\n%s\n", tx.Data)

	tx.Hash = tx.GenHash()

	j, _ := json.Marshal(tx.ToTxJson())
	fmt.Printf("TX JSON:\n%s\n", string(j))
}

func TestOutputMessage(t *testing.T) {
	output := OutputMessage{}
	fmt.Printf("%v\n", output)
}


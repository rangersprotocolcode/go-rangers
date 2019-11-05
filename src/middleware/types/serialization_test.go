package types

import (
	"testing"
	"x/src/middleware/pb"
	"encoding/json"
	"fmt"
)

func TestPbToBlockHeader(t *testing.T) {
	header := BlockHeader{}
	header.RequestIds = make(map[string]uint64)
	header.RequestIds["1"] = 1024

	pb := middleware_pb.BlockHeader{}
	pb.RequestIds, _ = json.Marshal(header.RequestIds)

	header2 := BlockHeader{}
	json.Unmarshal(pb.RequestIds, &header2.RequestIds)

	fmt.Println(header2.RequestIds["1"])

	fmt.Println(header2.RequestIds["2"] < 2)

}

func TestTxHash(t *testing.T) {
	tx := Transaction{}
	tx.Source = "aaa"
	tx.Target = "bbb"
	tx.Type = 0
	tx.Time = "1572940024276"
	tx.Data = "dafadfa"
	tx.Nonce = 1234

	hash := tx.GenHash()
	fmt.Printf("Tx Hash:%x\n", hash)
	fmt.Printf("Tx Hash:%s\n", hash.String())
}

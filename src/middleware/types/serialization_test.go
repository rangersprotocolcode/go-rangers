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

	fmt.Println(header2.RequestIds["2"]<2)

}

package core

import (
	"testing"
	"fmt"
	"x/src/common"
	"math/big"
	"encoding/json"
)

func TestVMExecutor_Execute(t *testing.T) {
	context := make(map[string]interface{})
	context["refund"] = make(map[uint64]RefundInfoList)
	data, _ := json.Marshal(context)
	fmt.Printf("before test, context: %s\n", string(data))

	testMap(context)

	data, _ = json.Marshal(context)
	fmt.Printf("after test, context: %s\n", context)

	refundInfos := getRefundInfo(context)
	refundHeight := uint64(999)
	minerId := common.FromHex("0x0001")
	money := big.NewInt(100)
	refundInfo, ok := refundInfos[refundHeight]
	if ok {
		fmt.Println(string(refundInfo.TOJSON()))
		refundInfo.AddRefundInfo(minerId, money)
	} else {
		refundInfo = RefundInfoList{}
		refundInfo.AddRefundInfo(minerId, money)
		refundInfos[refundHeight] = refundInfo
	}

	a := getRefundInfo(context)
	refundInfo, ok = a[refundHeight]
	if ok {
		fmt.Println(string(refundInfo.TOJSON()))
	}

}

func testMap(context map[string]interface{}) {
	refundInfos := getRefundInfo(context)
	refundHeight := uint64(999)
	minerId := common.FromHex("0x0001")
	money := big.NewInt(100)
	refundInfo, ok := refundInfos[refundHeight]
	if ok {
		refundInfo.AddRefundInfo(minerId, money)
	} else {
		refundInfo = RefundInfoList{}
		refundInfo.AddRefundInfo(minerId, money)
		refundInfos[refundHeight] = refundInfo
	}

	//fmt.Println(context)
	//fmt.Println(refundInfos)
}

package core

import (
	"testing"
	"x/src/utility"
	"math/big"
	"fmt"
	"encoding/json"
	"x/src/common"
	"sort"
)

func TestRefundInfoList_AddRefundInfo(t *testing.T) {
	list := RefundInfoList{}

	list.AddRefundInfo(utility.UInt64ToByte(100), big.NewInt(2000))
	fmt.Println(string(list.TOJSON()))
	list.AddRefundInfo(utility.UInt64ToByte(100), big.NewInt(2000))
	fmt.Println(string(list.TOJSON()))
	list.AddRefundInfo(utility.UInt64ToByte(100), big.NewInt(2000))
	fmt.Println(string(list.TOJSON()))

	list.AddRefundInfo(utility.UInt64ToByte(200), big.NewInt(9999))
	fmt.Println(string(list.TOJSON()))

}

func TestRefundInfoList_TOJSON(t *testing.T) {
	str := `{"List":[{"Value":6000,"Id":"AAAAAAAAAGQ="},{"Value":9999,"Id":"AAAAAAAAAMg="}]}`

	var refundInfoList RefundInfoList
	err := json.Unmarshal([]byte(str), &refundInfoList)
	if err != nil {
		fmt.Println(err.Error())
	}

	for i, refundInfo := range refundInfoList.List {
		fmt.Printf("%d: value: %d, id: %s\n", i, refundInfo.Value, common.ToHex(refundInfo.Id))
	}
}

func TestDismissHeightList_Len(t *testing.T) {
	dismissHeightList := DismissHeightList{}
	dismissHeightList = append(dismissHeightList, 1000)
	dismissHeightList = append(dismissHeightList, 200)
	dismissHeightList = append(dismissHeightList, 2000)

	fmt.Println(dismissHeightList)

	sort.Sort(dismissHeightList)
	fmt.Println(dismissHeightList)
	fmt.Println(dismissHeightList[0])
}

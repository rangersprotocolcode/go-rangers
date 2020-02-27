package groupsig

import (
	"testing"
	"fmt"
	"math/big"
)

//测试从big.Int生成ID，以及ID的序列化
func TestID(t *testing.T) {
	t.Log("testString")
	fmt.Printf("\nbegin test ID...\n")
	b := new(big.Int)
	b.SetString("001234567890abcdef", 16)
	c := new(big.Int)
	c.SetString("1234567890abcdef", 16)
	idc := NewIDFromBigInt(c)
	id1 := NewIDFromBigInt(b) //从big.Int生成ID
	if id1.IsEqual(*idc) {
		fmt.Println("id1 is equal to idc")
	}
	if id1 == nil {
		t.Error("NewIDFromBigInt")
	} else {
		buf := id1.Serialize()
		fmt.Printf("id Serialize, len=%v, data=%v.\n", len(buf), buf)
	}

	str := id1.GetHexString()
	fmt.Printf("ID export, len=%v, data=%v.\n", len(str), str)

	str0 := id1.value.GetHexString()
	fmt.Printf("str0 =%v\n", str0)

	{
		var id2 ID
		err := id2.SetHexString(id1.GetHexString()) //测试ID的十六进制导出和导入功能
		if err != nil || !id1.IsEqual(id2) {
			t.Errorf("not same\n%s\n%s", id1.GetHexString(), id2.GetHexString())
		}
	}

	{
		var id2 ID
		err := id2.Deserialize(id1.Serialize()) //测试ID的序列化和反序列化
		fmt.Printf("id2:%v", id2.GetHexString())
		if err != nil || !id1.IsEqual(id2) {
			t.Errorf("not same\n%s\n%s", id1.GetHexString(), id2.GetHexString())
		}
	}
	fmt.Printf("end test ID.\n")
}
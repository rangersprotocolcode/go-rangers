package storestate

import (
	"testing"
	"math/big"
	"fmt"
)

func Test_db(t *testing.T) {
	var ss StoreState
	ss.Init()

	//ss.AddGame("eth","123456","0x9Ebf525Cbe1Fb9Dd210F544d21Ef29e4f8311097")
	//addresss,_ := ss.GetAllGames()
	//for k,v := range addresss {
	//	fmt.Println(k,v)
	//}

	//ss.AddGame("eth","云斗龙","0xabcde")
	//ss.DelGame("eth","0xabcde")
	id,_ :=ss.AddInfo("eth","0x11111","0xaaaaa",[]byte("abcdefg"))
	ss.UpdatePending(id,*big.NewInt(123),"0xhhhhhhhhhhhhhhhhhhhh")
	ss.UpdateTransfered("0xhhhhhhhhhhhhhhhhhhhh",*big.NewInt(456),*big.NewInt(789),"logloglogloglog","topicstopicstopics")
	ss.UpdateFinished("0xhhhhhhhhhhhhhhhhhhhh",*big.NewInt(321))


	ss.Deinit()
}
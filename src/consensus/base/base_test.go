package base

import (
	"testing"
	"sync/atomic"
	"regexp"
	"fmt"
	"strings"
	"time"

	"x/src/common"
)

type something struct {
	ptr atomic.Value
}

func (st *something) setPrt(v *int) {
	//atomic.StorePointer(&st.ptr, unsafe.Pointer(&v))
	st.ptr.Store(v)
}

func (st *something) getPrt() *int {
	//return (*int)(atomic.LoadPointer(&st.ptr))
	return st.ptr.Load().(*int)
}

func TestAtomicPtr(t *testing.T) {
	sth := &something{}
	a := 100
	sth.setPrt(&a)
	l := sth.getPrt()
	t.Log(sth.ptr, *l)

	sth.setPrt(nil)
	l = sth.getPrt()
	if l != nil {
		t.Log(*sth.getPrt())
	} else {
		t.Log("nil")
	}
}

func TestRegex(t *testing.T) {
	prefix := "GetIDPrefix"
	data := prefix + "()"
	re2, _ := regexp.Compile(prefix + "\\((.*?)\\)")

	//FindSubmatch查找子匹配项
	sub := re2.FindSubmatch([]byte(data))
	//第一个匹配的是全部元素
	fmt.Println(string(sub[0]))
	//第二个匹配的是第一个()里面的
	fmt.Println(string(sub[1]))

	s := strings.Replace(data, data, string(sub[1])+".ShortS()", 1)
	fmt.Printf(s)

}

func TestTimeAdd(t *testing.T) {
	now := time.Now()
	b := now.Add(-time.Second * time.Duration(10))
	t.Log(b)
}

func TestHashEqual(t *testing.T) {
	h := common.BytesToHash([]byte("123"))
	h2 := common.BytesToHash([]byte("123"))
	h3 := common.BytesToHash([]byte("234"))

	t.Log(h == h2, h == h3)
}

// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package base

import (
	"com.tuntun.rangers/node/src/utility"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"com.tuntun.rangers/node/src/common"
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
	now := utility.GetTime()
	b := now.Add(-time.Second * time.Duration(10))
	t.Log(b)
}

func TestHashEqual(t *testing.T) {
	h := common.BytesToHash([]byte("123"))
	h2 := common.BytesToHash([]byte("123"))
	h3 := common.BytesToHash([]byte("234"))

	t.Log(h == h2, h == h3)
}

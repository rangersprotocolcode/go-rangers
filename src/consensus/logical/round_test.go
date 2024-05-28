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

package logical

import (
	"fmt"
	"testing"
)

type testRound interface {
	Pr()
}

type (
	testBaseRound struct {
	}
	testRound1 struct {
		*testBaseRound
	}
	testRound2 struct {
		*testRound1
	}
)

func (r *testBaseRound) Pr() {
	fmt.Println("testBaseRound")
}
func (r *testRound1) Pr() {
	fmt.Println("testRound1")
}
func (r *testRound2) Pr() {
	fmt.Println("testRound2")
	r.testRound1.Pr()
}

func TestBaseRound_Pr(t *testing.T) {
	var tr testRound
	tr = &testBaseRound{}
	tr.Pr()

	tr = &testRound1{}
	tr.Pr()

	tr = &testRound2{}
	tr.Pr()
}

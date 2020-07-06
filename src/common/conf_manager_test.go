// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package common

import "testing"

var (
	PATH = "tas_test.ini"
	cm   = NewConfINIManager(PATH)
)

func TestConfFileManager_SetBool(t *testing.T) {

	cm.SetBool("teSt_1", "bool_1", true)
	cm.SetDouble("test_2", "double_1", 10.33)
	cm.SetString("TTT", "STR", "abc好的")

	t.Log(cm.GetBool("test_1", "netId", true))
	t.Log(cm.GetDouble("test_2", "double_1", 100))
	t.Log(cm.GetString("test_2", "str1", "sss"))
	t.Log(cm.GetString("test_2", "str2", "223"))
	t.Log(cm.GetString("ttt", "ID", "dDDD"))

	sm := cm.GetSectionManager("test_2")
	t.Log(sm.GetString("str1", "sss"))
	sm.SetDouble("d1", 2932)
	sm.SetString("abc", "DBU")
	sm.Del("str2")
}

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

package mysql

type TxRaw struct {
	Nonce  uint64
	UserId string
	Data   string
}

func GetTxRaws(start uint64) []TxRaw {
	if MysqlDB == nil {
		return nil
	}

	rows, err := MysqlDB.Query("SELECT id,userid,data FROM `tx_raw` where id>? limit 100", start)
	defer func() {
		if nil != rows {
			rows.Close()
		}
	}()

	if nil != err {
		logger.Errorf(err.Error())
		return nil
	}

	result := make([]TxRaw, 0)
	var txRaw TxRaw
	for rows.Next() {
		err := rows.Scan(&txRaw.Nonce, &txRaw.UserId, &txRaw.Data)
		if nil != err {
			logger.Errorf(err.Error())
			return nil
		}
		result = append(result, txRaw)
	}

	if 0 != len(result) {
		logger.Debugf("receive nonce from %d to %d", result[0].Nonce, result[len(result)-1].Nonce)
	}
	return result
}

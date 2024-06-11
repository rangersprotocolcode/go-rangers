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

package mysql

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/middleware/types"
)

func DeleteGroup(id []byte) {
	psmt, err := mysqlDBLog.Prepare("DELETE FROM groupIndex WHERE hash = ?")
	if err != nil {
		logger.Error(err)
		return
	}
	defer psmt.Close()

	gid := common.ToHex(id)
	result, err := psmt.Exec(gid)
	if err != nil {
		logger.Error(err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	logger.Debugf("deleted group: %s, rowsAffected %d", gid, rowsAffected)
}

func CountGroups() int {
	sql := "select count(*) as num FROM groupIndex "
	rows, err := mysqlDBLog.Query(sql)
	if err != nil {
		return -1
	}
	defer rows.Close()

	var result int
	for rows.Next() {
		err = rows.Scan(&result)
		if err != nil {
			logger.Errorf("scan groups failed, err: %v", err)
			return -1
		}

		return result
	}

	return -1
}

func SelectGroups(height uint64) []string {
	sql := "select hash FROM groupIndex where workheight<=? and dismissheight>= ? order by groupheight"
	rows, err := mysqlDBLog.Query(sql, height, height)
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := make([]string, 0)
	for rows.Next() {
		var value string
		err = rows.Scan(&value)
		if err != nil {
			logger.Errorf("select groups failed, err: %v", err)
			return nil
		}

		result = append(result, value)
	}

	return result
}

func InsertGroup(group *types.Group) {
	id := common.ToHex(group.Id)
	workHeight := group.Header.WorkHeight
	dismissheight := group.Header.DismissHeight
	groupheight := group.GroupHeight

	logger.Infof("start insert group: %s, workHeight: %d, dismissheight: %d, groupheight: %d", id, workHeight, dismissheight, groupheight)

	tx, err := mysqlDBLog.Begin()
	if err != nil {
		logger.Errorf("fail to begin. group: %s, workHeight: %d, dismissheight: %d, groupheight: %d", id, workHeight, dismissheight, groupheight)
		return
	}
	defer tx.Commit()

	sqlStr := "replace INTO groupIndex(hash,workheight,dismissheight, groupheight) values (?, ?, ?, ?)"
	stmt, err := tx.Prepare(sqlStr)
	if err != nil {
		logger.Errorf("fail to prepare. group: %s, workHeight: %d, dismissheight: %d, groupheight: %d", id, workHeight, dismissheight, groupheight)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(id, workHeight, dismissheight, groupheight)
	if err != nil {
		logger.Errorf("fail to insert exec, err: %s. sql: %s", err, sqlStr)
		return
	}
	logger.Infof("inserted group: %s, workHeight: %d, dismissheight: %d, groupheight: %d", id, workHeight, dismissheight, groupheight)
}

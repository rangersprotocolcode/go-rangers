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
	"math"
)

func DeleteGroup(id []byte) error {
	psmt, err := mysqlDBLog.Prepare("DELETE FROM groupIndex WHERE hash = ?")
	if err != nil {
		logger.Error(err)
		return err
	}
	defer psmt.Close()

	gid := common.ToHex(id)
	result, err := psmt.Exec(gid)
	if err != nil {
		logger.Error(err)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	logger.Debugf("deleted group: %s, rowsAffected %d", gid, rowsAffected)
	return nil
}

func CountGroups() uint64 {
	sql := "select count(*) as num FROM groupIndex "
	rows, err := mysqlDBLog.Query(sql)
	if err != nil {
		return 0
	}
	defer rows.Close()

	var result uint64
	for rows.Next() {
		err = rows.Scan(&result)
		if err != nil {
			logger.Errorf("scan groups failed, err: %v", err)
			return 0
		}

		return result
	}

	return 0
}

func SelectValidGroups(height uint64) []string {
	sql := "select hash FROM groupIndex where dismissheight> ? order by groupheight"
	rows, err := mysqlDBLog.Query(sql, height)
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

func SelectGroup(id []byte) (uint64, uint64, uint64) {
	sql := "select workheight, dismissheight, groupheight FROM groupIndex where hash= ?"
	rows, err := mysqlDBLog.Query(sql, common.ToHex(id))
	if err != nil {
		return 0, 0, 0
	}
	defer rows.Close()

	var workHeight, dismissHeight, groupHeight uint64
	for rows.Next() {
		err = rows.Scan(&workHeight, &dismissHeight, &groupHeight)
		if err != nil {
			logger.Errorf("select groups failed, err: %v", err)
			return 0, 0, 0
		}
	}

	if dismissHeight == math.MaxInt64 {
		dismissHeight = math.MaxUint64
	}
	return workHeight, dismissHeight, groupHeight
}

func InsertGroup(group *types.Group) error {
	id := common.ToHex(group.Id)
	workHeight := group.Header.WorkHeight
	dismissHeight := group.Header.DismissHeight
	if dismissHeight == math.MaxUint64 {
		dismissHeight = math.MaxInt64
	}
	groupHeight := group.GroupHeight

	logger.Infof("insert group: %s, workHeight: %d, dismissheight: %d, groupheight: %d", id, workHeight, dismissHeight, groupHeight)

	sqlStr := "replace INTO groupIndex(hash,workheight,dismissheight, groupheight) values (?, ?, ?, ?)"
	stmt, err := mysqlDBLog.Prepare(sqlStr)
	if err != nil {
		logger.Errorf("fail to prepare. group: %s, workHeight: %d, dismissheight: %d, groupheight: %d", id, workHeight, dismissHeight, groupHeight)
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(id, workHeight, dismissHeight, groupHeight)
	if err != nil {
		logger.Errorf("fail to insert group, err: %s. sql: %s", err, sqlStr)
		return err
	}

	logger.Infof("inserted group: %s, workHeight: %d, dismissheight: %d, groupheight: %d", id, workHeight, dismissHeight, groupHeight)
	return nil
}

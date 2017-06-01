package status

import (
	"database/sql"
	"fmt"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

/*******************************************
*函数名：Query
*作用：执行Query并填充相关结构统计
*作者:liziang061
*时间：2017/5/23 15:37
*******************************************/
func (this *Status_t) Query(query string, args ...interface{}) (row *sql.Rows, err error) {
	if this != nil {
		ts := time.Now()
		childStatusIndex := 0
		childStatusIndex = this.AddChildStatus()
		row, err = this.thisMysql.db.Query(query, args...)
		td := time.Since(ts)
		this.OriginMysqlReport(td, childStatusIndex, 0, query, args...)
		return
	}

	return nil, errors.New("Status is Null")
}

/*******************************************
*函数名：Exec
*作用：执行sql语句并填充相关结构统计
*作者:liziang061
*时间：2017/5/23 15:37
*******************************************/
func (this *Status_t) Exec(query string, args ...interface{}) (result sql.Result, err error) {
	if this != nil {
		ts := time.Now()
		childStatusIndex := 0
		childStatusIndex = this.AddChildStatus()
		result, err = this.thisMysql.db.Exec(query, args...)
		td := time.Since(ts)
		var rowsAff int64
		defer this.OriginMysqlReport(td, childStatusIndex, rowsAff, query, args...)
		rowsAff, _ = result.RowsAffected()
		if err != nil {
			return
		}
	}

	return nil, errors.New("Status is Null")
}

/*******************************************
*函数名：QueryRow
*作用：query单行并填充相关结构统计
*作者:liziang061
*时间：2017/5/23 15:37
*******************************************/
func (this *Status_t) QueryRow(query string, args ...interface{}) (row *sql.Row, err error) {
	if this != nil {
		ts := time.Now()
		childStatusIndex := 0
		if this != nil {
			childStatusIndex = this.AddChildStatus()
		}
		row = this.thisMysql.db.QueryRow(query, args...)
		td := time.Since(ts)
		this.OriginMysqlReport(td, childStatusIndex, 0, query, args...)
		return
	}

	return nil, errors.New("Status is Null")
}

/*******************************************
*函数名：OriginMysqlReport
*作用：原生mysql记录log以及对应的kafka消息上报处理
*作者: zengyupeng015
*时间：2017/5/15
*******************************************/
func (this *Status_t) OriginMysqlReport(costTime time.Duration, childStatusIndex int, RowsAffected int64, query string, args ...interface{}) {
	if this != nil {
		sql := strings.Replace(query, "?", "'%v'", -1)
		if len(args) > 0 {
			sql = fmt.Sprintf(sql, args...)
		}
		this.AddSqlMillis(costTime.Nanoseconds() / int64(time.Millisecond))
		this.ChildStatus[childStatusIndex].InParam[ParamSingle] = sql //填充入参sql语句
		this.ChildStatus[childStatusIndex].SetEndTimeWithNow()
		this.ChildStatus[childStatusIndex].SetDurMillis(costTime.Nanoseconds() / int64(time.Millisecond))
		this.ChildStatus[childStatusIndex].OutParam[OutParamRowsAffected] = RowsAffected //填充出参的sql修改行数
		this.ChildStatus[childStatusIndex].ServiceType = ServiceTypeSql
		_, filename, line, _ := runtime.Caller(1)
		this.ChildStatus[childStatusIndex].Uri = fmt.Sprintf("%s:%d", filepath.Base(filename), line)
	}
}

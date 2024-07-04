package sequence

import (
	"database/sql"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

/* 建立MySQL链接，执行REPLACE INTO 语句
REPLACE INTO sequence(stub) values('a')
SELECT LAST_INSERT_ID()
*/

const sqlReplaceIntoStub = `REPLACE INTO sequence(stub)VALUES('a')`

type MySQL struct {
	conn sqlx.SqlConn
}

func NewMySQL(dsn string) *MySQL {
	conn := sqlx.NewMysql(dsn)
	return &MySQL{
		conn: conn,
	}
}

/* Next方法 用于实现取号器取号的操作 */
func (m *MySQL) Next() (seq uint64, err error) {
	// prepare
	var stmt sqlx.StmtSession
	stmt, err = m.conn.Prepare(sqlReplaceIntoStub) //预编译
	if err != nil {
		logx.Errorw("conn.Prepare failed", logx.LogField{Key: "err", Value: err.Error()})
	}
	defer stmt.Close()
	// 执行
	var rest sql.Result
	rest, err = stmt.Exec()
	if err != nil {
		logx.Errorw("stmt.Exec() failed", logx.LogField{Key: "err", Value: err.Error()})
		return
	}
	// 获取插入ID
	var lid int64
	lid, err = rest.LastInsertId()
	if err != nil {
		logx.Errorw("rest.LastInsertId failed", logx.LogField{Key: "err", Value: err.Error()})
		return
	}
	return uint64(lid), nil
}

/*
数据库操作类
*/
package Utils

import (
	"encoding/json"
	"fmt"

	_ "github.com/bmizerany/pq"        //postgresql驱动
	_ "github.com/go-sql-driver/mysql" //mysql驱动
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" //sqlite3驱动
	//	"database/sql"
)

//数据库配置参数
type DBMethed struct {
	DBType   string //mysql/postgres/sqlite3
	Host     string
	Port     string
	Username string
	Password string
	Database string
	DBFile   string
	DBHandle *sqlx.DB
}

//连接数据库
func (this *DBMethed) ConnDataBase() error {
	this.DBHandle = nil
	//默认是mysql数据库
	dbInfo := "" + this.Username + ":" + this.Password + "@tcp(" + this.Host + ":" + this.Port + ")/" + this.Database + ""
	//postgres 数据库
	if this.DBType == "postgres" {
		dbInfo = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", this.Host, this.Port, this.Username, this.Password, this.Database)
	}
	//sqlite3数据库
	if this.DBType == "sqlite3" {
		dbInfo = this.DBFile
	}

	DBHandle, err := sqlx.Open(this.DBType, dbInfo)
	if err != nil {
		return err
	}

	DBHandle.SetMaxOpenConns(200)
	DBHandle.SetMaxIdleConns(20)
	this.DBHandle = DBHandle
	return nil
}

//是否正常连接数据
func (this *DBMethed) IsConnect() bool {
	if this.DBHandle != nil {
		return true
	}
	return false
}

//关闭数据库连接
func (this *DBMethed) Close() {
	if this.DBHandle != nil {
		this.DBHandle.Close()
		this.DBHandle = nil
	}
}

//查询并转换成Map
func (this *DBMethed) Query(sqlString string, args ...any) ([]map[string]interface{}, error) {
	tableData := make([]map[string]interface{}, 0)
	if this.DBHandle == nil {
		err := this.ConnDataBase()
		if err != nil {
			errinfo := make(map[string]interface{})
			errinfo["ERROR0"] = err.Error()
			tableData = append(tableData, errinfo)
			return tableData, err

		}
	}

	stmt, err := this.DBHandle.Prepare(sqlString)
	if err != nil {
		errinfo := make(map[string]interface{})
		errinfo["ERROR1"] = err.Error()
		tableData = append(tableData, errinfo)
		return tableData, err

	}
	defer stmt.Close()
	rows, err := stmt.Query(args...)
	if err != nil {
		errinfo := make(map[string]interface{})
		errinfo["ERROR2"] = err.Error()
		tableData = append(tableData, errinfo)
		return tableData, err

	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		errinfo := make(map[string]interface{})
		errinfo["ERROR3"] = err.Error()
		tableData = append(tableData, errinfo)
		return tableData, err
	}
	count := len(columns)
	values := make([]interface{}, count)
	valuePtrs := make([]interface{}, count)
	for rows.Next() {
		for i := 0; i < count; i++ {
			valuePtrs[i] = &values[i]
		}
		rows.Scan(valuePtrs...)
		entry := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			entry[col] = v
		}
		tableData = append(tableData, entry)
	}
	return tableData, nil
}

//查询结果转成Json字符串
func (this *DBMethed) Map2Json(tableData []map[string]interface{}, oneFlag bool) string {
	rst := ""
	if oneFlag && len(tableData) > 0 {
		jsonData, err := json.Marshal(tableData[0])
		if err != nil {
			return "{\"ERROR\":\"" + err.Error() + "\"}"
		}
		rst = string(jsonData)
	} else {
		jsonData, err := json.Marshal(tableData)
		if err != nil {
			return "{\"ERROR\":\"" + err.Error() + "\"}"
		}
		rst = string(jsonData)
	}
	return rst
}

//普通查询数据
func (this *DBMethed) Query2Json(sql string) string {
	rst, _ := this.Query(sql)
	return this.Map2Json(rst, false)

}

func (this *DBMethed) Query2JsonOne(sql string) string {
	rst, _ := this.Query(sql)
	return this.Map2Json(rst, true)
}

//根据id查询单条数据
func (this *DBMethed) QueryById(TableName, ObjectID string) string {
	sql := "select * from " + TableName + " where id = '" + ObjectID + "' "
	rst, _ := this.Query(sql)
	return this.Map2Json(rst, true)
}

/*
查询数据量,返回整数
 parameter:
  TableName: 表名
  WhereStr: 条件语句
 return:
  int: 条数
*/
func (this *DBMethed) GetCount(TableName, WhereStr string) int {
	sql := "select count(1) as COUNT from " + TableName + " where " + WhereStr
	rst, err := this.Query(sql)
	if err != nil {
		fmt.Print("GetCount error, ", err)
		return -1
	}
	return int(rst[0]["COUNT"].(int64))
}

//查询数据量,返回json字符串
func (this *DBMethed) QueryCount(TableName, WhereStr string) string {
	sql := "select count(1) as COUNT from " + TableName + " where " + WhereStr
	rst, _ := this.Query(sql)
	return this.Map2Json(rst, true)

}

/*
执行sql，包括insert,update,delete,create,alter,drop等
 parameter:
  sql: 需要执行的sql,支持预编译
  args: sql中包含的参数
 return:
  int: 修改的条数
  error: 错误
*/
func (this *DBMethed) Exec(sql string, args ...any) (int, error) {
	if this.DBHandle == nil {
		err := this.ConnDataBase()
		if err != nil {
			return -1, err
		}

	}

	stmt, err := this.DBHandle.Prepare(sql)
	if err != nil {
		return -1, err
	}
	rst, err := stmt.Exec(args...)
	if err != nil {
		return -1, err
	}
	num, err := rst.RowsAffected()
	if err != nil {
		return -1, err
	}
	return int(num), nil
}

//执行sql，包括insert,update,delete,create,alter,drop等
func (this *DBMethed) ExecSql(sql string, args ...any) string {
	if this.DBHandle == nil {
		err := this.ConnDataBase()
		if err != nil {
			return "{\"ERROR\":\"" + err.Error() + "\"}"
		}
	}

	stmt, err := this.DBHandle.Prepare(sql)
	if err != nil {
		return "{\"ERROR\":\"" + err.Error() + "\"}"
	}
	rst, err := stmt.Exec(args...)
	if err != nil {
		return "{\"ERROR\":\"" + err.Error() + "\"}"
	}
	num, err := rst.RowsAffected()
	if err != nil {
		return "{\"ERROR\":\"" + err.Error() + "\"}"
	}
	return "{\"ChageLine\":" + Itoa(num) + "}"
}

//查询表字段，返回字段数组
func (this *DBMethed) QueryFields(TableName string) string {

	if this.DBHandle == nil {
		err := this.ConnDataBase()
		if err != nil {
			return "{\"ERROR\":\"" + err.Error() + "\"}"
		}
	}

	sqlString := "select * from " + TableName + " where 1 = 2 "

	stmt, err := this.DBHandle.Prepare(sqlString)
	if err != nil {
		return "{\"ERROR\":\"" + err.Error() + "\"}"
	}
	defer stmt.Close()
	rows, err := stmt.Query()
	if err != nil {
		return "{\"ERROR\":\"" + err.Error() + "\"}"
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return "{\"ERROR\":\"" + err.Error() + "\"}"
	}

	var rst map[string][]string
	rst = make(map[string][]string)
	rst["Fields"] = columns

	fmt.Println(rst)
	jsonData, err := json.Marshal(rst)
	if err != nil {
		return "{\"ERROR\":\"" + err.Error() + "\"}"
	}
	return string(jsonData)

}

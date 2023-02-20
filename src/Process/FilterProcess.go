/*
处理filter条件串
*/
package Process

import (
	//		"fmt"
	"encoding/json"
	"strings"

	Utils "Homepage/Utils" //通用类

	seelog "github.com/cihub/seelog"
)

type FilterProcess struct {
	DBType string
	Order  string
	Limit  int
	Skip   int
	Sql    string
	Fields string
	Where  string
}

func (this *FilterProcess) init() {
	if this.DBType == "" {
		this.DBType = "mysql"
	}
	this.Fields = "*"
	this.Limit = 100
	this.Skip = 0
}

func (this *FilterProcess) SetDBType(dbtype string) {
	this.DBType = dbtype
}

//解析Filter语句
func (this *FilterProcess) LoadFilter(filterStr string) {
	this.init()
	if filterStr == "" {
		return
	}
	var filter map[string]interface{}
	if err := json.Unmarshal([]byte(filterStr), &filter); err != nil {
		seelog.Error("json.Unmarshal err: ", err)
		seelog.Flush()
		return
	}
	//fmt.Println(filter)
	for key, value := range filter {
		switch key {
		case "fields":
			fieldStr := ""
			switch value.(type) {
			//解析 fields:["id", "username", "realname", "nextApproval_id", "mobile"],
			case []interface{}:
				for _, field := range value.([]interface{}) {
					if fieldStr != "" {
						fieldStr += ","
					}
					fieldStr += field.(string)
				}
			//解析 fields:{"id": true,  "username": true, "realname": true, "nextApproval_id": true, "mobile": true },
			case map[string]interface{}:
				for k, v := range value.(map[string]interface{}) {
					if v.(bool) {
						if fieldStr != "" {
							fieldStr += ","
						}
						fieldStr += k
					}
				}
			}
			this.Fields = fieldStr
		case "skip":
			this.Skip = int(value.(float64))
		case "order":
			this.Order = value.(string)
		case "limit":
			this.Limit = int(value.(float64))
		case "sql":
			this.Sql = value.(string)
		case "where":
			this.Where = DistillWhereMap("", value.(map[string]interface{}), 1)
		}
	}

}

//根据filter内容，组合成sql语句
func (this *FilterProcess) ProduceQuerySql(TableName, filterStr string) string {

	this.LoadFilter(filterStr)

	if this.Sql != "" {
		return this.Sql
	} else {
		var sql = "select " + this.Fields + " from " + TableName
		if this.Where != "" {
			sql += " where " + this.Where
		}
		if this.Order != "" {
			sql += " order by " + this.Order
		}
		if this.DBType == "mysql" {
			sql += " limit " + Utils.Itoa(this.Skip) + "," + Utils.Itoa(this.Limit)
		} else if this.DBType == "postgres" || this.DBType == "sqlite3" {
			sql += " limit " + Utils.Itoa(this.Limit) + " OFFSET " + Utils.Itoa(this.Skip)
		}
		return sql
	}

}

//根据filter内容，生成Where语句
func (this *FilterProcess) GetWhereString(filterStr string) string {
	this.LoadFilter(filterStr)
	if this.Where != "" {
		return this.Where
	} else {
		return "1 = 1"
	}
}

//使用递归的方式，组成where语句
//lastKey 上一层的key值
//whereMap 待处理的where
//flag 1：and；2：or
func DistillWhereMap(lastKey string, whereMap map[string]interface{}, flag int) string {
	rst := ""
	for key, value := range whereMap {
		//fmt.Println(key , "--" , value)
		if rst != "" {
			if flag == 1 {
				rst += " and "
			} else {
				rst += " or "
			}
		}
		switch key {
		case "or", "OR":
			var tmp string
			for _, val := range value.([]interface{}) {
				if tmp != "" {
					tmp += " or "
				}
				tmp += DistillWhereMap("or", val.(map[string]interface{}), 2)
			}
			if tmp != "" {
				rst += "( " + tmp + " ) "
			}
		case "and", "AND":
			var tmp string
			for _, val := range value.([]interface{}) {
				if tmp != "" {
					tmp += " or "
				}
				tmp += DistillWhereMap("or", val.(map[string]interface{}), 2)
			}
			if tmp != "" {
				rst += "( " + tmp + " ) "
			}
		case "between":
			between := value.([]interface{})
			if len(between) > 1 {
				tmp := lastKey + " >= " + Utils.Itoa(between[0].(float64)) + " and " + lastKey + " <= " + Utils.Itoa(between[1].(float64))
				rst += "( " + tmp + " ) "
			}
		case "inq":
			var tmp string
			for _, val := range value.([]interface{}) {
				if tmp != "" {
					tmp += ", "
				}
				tmp += "'" + val.(string) + "'"
			}
			if tmp != "" {
				rst += " " + lastKey + " in ( " + tmp + " ) "
			}
		case "nin":
			var tmp string
			for _, val := range value.([]interface{}) {
				if tmp != "" {
					tmp += ", "
				}
				tmp += "'" + val.(string) + "'"
			}
			if tmp != "" {
				rst += " " + lastKey + " not in ( " + tmp + " ) "
			}
		default:
			switch value.(type) {
			case map[string]interface{}:
				rst += DistillWhereMap(key, value.(map[string]interface{}), 1)
			case string:
				if InStrArray([]string{"gt", "gte", "lt", "lte", "ne", "like", "nlike"}, key) {
					rst += " ( " + GetOperatorString(lastKey, key, value.(string)) + " )"
				} else {
					rst += " ( " + key + " = '" + value.(string) + "' )"
				}
			case float64:
				if InStrArray([]string{"gt", "gte", "lt", "lte", "ne", "like", "nlike"}, key) {
					rst += " ( " + GetOperatorString(lastKey, key, Utils.Itoa(value.(float64))) + " )"
				} else {
					rst += " ( " + key + " = '" + Utils.Itoa(value.(float64)) + "' )"
				}
			}
		}
	}
	return rst
}

//根据运算符组合判断语句
func GetOperatorString(key, operator, value string) string {
	var rst string
	if strings.EqualFold(operator, "gt") {
		rst = key + " > '" + value + "'"
	} else if strings.EqualFold(operator, "gte") {
		rst = key + " >= '" + value + "'"
	} else if strings.EqualFold(operator, "lt") {
		rst = key + " < '" + value + "'"
	} else if strings.EqualFold(operator, "lte") {
		rst = key + " <= '" + value + "'"
	} else if strings.EqualFold(operator, "ne") {
		rst = key + " <> '" + value + "'"
	} else if strings.EqualFold(operator, "like") {
		rst = key + " like '%" + value + "%'"
	} else if strings.EqualFold(operator, "nlike") {
		rst = key + " not like '%" + value + "%'"
	}
	return rst
}

//判断字符串数组中含字符串
func InStrArray(str_array []string, target string) bool {
	for _, element := range str_array {
		if target == element {
			return true
		}
	}
	return false
}

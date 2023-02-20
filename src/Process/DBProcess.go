/*
   数据处理类
*/
package Process

import (
	//	"fmt"
	"strings"
	"time"

	Utils "Homepage/Utils" //通用类

	seelog "github.com/cihub/seelog"
)

type DBProcess struct {
	nDBMethed *Utils.DBMethed
}

//设置数据库连接
func (this *DBProcess) SetDBMethed(nDBMethed *Utils.DBMethed) {
	this.nDBMethed = nDBMethed
}

//获取表的数据量，支持filter
func (this *DBProcess) DoGetTableCount(TableName, filter string) string {
	var filterObject FilterProcess
	return this.nDBMethed.QueryCount(TableName, filterObject.GetWhereString(filter))
}

//获取表的字段列表
func (this *DBProcess) DoTableFields(TableName string) string {
	return this.nDBMethed.QueryFields(TableName)
}

//通用查询
func (this *DBProcess) DoGetQuery(TableName, ObjectID, filter string) string {
	if ObjectID != "" {
		return this.nDBMethed.QueryById(TableName, ObjectID)
	}
	var filterObject FilterProcess
	filterObject.SetDBType(this.nDBMethed.DBType)
	sql := filterObject.ProduceQuerySql(TableName, filter)
	seelog.Debug("SQL:", sql)
	seelog.Flush()
	return this.nDBMethed.Query2Json(sql)
}

//建表
func (this *DBProcess) DoCreateTable(TableName string, PostForm map[string][]string) string {
	sql := ""
	for filed, vals := range PostForm {
		value := vals[0]
		sql += ", " + filed + " " + value + ""
	}
	if sql != "" {
		sql = "create table " + TableName + "(id varchar(30)" + sql + ")"
		return this.nDBMethed.ExecSql(sql)
	} else {
		return ""
	}

}

//删表
func (this *DBProcess) DoDropTable(TableName string, PostForm map[string][]string) string {
	sql := "drop table " + TableName
	return this.nDBMethed.ExecSql(sql)
}

//增加字段
func (this *DBProcess) DoAddField(TableName string, PostForm map[string][]string) string {
	sql := ""
	for filed, vals := range PostForm {
		value := vals[0]
		if sql == "" {
			sql += filed + " " + value + ""
		} else {
			sql += ", " + filed + " " + value + ""
		}
	}
	if sql != "" {
		sql = "alter table " + TableName + " add " + sql
		return this.nDBMethed.ExecSql(sql)
	} else {
		return ""
	}
}

//删除字段
func (this *DBProcess) DoDelField(TableName string, PostForm map[string][]string) string {
	sql := "alert table " + TableName + " drop "
	for filed, vals := range PostForm {
		value := vals[0]
		if value == "true" {
			runsql := sql + filed
			this.nDBMethed.ExecSql(runsql)
		}
	}
	return `{"DeleteField":true}`
}

/*
用户登录验证
  username, password: 用户名和密码
  return :
    string : Json字符串
*/
func (this *DBProcess) DoUserLogin(username, password, RemoteAddr string) string {
	rst := make(map[string]interface{})
	if this.nDBMethed.GetCount("User", " User ='"+username+"' and Passwd = '"+password+"'") > 0 {
		rst["code"] = 0
		data := make(map[string]interface{})
		//login := this.nDBMethed.Query("select * from User where User ='" + username + "' and Passwd = '" + password + "'")[0]
		loginList, _ := this.nDBMethed.Query("select * from User where User = ? and Passwd = ? ", username, password)
		login := loginList[0]
		//将查出来的主机信息换成本次登录的RemoteAddr
		login["Host"] = RemoteAddr
		delete(login, "Passwd")
		var JwtUtil Utils.JwtUtil
		token, err := JwtUtil.EncodeJwtString(login)
		if err != nil {
			rst["code"] = 1
			rst["data"] = "获取token错误"
		} else {
			seelog.Debug("token: ", token)
			seelog.Flush()
			//登录成功
			login["Token"] = token
			data["login"] = login

			config := make(map[string]interface{})
			config["Encrypt"] = "off"
			config["Licence"] = "CC95C724DF1187462919EE9E70B058C85E5D1615B7AECCB7"
			data["config"] = config

			rst["data"] = data

			//在Auth表中新增一条数据
			token_lit := token[len(token)-40:]
			cnt, err := this.nDBMethed.Exec("insert into Auth(Token,TokenAll,Host,User) values(?,?,?,?)", token_lit, token, RemoteAddr, username)
			if err != nil {
				seelog.Error("Insert into Auth Error: ", err.Error())
				seelog.Flush()
			} else {
				seelog.Info("Insert into Auth Count: ", cnt)
				seelog.Flush()

			}
		}
	} else {
		rst["code"] = 1
		rst["data"] = "用户名密码错误"
	}

	return Utils.Any2Json(rst)

}

/*
用户退出操作
  parameter:
    token: 会话token
  return:
    string: 结果Json字符串
*/
func (this *DBProcess) DoUserLogout(token string) string {
	rst := make(map[string]interface{})
	rst["code"] = 0
	cnt, err := this.nDBMethed.Exec("delete from Auth where TokenAll = ?", token)
	if err != nil {
		rst["code"] = 1
		rst["data"] = err.Error()
	}
	rst["data"] = "Delete Count:" + Utils.Itoa(cnt)

	return Utils.Any2Json(rst)
}

/*
检查Token
  parameter:
    token: 会话token
  return:
    string: 结果Json字符串
*/
func (this *DBProcess) DoCheckToken(token string) string {
	rst := make(map[string]interface{})
	rst["code"] = 0
	var jwt Utils.JwtUtil
	nInfo, err := jwt.DecodeJwtString(token)
	if err != nil {
		rst["code"] = 1
		rst["data"] = err.Error()
	}
	rst["data"] = nInfo
	return Utils.Any2Json(rst)
}

//对表数据的增删改
func (this *DBProcess) DoTablePost(TableName, ObjectID string, PostForm map[string][]string) string {
	sql := ""
	if ObjectID == "" { //insert数据
		ObjectID = Utils.CreateRandomString(24)
		fileds := "id"
		//values := "'" + Utils.RunShellCmd("/usr/bin/uuidgen | sed 's/-//g' | cut -b 1-24") + "'"
		values := "'" + ObjectID + "'"

		for filed, vals := range PostForm {
			value := vals[0]
			fileds += ", " + filed
			values += ", '" + value + "'"
		}
		sql = "insert into " + TableName + "( " + fileds + " ) values( " + values + " )"
	} else {
		_method, ok := PostForm["_method"]
		if ok {
			if _method[0] == "PUT" { //修改数据
				updatestr := ""
				for filed, vals := range PostForm {
					value := vals[0]
					if filed == "_method" {
						continue
					}
					if updatestr == "" {
						updatestr = filed + "='" + value + "'"
					} else {
						updatestr += ", " + filed + "='" + value + "'"
					}
				}
				sql = "update " + TableName + " set " + updatestr + " where id ='" + ObjectID + "'"

			} else if _method[0] == "DELETE" { //删除数据
				sql = "delete from " + TableName + " where id = '" + ObjectID + "'"
			}

		}
	}
	if sql != "" {
		//return this.nDBMethed.ExecSql(sql)
		_, err := this.nDBMethed.Exec(sql)
		if err != nil {
			return "{\"ERROR\":\"" + err.Error() + "\"}"
		}
		return this.nDBMethed.QueryById(TableName, ObjectID)
	} else {
		return ""
	}
}

//文件删除
func (this *DBProcess) DoDeleteFile(tablename, objectid string) string {
	queryRst, _ := this.nDBMethed.Query("select filename from " + tablename + " where id = '" + objectid + "'")
	if len(queryRst) > 0 {
		if _, ok := queryRst[0]["ERROR"]; ok {
			return this.nDBMethed.Map2Json(queryRst, true)
		}
		rst := this.nDBMethed.ExecSql("delete from " + tablename + " where id =  '" + objectid + "'")
		if strings.Count(rst, "ERROR") > 0 {
			return rst
		} else {
			filename := queryRst[0]["filename"].(string)
			seelog.Debug("Delete " + filename + " ...")
			seelog.Flush()
			flag := Utils.DeleteFile(filename)
			if flag {
				return `{"DELETE":"true"}`
			} else {
				return `{"DELETE":"false"}`
			}

		}

	} else {
		return `{"DELETE":"NO RECODE"}`
	}
}

//文件上传成功后操作数据库
func (this *DBProcess) DoUploadFile(tablename, filename, filepath, url string) string {
	queryRst, _ := this.nDBMethed.Query("select count(1) COUNT from " + tablename + " where name = '" + filename + "'")
	if _, ok := queryRst[0]["ERROR"]; ok {
		if strings.Count(queryRst[0]["ERROR"].(string), "doesn't exist") > 0 {
			//sql := `CREATE TABLE `+tablename+` (  id varchar(30) DEFAULT NULL, name varchar(100) DEFAULT NULL, filename varchar(200) DEFAULT NULL, url varchar(300) DEFAULT NULL, lastModifiedDate datetime DEFAULT NULL, downcount int(10) DEFAULT '0' ) `
			sql := `CREATE TABLE ` + tablename + ` (  id varchar(30) DEFAULT NULL, name varchar(100) DEFAULT NULL, filename varchar(200) DEFAULT NULL, url varchar(300) DEFAULT NULL, lastModifiedDate datetime DEFAULT NULL) `
			this.nDBMethed.ExecSql(sql)
		} else {
			return this.nDBMethed.Map2Json(queryRst, true)
		}
	}
	count := 0
	if cnt, ok := queryRst[0]["COUNT"]; ok {
		count = int(cnt.(int64))
	}

	nowStr := time.Now().Format("2006-01-02 15:04:05")
	if count == 0 {
		id := Utils.CreateRandomString(24)
		this.nDBMethed.ExecSql("insert into " + tablename + "(id, name, filename, url, lastModifiedDate) values ('" + id + "', '" + filename + "','" + filepath + "','" + url + "', '" + nowStr + "' ) ")
	} else {
		this.nDBMethed.ExecSql("update " + tablename + " set lastModifiedDate = '" + nowStr + "' where name = '" + filename + "'")
	}
	return this.nDBMethed.Query2Json("select * from " + tablename + " where  name = '" + filename + "'")
}

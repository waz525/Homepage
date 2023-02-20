/*
   自定义API处理
*/
package Process

import (
	Utils "Homepage/Utils" //通用类
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	seelog "github.com/cihub/seelog"
)

type FqlProcess struct {
	nDBMethed  *Utils.DBMethed            //配置数据库连接
	isInitConn bool                       //是否初始化
	nConnList  map[string]*Utils.DBMethed //应用数据库连接MAP
}

//设置配置数据库连接
func (this *FqlProcess) SetDBMethed(nDBMethed *Utils.DBMethed) {
	this.nDBMethed = nDBMethed
}

/*
创建数据库连接，如果不在队列里则新建；如果在队列里不能正常连接的，重新连接
 parameter:

 return:

*/
func (this *FqlProcess) ProduceConnList() {
	//如果未初始化，则make一下map
	if !this.isInitConn {
		this.nConnList = make(map[string]*Utils.DBMethed)
		this.isInitConn = true
	}

	//根据数据库配置的连接串创建nConnList
	if this.nDBMethed.GetCount("Conn", "1=1") > 0 {
		ConnQuery, _ := this.nDBMethed.Query("select * from Conn")
		for _, conn := range ConnQuery {

			Dbid := conn["id"].(string)
			DBType := conn["DbType"].(string)
			//将DbConn解析成Map
			DbConn := Utils.Str2Map(conn["DbConn"].(string), ";", "=")

			//判断是否已经创建DBMethed对象
			if _, ok := this.nConnList[Dbid]; !ok {
				dbm := Utils.DBMethed{}

				dbm.DBType = DBType
				if dbm.DBType == "sqlite3" {
					dbm.DBFile = DbConn["DBFile"]
				} else {
					dbm.Host = DbConn["Host"]
					dbm.Port = DbConn["Port"]
					dbm.Username = DbConn["Username"]
					dbm.Password = DbConn["Password"]
					dbm.Database = DbConn["Database"]
				}

				this.nConnList[Dbid] = &dbm

			} else {
				//如果连接信息被修改，则关闭连接后，加载新的配置
				if this.nConnList[Dbid].DBType != DBType || this.nConnList[Dbid].Host != DbConn["Host"] || this.nConnList[Dbid].Port != DbConn["Port"] || this.nConnList[Dbid].Username != DbConn["Username"] || this.nConnList[Dbid].Password != DbConn["Password"] || this.nConnList[Dbid].Database != DbConn["Database"] {
					this.nConnList[Dbid].Close()

					if this.nConnList[Dbid].DBType == "sqlite3" {
						this.nConnList[Dbid].DBFile = DbConn["DBFile"]
					} else {
						this.nConnList[Dbid].Host = DbConn["Host"]
						this.nConnList[Dbid].Port = DbConn["Port"]
						this.nConnList[Dbid].Username = DbConn["Username"]
						this.nConnList[Dbid].Password = DbConn["Password"]
						this.nConnList[Dbid].Database = DbConn["Database"]
					}

				}

			}

			//新建的或连接不上的，重新连接一次
			if !this.nConnList[Dbid].IsConnect() {
				err := this.nConnList[Dbid].ConnDataBase()
				if err != nil {
					seelog.Error(err.Error())
					seelog.Flush()
				}
			}
		}

	}
}

/*
删除数据库连接，先判断连接是否在用，再删除
 parameter:
	dbid 数据库id
 return:
	string 处理结果字符串
*/
func (this *FqlProcess) DeleteConn(dbid string) string {
	if this.nDBMethed.GetCount("Api", "DbId = '"+dbid+"' ") > 0 {
		return `{"ERROR":"This Conn is in used , can't to delete !"}`
	}

	//如果存在链接，则关闭连接，并从nConnList中删除
	if _, ok := this.nConnList[dbid]; ok {
		this.nConnList[dbid].Close()
		delete(this.nConnList, dbid)
	}

	//删除表里记录
	if _, err := this.nDBMethed.Exec("delete from Conn where id = '" + dbid + "' "); err != nil {
		return "{\"ERROR\":\"" + err.Error() + "\"}"
	}

	return `{"code":"0"}`
}

/*
处理API请求，返回json字符串
 parameter:
	urlPath   请求URL
	reqMethod 请求方法post/get
	tokenStat token状态
	postForm  post的内容
 return:
	string 处理结果字符串
	{
		"code":0,  //0:正常  1:异常
		"msg":"Run Successfully!",
		"data":""
	}
*/
func (this *FqlProcess) DealWithApi(urlPath string, reqMethod string, tokenStat bool, formData map[string]interface{}) string {

	seelog.Debug("DealWith urlPath: ", urlPath, " , reqMethod: ", reqMethod, " , tokenStat: ", tokenStat, " , formData: ", formData, " ...")
	seelog.Flush()
	startTime := time.Now()

	rst := make(map[string]any)
	rst["code"] = 0 //0:正常  1:异常
	rst["msg"] = "Run Successfully!"
	rst["data"] = ""

	//查询URL是否已定义
	apiConfigList, _ := this.nDBMethed.Query("select * from Api where Url = ?", urlPath)
	//if this.nDBMethed.GetCount("Api", "Url = '"+urlPath+"' ") == 0 {
	if len(apiConfigList) == 0 {
		rst["code"] = 1
		rst["msg"] = fmt.Sprintf("Url[%s] is not define !", urlPath)
		rst["_t"] = time.Since(startTime).Seconds()
		return Utils.Any2Json(rst)
	}

	//取结果集中第一条记录
	apiConfig := apiConfigList[0]

	//判断是否需要token，以及token是否正常
	if apiConfig["Auth"].(int64) != 0 && !tokenStat {
		rst["code"] = 1
		rst["msg"] = fmt.Sprintf("Token is needed , but token is invalid !")
		rst["_t"] = time.Since(startTime).Seconds()
		return Utils.Any2Json(rst)
	}

	//判断是否需要token，以及token是否正常
	if !strings.EqualFold(apiConfig["Func"].(string), reqMethod) {
		rst["code"] = 1
		rst["msg"] = fmt.Sprintf("Http Method Error ! %s is needed , but this request is %s !", apiConfig["Func"].(string), reqMethod)
		rst["_t"] = time.Since(startTime).Seconds()
		return Utils.Any2Json(rst)
	}

	//命令字符串，判断是否为空
	cmd := apiConfig["Cmd"].(string)
	if len(cmd) == 0 {
		rst["code"] = 1
		rst["msg"] = fmt.Sprintf("Cmd is empty !")
		rst["_t"] = time.Since(startTime).Seconds()
		return Utils.Any2Json(rst)
	}

	mode := apiConfig["Mode"].(string)
	//不是处理系统命令
	if !strings.EqualFold(mode, "system") {
		dbid := apiConfig["DbId"].(string)
		//判断数据库连接实例是否在列表里，没有则重新连接
		if _, ok := this.nConnList[dbid]; !ok {
			this.ProduceConnList()
		}

		//如果还是没有，则报错退出
		mDBMethed, ok := this.nConnList[dbid]
		if !ok {
			rst["code"] = 1
			rst["msg"] = fmt.Sprintf("Database Connect is not define !")
			rst["_t"] = time.Since(startTime).Seconds()
			return Utils.Any2Json(rst)
		}

		seelog.Debug("Dbid:", dbid, " is ", mDBMethed.IsConnect())
		seelog.Flush()

		//判断数据库实例是否连接
		if !mDBMethed.IsConnect() {
			queryRst, _ := this.nDBMethed.Query("select DbName from Conn where id = ?", dbid)
			DbName := queryRst[0]["DbName"].(string)
			rst["code"] = 1
			rst["msg"] = fmt.Sprintf("Database[%s] Connect is not connected !", DbName)
			rst["_t"] = time.Since(startTime).Seconds()
			return Utils.Any2Json(rst)
		}
		if strings.EqualFold(mode, "query") {
			queryRst, err := this.dealWithApiQuery(mDBMethed, cmd, formData)
			if err != nil {
				rst["code"] = 1
				rst["msg"] = fmt.Sprintf("mode[%s] have error, %s", mode, err.Error())
				rst["_t"] = time.Since(startTime).Seconds()
				return Utils.Any2Json(rst)
			}

			rst["data"] = queryRst
			rst["_t"] = time.Since(startTime).Seconds()
			return Utils.Any2Json(rst)

		} else if strings.EqualFold(mode, "commit") {
			execRst, err := this.dealWithApiCommit(mDBMethed, cmd, formData)
			if err != nil {
				rst["code"] = 1
				rst["msg"] = fmt.Sprintf("mode[%s] have error, %s", mode, err.Error())
				rst["_t"] = time.Since(startTime).Seconds()
				return Utils.Any2Json(rst)
			}
			rst["msg"] = fmt.Sprintf("This sql update %d lines", execRst)
			rst["data"] = execRst
			rst["_t"] = time.Since(startTime).Seconds()
			return Utils.Any2Json(rst)

		} else {
			rst["code"] = 1
			rst["msg"] = fmt.Sprintf("mode[%s] is not supported !", mode)
			rst["_t"] = time.Since(startTime).Seconds()
			return Utils.Any2Json(rst)
		}

	} else {
		//执行系统命令
		//cmdList := strings.Split(cmd, " ")
		// cmdfile := cmdList[0]
		//判断命令文件是否存在
		// if !Utils.IsFileExist(cmdfile) {
		// 	rst["code"] = 1
		// 	rst["msg"] = fmt.Sprintf("System cmd file[%s] not exist !", cmdfile)
		// 	return Utils.Any2Json(rst)
		// }
		//运行系统命令时加上超时时间，默认超时20秒

		cmdResult := Utils.RunShellCmd(cmd)
		rst["data"] = cmdResult

	}

	rst["_t"] = time.Since(startTime).Seconds()
	return Utils.Any2Json(rst)
}

/*
查询数据(小写字母开头的函数只能内部调用)
 parameter:
	mDBMethed  数据库连接
	mSql       配置的查询sql
	formData   前端提交的数据
 return:
	map[string]any 查询结果
	error          错误
*/
func (this *FqlProcess) dealWithApiQuery(mDBMethed *Utils.DBMethed, mSql string, formData map[string]any) ([]map[string]any, error) {

	var queryArgs []any

	//判断formData中是否有"param"段
	paramMap, isHaveParam := formData["param"]
	if isHaveParam {
		//组合查询参数列表
		for key, val := range paramMap.(map[string]any) {
			queryArgs = append(queryArgs, sql.Named(key, val))
		}
	}

	//处理原始sql里的回车等
	mSql = strings.ReplaceAll(mSql, "\r\n", " ")
	mSql = strings.ReplaceAll(mSql, "\n", " ")
	mSql = strings.ReplaceAll(mSql, "\r", " ")

	//处理动态sql，[] 里是否有效，[]内所有变量都提交了才有效
	rstSql := ""
	if strings.ContainsAny(mSql, "[") {
		if !isHaveParam {
			return nil, errors.New("request must add param ! ")
		}
		//是否被单引号引起来了,出现在字符串里的不算
		strFlag := false
		//[]子串的开始索引
		startIndex := 0
		//[]子串的结束索引
		endIndex := 0
		for i := 0; i < len(mSql); i++ {
			if mSql[i] == '\'' {
				strFlag = !strFlag
			}
			if !strFlag && mSql[i] == '[' {
				startIndex = i + 1
				rstSql = rstSql + mSql[endIndex:i]
			}
			if !strFlag && mSql[i] == ']' {
				endIndex = i
				//标识本子句是否有效
				vaild := true
				//获取变量，并判断是否在请求的param里；全部都有才有效
				rKeys := Utils.RegexpString(mSql[startIndex:endIndex], "[@:]+[a-zA-Z0-9_-]+")
				if rKeys != nil {
					for _, rKey := range rKeys {
						if _, isInParam := (paramMap.(map[string]any))[rKey[1:]]; isInParam == false {
							vaild = false
						}
					}
				}

				//如果有效，则把子串加到rstSql中去
				if vaild {
					rstSql = rstSql + mSql[startIndex:endIndex]
				}
				//本次子串判断结束，将startIndex置为0
				startIndex = 0
				//endIndex 作为后续的开头
				endIndex++
			}
		}
		//如果startIndex不为0，则是没找对应的]符号
		if startIndex != 0 {
			return nil, errors.New("[ ] is not match! ")
		}
		//将后续字符串加入到rstSql中去
		if endIndex < len(mSql) {
			rstSql = rstSql + mSql[endIndex:]
		}
		mSql = rstSql
	}

	//判断formData中是否有"limit"段
	limitMap, isHaveLimit := formData["limit"]
	if isHaveLimit {
		var page, size float64
		page = 1
		if pageVal, valid := limitMap.(map[string]any)["page"]; valid {
			page = pageVal.(float64)
		}
		size = 10
		if sizeVal, valid := limitMap.(map[string]any)["size"]; valid {
			size = sizeVal.(float64)
		}

		//计算skip的值
		skip := (page - 1) * size
		//根据不同数据库组合不同的limit
		if mDBMethed.DBType == "mysql" {
			mSql = "select * from ( " + mSql + " ) tfql  limit " + Utils.Itoa(skip) + "," + Utils.Itoa(size)
		} else if mDBMethed.DBType == "postgres" || mDBMethed.DBType == "sqlite3" {
			mSql = "select * from ( " + mSql + " ) tfql limit " + Utils.Itoa(size) + " OFFSET " + Utils.Itoa(skip)
		}

	}

	//mysql不支持Named Parameters，处理成？号
	if mDBMethed.DBType == "mysql" {

		//获取变量
		rKeys := Utils.RegexpString(mSql, "[@:]+[a-zA-Z0-9_-]+")
		if rKeys != nil {
			var queryArgsList []any
			for _, rKey := range rKeys {
				if rValue, isInParam := (paramMap.(map[string]any))[rKey[1:]]; isInParam {
					queryArgsList = append(queryArgsList, rValue)
				}
				mSql = strings.ReplaceAll(mSql, rKey, "?")
			}
			if len(queryArgsList) > 0 {
				queryArgs = queryArgsList
			}
		}
	}

	seelog.Debug("DBType: ", mDBMethed.DBType)
	seelog.Debug("mSql: ", mSql)
	seelog.Debug("queryArgs:", queryArgs)
	seelog.Flush()

	return mDBMethed.Query(mSql, queryArgs...)
}

/*
执行sql(小写字母开头的函数只能内部调用)
 parameter:
	mDBMethed  数据库连接
	mCmd       配置的查询sql
	formData   前端提交的数据
 return:
	map[string]any 查询结果
	error          错误
*/
func (this *FqlProcess) dealWithApiCommit(mDBMethed *Utils.DBMethed, mCmd string, formData map[string]any) (int, error) {

	var queryArgs []any
	//组合查询参数列表
	param, ok := formData["param"]
	fmt.Println(param)
	if ok {
		switch param.(type) {
		case map[string]any: //单条参数更新
			for key, val := range param.(map[string]any) {
				queryArgs = append(queryArgs, sql.Named(key, val))
			}
			return mDBMethed.Exec(mCmd, queryArgs...)
		case []any: //支持参数数组，批量更新
			allCount := 0
			for _, nPara := range param.([]any) {
				var queryArgs []any
				for key, val := range nPara.(map[string]any) {
					queryArgs = append(queryArgs, sql.Named(key, val))
				}
				n, err := mDBMethed.Exec(mCmd, queryArgs...)
				allCount += n
				if err != nil {
					return allCount, err
				}
			}
			return allCount, nil
		default: //未知类型，直接报错
			return 0, errors.New("unknow param type")
		}

	}
	return mDBMethed.Exec(mCmd, queryArgs...)
}

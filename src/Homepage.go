/*
Name:		McmAPI.go
Create:		20210512
Modify:		20210512
Version:	1.0.0
Auth:		Worden
*/
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	//	"regexp"
	"io/ioutil"
	"net/http"
	"net/http/cgi"
	filepath "path"

	seelog "github.com/cihub/seelog"

	//	"encoding/json"
	BinInfo "Homepage/BinInfo"
	Process "Homepage/Process"

	Utils "Homepage/Utils" //通用类
)

//主配置对象
var nServerConf Utils.ServerConf

//配置数据库DBMethed对象
var nDBMethed Utils.DBMethed

//配置接口处理对象
var nDBProcess Process.DBProcess

//Jwt对象
var nJwtUtil Utils.JwtUtil

//自定义API处理
var nFqlProcess Process.FqlProcess

//主站接口
func HttpSiteInterface(rw http.ResponseWriter, req *http.Request) {
	url_path := req.URL.Path
	//seelog.Info("Receive " + req.Method + " from " + req.RemoteAddr + " ,URL.Path: " + url_path)
	//seelog.Flush()

	filePath := Utils.GetRealPath(nServerConf.FileStaticPath + "/" + url_path[len(nServerConf.FileStaticPrefix)-1:])

	//默认路径加上index.html
	if filePath[len(filePath)-1] == '/' {
		filePath = filePath + "index.html"
	}

	if Utils.IsFileExist(filePath) {
		rw.Header().Set("Content-Type", Utils.GetContentTypeByFilePath(filePath))
		fmt.Fprintln(rw, Utils.GetFileContent(filePath))
	} else {
		seelog.Error("File Path ( " + filePath + " ) not Exist  , http.NotFound !")
		seelog.Flush()
		//返回404
		http.NotFound(rw, req)
	}
}

//处理cgi-bin
func CgibinInterface(rw http.ResponseWriter, req *http.Request) {
	url_path := req.URL.Path
	seelog.Info("Receive " + req.Method + " from " + req.RemoteAddr + " ,URL.Path: " + url_path)
	seelog.Flush()
	handler := new(cgi.Handler)
	handler.Path = Utils.GetRealPath(nServerConf.CigBinPath) + "/" + url_path[len(nServerConf.CigBinPrefix)-1:]
	//fmt.Println("handler.Path: "+handler.Path)
	handler.Dir = Utils.GetRealPath(nServerConf.CigBinPath)
	handler.ServeHTTP(rw, req)
}

//处理登录验证
func McmUserLoginInterface(rw http.ResponseWriter, req *http.Request) {
	url_path := req.URL.Path
	seelog.Info("Receive " + req.Method + " from " + req.RemoteAddr + " ,URL.Path: " + url_path)
	seelog.Flush()
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	if strings.EqualFold(req.Method, "post") {
		if err := req.ParseForm(); err != nil {
			fmt.Fprintf(rw, "{\"ERROR\":\"ParseForm() err: %v\"}", err)
			return
		}
		postForm := req.PostForm
		seelog.Debug("req.PostForm: ", postForm)
		seelog.Flush()
		rst := nDBProcess.DoUserLogin(postForm["username"][0], postForm["password"][0], req.RemoteAddr)
		seelog.Debug("UserLogin rst: " + rst)
		seelog.Flush()
		fmt.Fprintln(rw, rst)
	} else {
		http.NotFound(rw, req)
	}
}

//处理退出
func McmUserLogoutInterface(rw http.ResponseWriter, req *http.Request) {
	url_path := req.URL.Path
	seelog.Info("Receive " + req.Method + " from " + req.RemoteAddr + " ,URL.Path: " + url_path)
	seelog.Flush()
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	if strings.EqualFold(req.Method, "post") {
		if err := req.ParseForm(); err != nil {
			fmt.Fprintf(rw, "{\"ERROR\":\"ParseForm() err: %v\"}", err)
			return
		}
		postForm := req.PostForm
		seelog.Debug("req.PostForm: ", postForm)
		seelog.Flush()
		rst := nDBProcess.DoUserLogout(postForm["token"][0])
		seelog.Debug("UserLogout rst: " + rst)
		seelog.Flush()
		fmt.Fprintln(rw, rst)
	} else {
		http.NotFound(rw, req)
	}
}

//检查token
func McmUserChktokenInterface(rw http.ResponseWriter, req *http.Request) {
	url_path := req.URL.Path
	seelog.Info("Receive " + req.Method + " from " + req.RemoteAddr + " ,URL.Path: " + url_path)
	seelog.Flush()
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	if strings.EqualFold(req.Method, "post") {
		if err := req.ParseForm(); err != nil {
			fmt.Fprintf(rw, "{\"ERROR\":\"ParseForm() err: %v\"}", err)
			return
		}
		postForm := req.PostForm
		seelog.Debug("req.PostForm: ", postForm)
		seelog.Flush()
		rst := nDBProcess.DoCheckToken(postForm["token"][0])
		seelog.Debug("Chktoken rst: " + rst)
		seelog.Flush()
		fmt.Fprintln(rw, rst)
	} else {
		http.NotFound(rw, req)
	}
}

//处理McmApi
func McmApiInterface(rw http.ResponseWriter, req *http.Request) {
	url_path := req.URL.Path
	seelog.Info("Receive " + req.Method + " from " + req.RemoteAddr + " ,URL.Path: " + url_path)
	seelog.Flush()
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	// seelog.Debug("req.Header: ", req.Header)
	// seelog.Debug("req.Header[token]: ", req.Header.Get("token"))
	// seelog.Flush()

	token := req.Header.Get("token")
	if token == "" {
		fmt.Fprintf(rw, "{\"ERROR\":\"Token is inValid, Token is empty \"}")
		return
	}
	if _, err := nJwtUtil.DecodeJwtString(token); err != nil {
		fmt.Fprintf(rw, "{\"ERROR\":\"Token is inValid, err: %v \"}", err)
		return
	}

	url_split := strings.Split(url_path, "/")
	var TableName, ObjectID string
	rst := "[]"
	//获取TableName
	if len(url_split) > 3 {
		TableName = url_split[3]
		seelog.Debug("TableName: " + TableName)
		seelog.Flush()
	}
	//获取ObjectID
	if len(url_split) > 4 {
		ObjectID = url_split[4]
		seelog.Debug("ObjectID: " + ObjectID)
		seelog.Flush()
	}
	if strings.EqualFold(req.Method, "get") {
		//获取filter
		filter := req.URL.Query().Get("filter")
		seelog.Debug("filter: " + filter)
		seelog.Flush()

		if strings.EqualFold(ObjectID, "count") { //查询数据量
			rst = nDBProcess.DoGetTableCount(TableName, filter)
		} else if strings.EqualFold(ObjectID, "TableFields") { //查询表结构
			rst = nDBProcess.DoTableFields(TableName)
		} else { //查询数据
			rst = nDBProcess.DoGetQuery(TableName, ObjectID, filter)
		}
	} else if strings.EqualFold(req.Method, "post") {
		if err := req.ParseForm(); err != nil {
			fmt.Fprintf(rw, "{\"ERROR\":\"ParseForm() err: %v\"}", err)
			return
		}
		postForm := req.PostForm
		seelog.Debug("req.PostForm: ", postForm)
		seelog.Flush()
		if strings.EqualFold(ObjectID, "CreateTable") { //查询表
			rst = nDBProcess.DoCreateTable(TableName, postForm)
		} else if strings.EqualFold(ObjectID, "DropTable") { //删除表
			rst = nDBProcess.DoDropTable(TableName, postForm)
		} else if strings.EqualFold(ObjectID, "AddField") { //增加字段
			rst = nDBProcess.DoAddField(TableName, postForm)
		} else if strings.EqualFold(ObjectID, "DelField") { //删除字段
			rst = nDBProcess.DoDelField(TableName, postForm)
		} else { //对表数据的增删改
			rst = nDBProcess.DoTablePost(TableName, ObjectID, postForm)
		}

	}
	seelog.Debug("API Result: " + rst)
	seelog.Flush()
	fmt.Fprintln(rw, rst)
}

//文件删除服务接口
func McmFileDeleteInterface(rw http.ResponseWriter, req *http.Request) {
	url_path := req.URL.Path
	seelog.Info("Receive " + req.Method + " from " + req.RemoteAddr + " ,URL.Path: " + url_path)
	seelog.Flush()
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	url_split := strings.Split(url_path, "/")
	rst := ""

	//获取ObjectID
	if len(url_split) > 4 {
		ObjectID := url_split[4]
		seelog.Debug("ObjectID: " + ObjectID)
		seelog.Flush()
		rst = nDBProcess.DoDeleteFile("file", ObjectID)
	} else {
		rst = `{"ERROR":"INVALID FILE ID"}`
	}

	fmt.Fprintln(rw, rst)
}

//文件上传服务接口
func McmFileUploadInterface(rw http.ResponseWriter, req *http.Request) {

	url_path := req.URL.Path
	seelog.Info("Receive " + req.Method + " from " + req.RemoteAddr + " ,URL.Path: " + url_path)
	seelog.Flush()

	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")

	var maxUploadSize int64
	maxUploadSize = 200 * 1024 * 1024 // 200 MB
	uploadPath := nServerConf.FileStaticPath + "UploadFile/"
	req.Body = http.MaxBytesReader(rw, req.Body, maxUploadSize)
	if err := req.ParseMultipartForm(maxUploadSize); err != nil {
		fmt.Fprintln(rw, `{"ERROR":"FILE TOO BIG, UP TO 200M"}`)
		return
	}
	file, handler, err := req.FormFile("file_upload")
	if err != nil {
		fmt.Fprintln(rw, `{"ERROR":"INVALID FILE"}`)
		return
	}
	defer file.Close()
	//获取文件名
	fileName := handler.Filename
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Fprintln(rw, `{"ERROR":"INVALID FILE"}`)
		return
	}
	/*
		filetype := http.DetectContentType(fileBytes)

		//根据文件类型获取文件后缀
		fileEndings, err := mime.ExtensionsByType(filetype)
		if err != nil {
			renderError(rw, "CANT_READ_FILE_TYPE", http.StatusInternalServerError)
			return
		}
		seelog.Debug("Upload FileType:",filetype,", fileEndings:",fileEndings) ; seelog.Flush() ;
	*/
	newPath := Utils.GetRealPath(filepath.Join(uploadPath, fileName))
	newFile, err := os.Create(newPath)
	if err != nil {
		fmt.Fprintln(rw, `{"ERROR":"CANT WRITE FILE"}`)
		return
	}
	defer newFile.Close()
	if _, err := newFile.Write(fileBytes); err != nil {
		fmt.Fprintln(rw, `{"ERROR":"CANT WRITE FILE"}`)
		return
	}
	seelog.Info("Upload File: ", newPath, " success !")
	seelog.Flush()
	url := "http://" + req.Host + nServerConf.FileStaticPrefix + "UploadFile/" + fileName
	rst := nDBProcess.DoUploadFile("file", fileName, newPath, url)
	fmt.Fprintln(rw, rst)
}

//数据连接串修改
func McmConnChangeInterface(rw http.ResponseWriter, req *http.Request) {
	url_path := req.URL.Path
	seelog.Info("Receive " + req.Method + " from " + req.RemoteAddr + " ,URL.Path: " + url_path)
	seelog.Flush()
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	if strings.EqualFold(req.Method, "post") {
		//删除，先判断是否能够删除，如果不有删除，则发出告警；
		token := req.Header.Get("token")

		if token == "" {
			fmt.Fprintf(rw, "{\"ERROR\":\"Token is inValid, Token is empty \"}")
			return
		}
		if _, err := nJwtUtil.DecodeJwtString(token); err != nil {
			fmt.Fprintf(rw, "{\"ERROR\":\"Token is inValid, err: %v \"}", err)
			return
		}

		if err := req.ParseForm(); err != nil {
			fmt.Fprintf(rw, "{\"ERROR\":\"ParseForm() err: %v\"}", err)
			return
		}
		postForm := req.PostForm
		seelog.Debug("req.PostForm: ", postForm)
		seelog.Flush()
		rst := nFqlProcess.DeleteConn(postForm["dbid"][0])
		seelog.Debug("ConnChange rst: " + rst)
		seelog.Flush()
		fmt.Fprintln(rw, rst)
	} else if strings.EqualFold(req.Method, "get") {
		//新增或修改，则调用nFqlProcess.ProduceConnList()，生成新的连接
		nFqlProcess.ProduceConnList()
		rst := "{\"code\":0}"
		seelog.Debug("ConnChange rst: " + rst)
		seelog.Flush()
		fmt.Fprintln(rw, rst)
	} else {
		http.NotFound(rw, req)
	}
}

//自定义API处理，404
func HttpDefaultInterface(rw http.ResponseWriter, req *http.Request) {
	url_path := req.URL.Path
	seelog.Info("Receive " + req.Method + " from " + req.RemoteAddr + " ,URL.Path: " + url_path)
	seelog.Flush()
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")

	//先判断token状态
	tokenStat := true
	token := req.Header.Get("token")
	if token == "" {
		tokenStat = false
	} else {
		if _, err := nJwtUtil.DecodeJwtString(token); err != nil {
			seelog.Debug("DecodeJwtString error: ", err.Error())
			seelog.Flush()
			tokenStat = false
		}
	}
	//初始化请求变量结构
	formData := make(map[string]interface{})
	if strings.EqualFold(req.Method, "post") {
		// 调用json包的解析，解析请求body
		json.NewDecoder(req.Body).Decode(&formData)
		// fmt.Println("formData:", formData)
		// for key, value := range formData {
		// 	fmt.Println("key:", key, " => value :", value)
		// }

	}

	if len(url_path) > 1 {
		rst := nFqlProcess.DealWithApi(url_path, req.Method, tokenStat, formData)
		seelog.Debug("nFqlProcess.DealWithApi rst: " + rst)
		seelog.Flush()
		fmt.Fprintln(rw, rst)

	} else {
		//返回404
		http.NotFound(rw, req)
	}
}

//开启Http服务
func StartHttpService() {
	http.HandleFunc(nServerConf.CigBinPrefix, CgibinInterface)                                  //配置cgi处理接口
	http.HandleFunc(nServerConf.FileStaticPrefix+"api/user/login", McmUserLoginInterface)       //用户登录接口
	http.HandleFunc(nServerConf.FileStaticPrefix+"api/user/logout", McmUserLogoutInterface)     //用户退出接口
	http.HandleFunc(nServerConf.FileStaticPrefix+"api/user/chktoken", McmUserChktokenInterface) //检查token接口
	http.HandleFunc(nServerConf.FileStaticPrefix+"api/conn/change", McmConnChangeInterface)     //数据库连接变动接口
	http.HandleFunc(nServerConf.FileStaticPrefix+"api/fileUpload", McmFileUploadInterface)      //配置文件上传处理接口
	http.HandleFunc(nServerConf.FileStaticPrefix+"api/fileDelete/", McmFileDeleteInterface)     //配置文件删除处理接口
	http.HandleFunc(nServerConf.FileStaticPrefix+"api/", McmApiInterface)                       //配置api处理接口
	http.HandleFunc(nServerConf.FileStaticPrefix, HttpSiteInterface)                            //配置一般http文件处理接口
	http.HandleFunc("/", HttpDefaultInterface)                                                  //自定义API处理接口

	//设置静态文件路径，但没有日志，不再使用
	//fs := http.FileServer(http.Dir(nServerConf.FileStaticPath))
	//http.Handle(nServerConf.FileStaticPrefix, http.StripPrefix(nServerConf.FileStaticPrefix, fs))

	seelog.Info("Server Listen on " + nServerConf.ServerPort + " by site " + nServerConf.FileStaticPrefix + " ...")
	seelog.Flush()
	//开启http侦听
	err := http.ListenAndServe(":"+nServerConf.ServerPort, nil)
	if err != nil {
		seelog.Error("ListenAndServe error:", err)
		seelog.Flush()
	}
}

func main() {
	var conf_file_name string
	//读取命令行,-c 配置文件路径
	flag.StringVar(&conf_file_name, "c", "../conf/Waiters.conf", "Config File Path")
	v := flag.Bool("v", false, "Show Version Info")
	flag.Parse()

	//显示版本信息
	if *v {
		_, _ = fmt.Fprint(os.Stderr, BinInfo.StringifyMultiLine())
		os.Exit(1)
	}

	//读取主配置文件
	ini_parser := Utils.IniParser{}
	if err := ini_parser.Load(conf_file_name); err != nil {
		fmt.Printf("try load config file[%s] error[%s]\n", conf_file_name, err.Error())
		return
	}
	//读取主配置，包括日志配置/侦听端口
	nServerConf.LogConfigPath = ini_parser.GetString("system", "LogConfigPath")
	nServerConf.ServerPort = ini_parser.GetString("system", "ServerPort")
	nServerConf.DBType = ini_parser.GetString("system", "DBType")

	//加载日志配置
	logger, err := seelog.LoggerFromConfigAsFile(nServerConf.LogConfigPath)
	if err != nil {
		seelog.Critical("err parsing config log file", err)
		return
	}
	seelog.ReplaceLogger(logger)

	//读取数据库配置
	nDBMethed.DBType = nServerConf.DBType
	nDBMethed.Host = ini_parser.GetString(nServerConf.DBType, "Host")
	nDBMethed.Port = ini_parser.GetString(nServerConf.DBType, "Port")
	nDBMethed.Username = ini_parser.GetString(nServerConf.DBType, "Username")
	nDBMethed.Password = ini_parser.GetString(nServerConf.DBType, "Password")
	nDBMethed.Database = ini_parser.GetString(nServerConf.DBType, "Database")
	nDBMethed.DBFile = ini_parser.GetString(nServerConf.DBType, "DBFile")

	if err := nDBMethed.ConnDataBase(); err != nil {
		seelog.Critical("Open "+nDBMethed.DBType+" failed,", err)
		seelog.Flush()
		return
	}

	//数据库初始化建表
	if _, err := nDBMethed.Query("select count(1) from User"); err != nil {
		seelog.Info("Init " + nDBMethed.DBType + " ...")
		seelog.Flush()
		sqlContent := Utils.GetFileContent("../bin/initdb.sql")
		nSql := ""
		for _, sql := range strings.Split(sqlContent, ";") {
			nSql = nSql + sql
			//对于begin的语句，必须要有end
			if strings.Contains(nSql, "begin") && (!strings.Contains(nSql, "end")) {
				nSql = nSql + ";"
				continue
			}
			if len(nSql) < 5 {
				continue
			}
			if _, err := nDBMethed.Exec(nSql); err != nil {
				seelog.Info("sql: ", nSql)
				seelog.Error("Init "+nDBMethed.DBType+" failed,", err)
				seelog.Flush()
				return
			}
			nSql = ""
		}
		seelog.Info("Init " + nDBMethed.DBType + " success !")

	}

	//将nDBMethed赋给nDBProcess作数据处理
	nDBProcess.SetDBMethed(&nDBMethed)

	//将nDBMethed赋给nFqlProcess作数据处理
	nFqlProcess.SetDBMethed(&nDBMethed)
	nFqlProcess.ProduceConnList()

	//读取http配置
	nServerConf.FileStaticPrefix = ini_parser.GetString("http", "FileStaticPrefix")
	if nServerConf.FileStaticPrefix == "" || nServerConf.FileStaticPrefix == "/" {
		nServerConf.FileStaticPrefix = "/waiters/"
	}
	nServerConf.FileStaticPath = ini_parser.GetString("http", "FileStaticPath")
	nServerConf.CigBinPrefix = ini_parser.GetString("http", "CigBinPrefix")
	nServerConf.CigBinPath = ini_parser.GetString("http", "CigBinPath")
	//启动服务侦听
	StartHttpService()
}

/*
私有类型
ConntrackServer
注意：在做json转换的时候，属性的首字母都要大写
*/
package Utils

//主配置参数类
type ServerConf struct {
		LogConfigPath		string
		ServerPort			string
		DBType				string
		FileStaticPrefix	string
		FileStaticPath		string
		CigBinPrefix		string
		CigBinPath			string

}



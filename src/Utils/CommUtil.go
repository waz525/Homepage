/*
通用工具
1 常用类型
2 常用函数
*/
package Utils

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	seelog "github.com/cihub/seelog"
)

/* 获取绝对路径 */
func GetRealPath(path string) string {
	filePath := ""
	if strings.Index(path, "/") == 0 {
		filePath = path
	} else {
		pwd, _ := os.Getwd()
		filePath = pwd + "/" + path
	}
	//filePath = strings.Replace(filePath, "/../" , "/" , -1 )
	filePath = strings.Replace(filePath, "//", "/", -1)
	return filePath
}

/* 字符串转整型 */
func Atoi(s string) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		seelog.Error("Atoi Error: ", err)
		seelog.Flush()
		return 0
	}
	return v
}

/* 数字转字符型 */
func Itoa(v interface{}) string {
	switch v.(type) {
	case int:
		return strconv.Itoa(v.(int))
	case int64:
		return strconv.FormatInt(v.(int64), 10)
	case float64:
		return strconv.FormatFloat(v.(float64), 'f', -1, 64)
	default:
		fmt.Println("Itoa Error: Not support type !")
	}
	return ""
}

/* 执行shell命令 */
func RunShellCmd(cmd string) string {
	//fmt.Println("Running Shell cmd:" , cmd)
	//result, err := exec.Command("/bin/bash", "-c", cmd).Output()
	result, err := exec.Command(cmd).Output()
	if err != nil {
		seelog.Error("Exec Command Error: ", err)
		seelog.Flush()
	}
	return strings.TrimSpace(string(result))
}

/* 判断文件是否存在  存在返回 true 不存在返回false */
func IsFileExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

/* 删除文件 */
func DeleteFile(filename string) bool {
	if IsFileExist(filename) {
		err := os.Remove(filename)
		if err == nil {
			return true
		}
	}
	return false
}

/* 读取文本文件内容 */
func GetFileContent(filepath string) string {
	bytes, err := ioutil.ReadFile(filepath)

	if err != nil {
		seelog.Error("read file: ", filepath, ", error:", err)
		seelog.Flush()
		return ""
	}
	return string(bytes)
}

/* 写入文件 */
func WriteFileContent(filepath, content string) {
	var d1 = []byte(content)
	err := ioutil.WriteFile(filepath, d1, 0666) //写入文件(字节数组)
	if err != nil {
		seelog.Error("Write file: ", filepath, ", error:", err)
		seelog.Flush()
	}
}

/* 将字符串转换成二维字符串数组 */
func Str2List(rows string, line_fd string, str_fd string) [][]string {
	rst := [][]string{}
	lines := strings.Split(rows, line_fd)
	for ind := 0; ind < len(lines); ind++ {
		rst = append(rst, strings.Split(strings.TrimSpace(lines[ind]), str_fd))
	}
	return rst
}

/* 将字符串转换成Map */
func Str2Map(rows string, line_fd string, str_fd string) map[string]string {
	rst := make(map[string]string)
	lines := strings.Split(rows, line_fd)
	for ind := 0; ind < len(lines); ind++ {
		columns := strings.Split(strings.TrimSpace(lines[ind]), str_fd)
		if len(columns) > 1 {
			rst[columns[0]] = columns[1]
		}

	}

	return rst
}

/* 打印二维数组 */
func PrintList(rows [][]string) {
	for ind := 0; ind < len(rows); ind++ {
		line := rows[ind]
		for j := 0; j < len(line); j++ {
			fmt.Printf("" + strconv.Itoa(j) + ":" + line[j] + "\t")
		}
		fmt.Println()
	}
}

/* 从二维数组中查找对应数据 */
func FindListCell(rows [][]string, key string, f_index int, r_index int) string {
	for ind := 0; ind < len(rows); ind++ {
		line := rows[ind]
		if len(line) > f_index && len(line) > r_index {
			if line[f_index] == key {
				return line[r_index]
			}
		}
	}
	return ""
}

/* 查找字符是否在数组/MAP中 */
func InArray(obj interface{}, target interface{}) bool {
	targetValue := reflect.ValueOf(target)
	switch reflect.TypeOf(target).Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < targetValue.Len(); i++ {
			if targetValue.Index(i).Interface() == obj {
				return true
			}
		}
	case reflect.Map:
		if targetValue.MapIndex(reflect.ValueOf(obj)).IsValid() {
			return true
		}
	}

	return false
}

/* 打印类的所有属性 */
func GetClassAttribute(body interface{}) {
	var prop []string
	refType := reflect.TypeOf(body)
	if refType.Kind() != reflect.Struct {
		fmt.Println("Not a structure type.")
	}
	for i := 0; i < refType.NumField(); i++ {
		field := refType.Field(i)
		if field.Anonymous {
			prop = append(prop, field.Name)
			for j := 0; j < field.Type.NumField(); j++ {
				prop = append(prop, field.Type.Field(j).Name)
			}
			continue
		}
		prop = append(prop, field.Name)
	}
	fmt.Printf("%v\n", prop)
}

/* 将Struct转成字符串 */
func Struct2String(v interface{}) string {
	res, _ := json.Marshal(v)
	return string(res)
}

/*将任意类型转成json字符串 */
func Any2Json(v any) string {
	rst := ""

	jsonData, err := json.Marshal(v)
	if err != nil {
		return "{\"ERROR\":\"" + err.Error() + "\"}"
	}
	rst = string(jsonData)
	return rst
}

/* 发送GET请求 */
func HttpGet(url string) string {
	// 超时时间：5秒
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		seelog.Error("HttpGet Error:", err)
		seelog.Flush()
		return ""
	}
	defer resp.Body.Close()
	var buffer [512]byte
	result := bytes.NewBuffer(nil)
	for {
		n, err := resp.Body.Read(buffer[0:])
		result.Write(buffer[0:n])
		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			seelog.Error("HttpGet Error:", err)
			seelog.Flush()
			return ""
		}
	}

	return result.String()
}

/* 发送POST请求 */
func HttpPost(url string, data string, contentType string) string {
	// 超时时间：5秒
	client := &http.Client{Timeout: 5 * time.Second}
	//jsonStr, _ := json.Marshal(data)
	resp, err := client.Post(url, contentType, strings.NewReader(data))
	if err != nil {
		seelog.Error("HttpPost Error:", err)
		seelog.Flush()
		return ""
	}
	defer resp.Body.Close()

	result, _ := ioutil.ReadAll(resp.Body)
	return string(result)
}

/*
产生特定长度的随机字符串
  parameter:
    len: 设定字符串长度
  return:
    string: 随机字符串

*/
func CreateRandomString(len int) string {
	var container string
	var str = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	b := bytes.NewBufferString(str)
	length := b.Len()
	bigInt := big.NewInt(int64(length))
	for i := 0; i < len; i++ {
		randomInt, _ := rand.Int(rand.Reader, bigInt)
		container += string(str[randomInt.Int64()])
	}
	return container
}

/*
使用正则匹配字符串
  parameter:
    str: 源字符串
	exp: 正则字符串
  return:
    []string: 匹配字符串，错误返回nil

*/
func RegexpString(str, exp string) []string {
	reg := regexp.MustCompile(exp)
	if reg == nil {
		seelog.Error("regexp.MustCompile error !")
		seelog.Flush()
		return nil
	}
	return reg.FindAllString(str, -1)
}

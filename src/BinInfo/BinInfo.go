package BinInfo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	//"runtime"
	//"strings"
)

var (
	// 初始化为 unknown，如果编译时没有传入这些值，则为 unknown
	Version        = "unknown"
	Author         = "unknown"
	BuildTime      = "unknown"
	BuildGoVersion = "unknown"
	fileName       = ""
)

// 返回单行格式
func StringifySingleLine() string {
	return fmt.Sprintf("fileName=%s. Version=%s. Author=%s. BuildTime=%s. GoVersion=%s.\n",
		fileName, Version, Author, BuildTime, BuildGoVersion )
}

// 返回多行格式
func StringifyMultiLine() string {
	return fmt.Sprintf("\n%s Version Info:\n\tVersion: %s\n\tAuthor: %s\n\tBuildTime: %s\n\tGoVersion: %s\n\n",
		fileName, Version, Author, BuildTime, BuildGoVersion )
}

func init() {
	filePath, _ := exec.LookPath(os.Args[0])
	fileName = filepath.Base(filePath)
}


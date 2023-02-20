/*
   mime获取方式
*/
package Utils

import (
		"path"
		"mime"
		"strings"
)

//根据文件名获取ContentType
func GetContentTypeByFilePath(filePath string) string {
	extName := path.Ext(filePath)
	ContentType := mime.TypeByExtension(extName)
	if ContentType == "" {
		ContentType = "text/html"
	}
	if strings.Count(ContentType, "charset=" ) == 0  {
		ContentType += "; charset=utf-8"
	}
	return ContentType
}

//根据文件类型获取后缀名，用于新建文件
func GetExtByType(filetype string) string {
	fileEndings, err := mime.ExtensionsByType(filetype)
	if err != nil {
		return ".none"
	}
	return fileEndings[0]
}



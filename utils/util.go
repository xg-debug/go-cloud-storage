package utils

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"strings"
)

// SHA256生成哈希值
func GetSHA256HashCode(file *os.File) string {
	//创建一个基于SHA256算法的hash.Hash接口的对象
	hash := sha256.New()
	_, _ = io.Copy(hash, file)
	//计算哈希值
	bytes := hash.Sum(nil)
	//将字符串编码为16进制格式,返回字符串
	hashCode := hex.EncodeToString(bytes)
	//返回哈希值
	return hashCode

}

// md5加密
func EncodeMd5(data string) string {
	h := md5.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// 判断文件后缀获取类型id
func GetFileTypeInt(filePostfix string) int {
	filePostfix = strings.ToLower(filePostfix)
	if filePostfix == ".doc" || filePostfix == ".docx" || filePostfix == ".txt" || filePostfix == ".pdf" {
		return 1
	}
	if filePostfix == ".jpg" || filePostfix == ".png" || filePostfix == ".gif" || filePostfix == ".jpeg" {
		return 2
	}
	if filePostfix == ".mp4" || filePostfix == ".avi" || filePostfix == ".mov" || filePostfix == ".rmvb" || filePostfix == ".rm" {
		return 3
	}
	if filePostfix == ".mp3" || filePostfix == ".cda" || filePostfix == ".wav" || filePostfix == ".wma" || filePostfix == ".ogg" {
		return 4
	}

	return 5
}

// 将body中=号格式的字符串转为map
func ConvertToMap(str string) map[string]string {
	var resultMap = make(map[string]string)
	values := strings.Split(str, "&")
	for _, value := range values {
		vs := strings.Split(value, "=")
		resultMap[vs[0]] = vs[1]
	}
	return resultMap
}

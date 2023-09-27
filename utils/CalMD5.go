package utils

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"log"
	"os"
)

// CalMD5 计算文件的MD5值
func CalMD5(file *os.File) (string, error) {
	hash := md5.New()

	if _, err := io.Copy(hash, file); err != nil {
		log.Fatalln("Error calculating file MD5:", err)
		return "", err
	}
	// 计算哈希值的字节数组
	hashBytes := hash.Sum(nil)
	// 将字节数组转换为十六进制字符串
	md5Str := hex.EncodeToString(hashBytes)
	return md5Str, nil
}

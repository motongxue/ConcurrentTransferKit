package client

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/motongxue/concurrentChunkTransfer/models"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
)

var (
	// todo 变量抽取到配置文件中
	fileName   = "D:\\my_data\\my_code\\go_code\\ConcurrentTransferKit\\test_in\\test.zip"
	url        = "http://localhost:8080/getFileTransferInfo"
	tcpAddress = "localhost:8081"
	ChunkSize  = 1 << 23 // 8MB
)

func main() {
	file, err := os.Open(fileName)
	defer func() {
		err = file.Close()
		if err != nil {
			log.Fatalln("Error closing file:", err)
			return
		}
	}()
	if err != nil {
		log.Fatalln("Error opening file:", err)
	}
	stat, _ := file.Stat()
	size := stat.Size()
	md5Str, err := calMD5(file)
	if err != nil {
		log.Fatalln("Error calculating file MD5:", err)
	}

	// 创建并发送 FileFragment 结构体
	fileMetaData := models.FileMetaData{
		MD5:       md5Str,
		Name:      filepath.Base(file.Name()),
		FileSize:  size,
		ChunkSize: int64(ChunkSize), // 1MB
	}
	t := fileMetaData.FileSize / fileMetaData.ChunkSize
	if fileMetaData.FileSize%fileMetaData.ChunkSize != 0 {
		t++
	}
	fileMetaData.ChunkNum = int(t)
	log.Println("FileMetaData:", fileMetaData)

	// 发送http请求，获取文件分片信息
	jsonData, err := json.Marshal(fileMetaData)
	if err != nil {
		log.Fatalln("Error marshalling FileMetaData:", err)
	}

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalln("Error creating HTTP request:", err)
	}
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Fatalln("Error sending HTTP request:", err)
	}
	defer response.Body.Close()

	// 读取响应体
	var responseData models.ResponseData
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&responseData); err != nil {
		log.Fatalln("Error decoding response:", err)
	}
	var info models.FileTransferInfo
	info = responseData.Data
	for _, idx := range info.Unreceived {
		conn, err := net.Dial("tcp", tcpAddress)
		if err != nil {
			fmt.Println("Error connecting to server:", err)
			return
		}
		defer conn.Close()
		go func(idx int) {
			// todo 读取文件分片
		}(idx)
	}
	log.Println("File sent successfully")
}

// 计算文件的MD5值
func calMD5(file *os.File) (string, error) {
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
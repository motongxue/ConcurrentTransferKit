package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/motongxue/concurrentChunkTransfer/models"
	"github.com/motongxue/concurrentChunkTransfer/utils"
	"github.com/spf13/viper"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

var (
	fileName   string
	httpUrl    string
	tcpAddress string
	chunkSize  int // 1MB
	bufferSize int // 每次发送 1KB 数据
)

func init() {

	config := viper.New()
	config.AddConfigPath("./conf/")
	config.SetConfigName("application")
	config.SetConfigType("yaml")
	if err := config.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Fatalln("找不到配置文件..")
		} else {
			log.Fatalln("配置文件出错..")
		}
	}

	fileName = config.GetString("client.fileName")
	httpUrl = config.GetString("client.httpUrl")
	tcpAddress = config.GetString("client.tcpAddress")
	chunkSize, _ = utils.GetConfInt(config, "client.chunkSize")
	bufferSize, _ = utils.GetConfInt(config, "client.bufferSize")
	fmt.Printf("fileName:%s\t, httpUrl:%s\t, tcpAddress:%s\t, chunkSize:%d\t, bufferSize:%d\t\n", fileName, httpUrl, tcpAddress, chunkSize, bufferSize)
}

func main() {
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatalln("Error opening file:", err)
	}
	fileMetaData, err := calFileMetaData(file)
	if err != nil {
		log.Fatalln("Error calculating file metadata:", err)
		return
	}
	log.Println("FileMetaData:", fileMetaData)
	// 此时关于文件信息的提取已经完成，可以关闭文件了
	err = file.Close()
	if err != nil {
		log.Fatalln("Error closing file:", err)
		return
	}
	info, err := getFileTransferInfo(fileMetaData)
	if err != nil {
		log.Fatalln("Error getting file transfer info:", err)
		return
	}
	var wg sync.WaitGroup
	for _, idx := range info.Unreceived {
		if idx == -1 {
			// 已经接收完毕
			break
		}
		wg.Add(1)
		go sendSliceFile(idx, fileMetaData, &wg)
	}
	wg.Wait()
	log.Println("File sent successfully")
}

// sendSliceFile 发送文件分片
func sendSliceFile(idx int, fileMetaData models.FileMetaData, wg *sync.WaitGroup) {
	conn, err := net.Dial("tcp", tcpAddress)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()
	startOffset := int64(idx) * fileMetaData.ChunkSize
	endOffset := min(startOffset+fileMetaData.ChunkSize, fileMetaData.FileSize)
	defer (*wg).Done()
	// 写入 FileFragment 结构体信息
	fileFragment := models.FileFragment{
		MD5:     fileMetaData.MD5,
		Current: idx,
	}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(fileFragment); err != nil {
		log.Println("Error encoding FileFragment:", err)
		return
	}
	// 由于多个协程同时对文件指针进行操作，需要对每个协程的文件指针进行独立的操作
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatalln("Error opening file:", err)
	}

	// 将文件指针移动到start
	if _, err = file.Seek(startOffset, io.SeekStart); err != nil {
		log.Fatalln("Error seeking file:", err)
	}
	// 使用缓冲区逐块发送文件
	buffer := make([]byte, bufferSize)
	totalBytesSent := int64(0)

	for {
		n, err := file.Read(buffer)
		totalBytesSent += int64(n)
		if totalBytesSent > endOffset-startOffset {
			n -= int(totalBytesSent - endOffset + startOffset)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println("Error reading file:", err)
			return
		}

		_, err = conn.Write(buffer[:n])
		if err != nil {
			log.Println("Error sending file fileMetaData:", err)
			return
		}
		if totalBytesSent >= endOffset-startOffset {
			break
		}
	}
	fmt.Printf("File sent successfully: start: %d, end: %d, len: %d == %d\n", startOffset, endOffset, endOffset-startOffset, totalBytesSent)
}

// getFileTransferInfo 获取文件分片信息
func getFileTransferInfo(fileMetaData models.FileMetaData) (models.FileTransferInfo, error) {
	// 发送http请求，获取文件分片信息
	jsonData, err := json.Marshal(fileMetaData)
	if err != nil {
		log.Fatalln("Error marshalling FileMetaData:", err)
		return models.FileTransferInfo{}, err
	}

	request, err := http.NewRequest("POST", httpUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalln("Error creating HTTP request:", err)
		return models.FileTransferInfo{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Fatalln("Error sending HTTP request:", err)
		return models.FileTransferInfo{}, err
	}
	defer response.Body.Close()

	// 读取响应体
	var responseData models.ResponseData
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&responseData); err != nil {
		log.Fatalln("Error decoding response:", err)
		return models.FileTransferInfo{}, err
	}
	var info models.FileTransferInfo
	info = responseData.Data
	return info, nil
}

// calFileMetaData 计算文件的MD5值、文件名、文件大小、分片大小、分片数量
func calFileMetaData(file *os.File) (models.FileMetaData, error) {
	stat, _ := file.Stat()
	size := stat.Size()
	md5Str, err := utils.CalMD5(file)
	if err != nil {
		return models.FileMetaData{}, err
	}

	// 创建并发送 FileFragment 结构体
	fileMetaData := models.FileMetaData{
		MD5:       md5Str,
		Name:      filepath.Base(file.Name()),
		FileSize:  size,
		ChunkSize: int64(chunkSize), // 1MB
	}
	t := fileMetaData.FileSize / fileMetaData.ChunkSize
	if fileMetaData.FileSize%fileMetaData.ChunkSize != 0 {
		t++
	}
	fileMetaData.ChunkNum = int(t)
	return fileMetaData, nil
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

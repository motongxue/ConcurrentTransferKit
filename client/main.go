package main

import (
	"fmt"
	"github.com/motongxue/concurrentChunkTransfer/utils"
	"github.com/spf13/viper"
	"log"
	"os"
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
		wg.Add(1)
		go sendSliceFile(idx, fileMetaData, &wg)
	}
	wg.Wait()
	log.Println("File sent successfully")
}

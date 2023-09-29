package utils

import (
	"context"
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"github.com/motongxue/concurrentChunkTransfer/models"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// ReceiveFile 接收文件
func ReceiveFile(redisClient *redis.Client, outputDir string, conn *net.Conn) {
	defer (*conn).Close()

	// 读取 FileFragment 结构体信息
	var fileFragment models.FileFragment
	decoder := json.NewDecoder(*conn)
	if err := decoder.Decode(&fileFragment); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Received FileFragment: %+v\n", fileFragment)

	outputFile, err := os.Create(filepath.Join(outputDir, fileFragment.MD5, strconv.Itoa(fileFragment.Current)))
	if err != nil {
		log.Fatalln(err)
	}
	defer outputFile.Close()

	// 使用缓冲区逐块接收并写入文件
	bufferSize := 1024 // 每次接收 1KB 数据
	buffer := make([]byte, bufferSize)
	totalReceived := int64(0)
	// 从Redis中获取FileMetaData
	fileMetaData := redisClient.Get(context.Background(), FILE_MATEDATA_KEY+fileFragment.MD5)
	if fileMetaData.Err() != nil {
		log.Fatalln("Failed to get file metadata:", err)
		return
	}
	var metaData models.FileMetaData
	err = json.Unmarshal([]byte(fileMetaData.Val()), &metaData)
	if err != nil {
		log.Fatalln("Failed to unmarshal file metadata:", err)
		return
	}
	for totalReceived < metaData.ChunkSize {
		n, err := (*conn).Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalln("Error receiving file data:", err)
		}
		if _, err := outputFile.Write(buffer[:n]); err != nil {
			log.Fatalln("Error writing to file:", err)
			return
		}
		totalReceived += int64(n)
	}
	// 加锁，从Redis中删除该分片
	redisClient.SRem(context.Background(), FILE_TRANSFER_INFO_KEY+fileFragment.MD5, fileFragment.Current)
	// 判断是否已经接收完毕
	if redisClient.SCard(context.Background(), FILE_TRANSFER_INFO_KEY+fileFragment.MD5).Val() == 0 {
		// redis互斥锁，-1表示文件传输完成
		nx := redisClient.SetNX(context.Background(), FILE_TRANSFER_LOCK_KEY+fileFragment.MD5, -1, time.Hour*24)
		// 如果上锁失败
		if nx.Val() == false {
			return
		}
		// 设置isTransmitted为true
		metaData.IsTransmitted = true
		jsonMetaData, err := json.Marshal(metaData)
		if err != nil {
			log.Fatalln("Failed to marshal file metadata:", err)
			return
		}
		// 将FileMetaData写入redis
		redisClient.Set(context.Background(), FILE_MATEDATA_KEY+fileFragment.MD5, jsonMetaData, redis.KeepTTL)

		log.Println("File transfer completed:", fileFragment.MD5)
		// 删除FileMetaData
		//redisClient.Del(context.Background(), FILE_MATEDATA_KEY+fileFragment.MD5)
		// 将文件合并
		MergeFile(redisClient, outputDir, fileFragment.MD5, metaData.Name)
		// 删除redis互斥锁
		redisClient.Del(context.Background(), FILE_TRANSFER_LOCK_KEY+fileFragment.MD5)
	}
}

// MergeFile 合并文件
func MergeFile(redisClient *redis.Client, outputDir, dirName, outputFilename string) {
	// 获取dirName下的所有文件
	dir, err := os.ReadDir(filepath.Join(outputDir, dirName))
	if err != nil {
		log.Fatalln("Error reading directory:", err)
		return
	}
	files := make([]string, 0, len(dir))
	for _, entry := range dir {
		if entry.Name() == outputFilename {
			continue
		}
		files = append(files, entry.Name())
	}
	log.Println("file length:", len(files))
	log.Println("files:", files)
	outputFileName := filepath.Join(outputDir, dirName, outputFilename)
	outputFile, err := os.Create(outputFileName)
	if err != nil {
		log.Fatalln("Error creating output file:", err)
		return
	}
	defer outputFile.Close()
	for _, fileName := range files {
		log.Println("Merging file:", fileName)
		file, err := os.Open(filepath.Join(outputDir, dirName, fileName))
		if err != nil {
			log.Fatalln("Error opening file:", err)
			return
		}
		defer file.Close()
		if _, err := io.Copy(outputFile, file); err != nil {
			log.Fatalln("Error copying file:", err)
			return
		}
	}
	// 在redis中更新FileMetaData
	// 从Redis中获取FileMetaData
	fileMetaData := redisClient.Get(context.Background(), FILE_MATEDATA_KEY+dirName)
	if fileMetaData.Err() != nil {
		log.Fatalln("Failed to get file metadata:", err)
		return
	}
	var metaData models.FileMetaData
	err = json.Unmarshal([]byte(fileMetaData.Val()), &metaData)
	if err != nil {
		log.Fatalln("Failed to unmarshal file metadata:", err)
		return
	}
	metaData.IsCompleted = true
	jsonMetaData, err := json.Marshal(metaData)
	if err != nil {
		log.Fatalln("Failed to marshal file metadata:", err)
		return
	}
	// 将FileMetaData写入redis
	redisClient.Set(context.Background(), FILE_MATEDATA_KEY+dirName, jsonMetaData, redis.KeepTTL)

	log.Println("File merged:", outputFileName)
}

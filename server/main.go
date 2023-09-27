package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
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

var (
	// todo 变量抽取到配置文件中
	redisClient   *redis.Client
	outputDir     = "test_out"
	redisAddr     = "172.22.121.54:20001"
	redisPassword = "3739e394237c4e14a4b3b6bf64524680"
	httpPort      = "8080"
	tcpPort       = "8081"
)

func init() {
	//初始化Redis
	// 创建一个Redis客户端连接
	ctx := context.Background()
	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddr,     // Redis服务器地址和端口
		Password: redisPassword, // Redis服务器密码
		DB:       6,             // 默认使用的数据库
	})

	// 使用Ping检查是否成功连接到Redis
	pong, err := redisClient.Ping(ctx).Result()
	if err != nil {
		fmt.Println("Failed to connect to Redis:", err)
		return
	}
	fmt.Println("Connected to Redis:", pong)
}

func main() {
	// 使用gin
	engine := gin.Default()
	engine.POST("/getFileTransferInfo", getFileTransferInfo)
	go engine.Run(fmt.Sprintf(":%s", httpPort))
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", tcpPort))
	log.Printf("Server started on port %s\n", tcpPort)
	if err != nil {
		log.Fatalln(err)
	}
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalln(err)
		}
		go ReceiveFile(conn)
	}
}

// getFileTransferInfo 获取文件传输信息
func getFileTransferInfo(ctx *gin.Context) {
	var metaData models.FileMetaData
	if err := ctx.ShouldBindJSON(&metaData); err != nil {
		ctx.JSON(200, gin.H{
			"code": 1,
			"msg":  "参数错误",
		})
		return
	}
	log.Println("FileMetaData:", metaData)
	// 将metaData写入redis
	jsonMetaData, err := json.Marshal(metaData)
	if err != nil {
		ctx.JSON(500, gin.H{
			"code": 2,
			"msg":  "序列化失败",
		})
		return
	}
	// 将FileMetaData写入redis
	redisClient.SetEX(context.Background(), "FileMetaData:"+metaData.MD5, jsonMetaData, time.Hour)

	md5 := metaData.MD5
	info := models.FileTransferInfo{
		MD5: md5,
	}
	// 从Redis中判断该dirName是否存在FileTransferInfo
	if redisClient.Exists(context.Background(), "FileTransferInfo:"+md5).Val() == 0 {
		// 不存在，创建
		info.MD5 = metaData.MD5
		info.Unreceived = make([]int, metaData.ChunkNum)
		// 创建目录
		if err := os.MkdirAll(filepath.Join(outputDir, metaData.MD5), os.ModeDir); err != nil {
			log.Fatalln("Error creating output directory:", err)
		}
		for i := 0; i < metaData.ChunkNum; i++ {
			info.Unreceived[i] = i
		}
		log.Println("Unreceived:", info.Unreceived)
		// 以MD5为key的set
		for _, num := range info.Unreceived {
			// FileTransferInfo
			_, err := redisClient.SAdd(context.Background(), "FileTransferInfo:"+md5, num).Result()
			if err != nil {
				log.Fatalln("Failed to add members to set:", err)
				return
			}
		}
	} else {
		// FileTransferInfo存在，从Redis中获取
		result, err := redisClient.MGet(context.Background(), "FileTransferInfo:"+md5).Result()
		// 将result转换为info
		for _, v := range result {
			val, _ := strconv.Atoi(v.(string))
			info.Unreceived = append(info.Unreceived, val)
		}
		if err != nil {
			ctx.JSON(500, gin.H{
				"code": 3,
				"msg":  "获取文件信息失败",
			})
			return
		}
	}
	ctx.JSON(200, gin.H{
		"code": 0,
		"msg":  "成功",
		"data": info,
	})
	return
}

// ReceiveFile 接收文件
func ReceiveFile(conn net.Conn) {
	defer conn.Close()

	// 读取 FileFragment 结构体信息
	var fileFragment models.FileFragment
	decoder := json.NewDecoder(conn)
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
	fileMetaData := redisClient.Get(context.Background(), "FileMetaData:"+fileFragment.MD5)
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
		n, err := conn.Read(buffer)
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
	redisClient.SRem(context.Background(), "FileTransferInfo:"+fileFragment.MD5, fileFragment.Current)
	// 判断是否已经接收完毕
	if redisClient.SCard(context.Background(), "FileTransferInfo:"+fileFragment.MD5).Val() == 0 {
		// 已经接收完毕，删除FileTransferInfo
		redisClient.Del(context.Background(), "FileTransferInfo:"+fileFragment.MD5)
		// 将文件合并
		mergeFile(fileFragment.MD5, metaData.Name)
	}
}

func mergeFile(dirName, outputFilename string) {
	// todo
	panic("implement me")
}

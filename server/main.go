package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/motongxue/concurrentChunkTransfer/models"
	"github.com/motongxue/concurrentChunkTransfer/utils"
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
		go utils.ReceiveFile(redisClient, outputDir, &conn)
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
	log.Println(utils.FILE_MATEDATA_KEY, metaData)
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
	redisClient.SetEX(context.Background(), utils.FILE_MATEDATA_KEY+metaData.MD5, jsonMetaData, time.Hour)

	md5 := metaData.MD5
	info := models.FileTransferInfo{
		MD5: md5,
	}
	// 从Redis中判断该dirName是否存在FileTransferInfo
	if redisClient.Exists(context.Background(), utils.FILE_TRANSFER_INFO_KEY+md5).Val() == 0 {
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
			_, err := redisClient.SAdd(context.Background(), utils.FILE_TRANSFER_INFO_KEY+md5, num).Result()
			if err != nil {
				log.Fatalln("Failed to add members to set:", err)
				return
			}
		}
	} else {
		// FileTransferInfo存在，从Redis中获取
		result, err := redisClient.MGet(context.Background(), utils.FILE_TRANSFER_INFO_KEY+md5).Result()
		// 将result转换为info
		for _, v := range result {
			fmt.Println(v)
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

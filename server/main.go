package server

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"log"
	"net"
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
	// panic 方法未实现
	panic("implement me")
}

// ReceiveFile 接收文件
func ReceiveFile(conn net.Conn) {
	// panic 方法未实现
	panic("implement me")
}

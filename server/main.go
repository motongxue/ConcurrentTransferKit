package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/motongxue/concurrentChunkTransfer/models"
)

var (
	outputDir = "test_out"
	port      = "12345"
)

func main() {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		fmt.Printf("Server started. Listening on port %s...\n", port)

		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		fmt.Println("Client connected:", conn.RemoteAddr())

		// Create the output directory if it doesn't exist
		if _, err := os.Stat(outputDir); os.IsNotExist(err) {
			err = os.MkdirAll(outputDir, os.ModePerm)
			if err != nil {
				fmt.Println("Error creating output directory:", err)
				return
			}
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	// Create a buffer to hold the received fragments
	fileRecord := &models.FileRecord{}
	// receiver from conn
	err := decoder.Decode(fileRecord)
	if err != nil {
		fmt.Println("Error decoding FileRecord JSON:", err)
		return
	}
	fmt.Println("File record received:", fileRecord)

	lock := sync.Mutex{}
	cnt := 0

	for {
		fileFragment := &models.FileFragment{}
		decoder = json.NewDecoder(conn)
		err := decoder.Decode(fileFragment)
		if err != nil {
			fmt.Println("Error decoding FileFragment JSON:", err)
			return
		}
		// 获取到FileFragment
		fmt.Println("File fragment received:", fileFragment)

		// 根据fileRecord的文件名+文件片段ID，创建文件
		sliceFileName := fileRecord.FileName + "_" + fmt.Sprintf("%d", fileFragment.FragmentID)

		// 生成文件
		sliceFile, err := os.Create(filepath.Join(outputDir, sliceFileName))
		if err != nil {
			fmt.Println("Error creating file:", err)
			return
		}
		defer sliceFile.Close()
		// 写入文件
		_, err = sliceFile.Write(fileFragment.Fragment)
		if err != nil {
			fmt.Println("Error writing fragment to file:", err)
			return
		}
		lock.Lock()
		cnt++
		if cnt == fileRecord.NumFragments {
			break
		}
		lock.Unlock()
	}

	// Combine the fragments into a single file
	filePath := filepath.Join(outputDir, fileRecord.FileName) // Adjust the file path
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()
	// 根据上面的fileRecord的文件名+文件片段ID，读取文件
	for fragmentID := 0; fragmentID < fileRecord.NumFragments; fragmentID++ {
		sliceFileName := fileRecord.FileName + "_" + fmt.Sprintf("%d", fragmentID)
		sliceFile, err := os.Open(filepath.Join(outputDir, sliceFileName))
		if err != nil {
			fmt.Println("Error opening file:", err)
			return
		}
		defer sliceFile.Close()
		// 读取文件
		_, err = io.Copy(file, sliceFile)
		if err != nil {
			fmt.Println("Error writing fragment to file:", err)
			return
		}
	}

	fmt.Println("File received:", filePath)
}

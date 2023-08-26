package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/motongxue/concurrentChunkTransfer/models"
)

func sendFileFragment(conn net.Conn, wg *sync.WaitGroup, fragmentID int, fileFragment *models.FileFragment) {
	defer wg.Done()

	// Send the fragment to the server
	// Send the file record to the server
	fileFragmentJSON, err := json.Marshal(*fileFragment)
	if err != nil {
		fmt.Println("Error marshaling file record:", err)
		return
	}

	fmt.Println("Sending fragment:", fileFragment.FragmentID)
	// set write deadline to 10 seconds
	conn.SetWriteDeadline(time.Now().Add(time.Second * 10))
	_, err = conn.Write([]byte(fileFragmentJSON))
	if err != nil {
		fmt.Printf("Error sending fragment %d: %s\n", fragmentID, err)
	}
}

func main() {
	conn, err := net.Dial("tcp", "localhost:12345")
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()

	filePath := "D:\\my_data\\my_code\\go_code\\ConcurrentTransferKit\\test_in\\README.md" // Adjust the file path
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}

	defer file.Close()

	fileRecord := &models.FileRecord{}
	fileRecord.FileName = filepath.Base(file.Name())

	fileInfo, _ := file.Stat()
	fileRecord.FileSize = fileInfo.Size()

	fileRecord.FragmentSize = 64

	// Calculate the number of fragments needed
	fileRecord.NumFragments = int(fileRecord.FileSize) / fileRecord.FragmentSize
	if int(fileRecord.FileSize)%fileRecord.FragmentSize != 0 {
		fileRecord.NumFragments++
	}
	// Calculate the hash of the file
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		fmt.Println("Error calculating file hash:", err)
		return
	}
	fileRecord.HashValue = hex.EncodeToString(hash.Sum(nil))

	// Send the file record to the server
	fileRecordJSON, err := json.Marshal(fileRecord)
	if err != nil {
		fmt.Println("Error marshaling file record:", err)
		return
	}

	_, err = conn.Write([]byte(fileRecordJSON))
	if err != nil {
		fmt.Println("Error sending file record:", err)
		return
	}

	var wg sync.WaitGroup
	// Seek back to the beginning of the file
    if _, err := file.Seek(0, io.SeekStart); err != nil {
        fmt.Println("Error seeking file:", err)
        return
    }
	for fragmentID := 0; fragmentID < fileRecord.NumFragments; fragmentID++ {
		fragment := make([]byte, fileRecord.FragmentSize)
		n, err := file.Read(fragment)
		if err != nil && err != io.EOF {
			fmt.Println("Error reading fragment:", err)
			return
		}
		fragment = fragment[:n]
		fileFragment := &models.FileFragment{
			FragmentID: fragmentID,
			Fragment:   fragment,
		}
		wg.Add(1)
		time.Sleep(10 * time.Millisecond)
		// 并发发送，无法控制顺序
		go sendFileFragment(conn, &wg, fragmentID, fileFragment)
	}

	// Wait for all fragments to be sent
	wg.Wait()

	fmt.Println("File sent:", filePath)
}



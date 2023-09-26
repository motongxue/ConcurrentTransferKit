package models

type FileTransferInfo struct {
	// MD5
	MD5 string `json:"md5"`
	// 未接收分片列表
	Unreceived []int `json:"unreceived"`
}

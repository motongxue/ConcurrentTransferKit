package models

type FileMetaData struct {
	Name     string `json:"name"`
	MD5      string `json:"md5"`
	FileSize int64  `json:"file_size"`
	// 分割的大小
	ChunkSize int64 `json:"chunk_size"`
	// 总分片数
	ChunkNum int `json:"total"`
	// 分片是否接收完成
	IsTransmitted bool
	// 文件合并是否完成
	IsCompleted bool
}

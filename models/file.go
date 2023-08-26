package models

// 文件片段
type FileFragment struct {
	FragmentID int
	Fragment   []byte
}

// 记录发送的文件片段
type FileRecord struct {
	FileName     string // 文件名
	HashValue    string // 文件hash值
	FileSize     int64  // 文件大小
	FragmentSize int    // 文件片段大小
	NumFragments int    // 文件片段数量
}

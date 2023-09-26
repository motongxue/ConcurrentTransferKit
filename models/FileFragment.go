package models

type FileFragment struct {
	// MD5
	MD5 string `json:"md5"`
	// 当前分片
	Current int `json:"current"`
}

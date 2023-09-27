package utils

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
	"strconv"
)

// GetConfInt 从配置文件中读取整数
func GetConfInt(config *viper.Viper, configStr string) (int, error) {
	atoi, err := strconv.Atoi(config.GetString(configStr))
	if err != nil {
		// 不是整数，则考虑是否为二进制
		atoi, err = parseBitwiseExpression(config.GetString(configStr))
		if err != nil {
			log.Println(fmt.Sprintf("Error reading %s:", configStr), err)
			// 说明配置文件中没有配置 chunkSize，使用默认值
			if configStr == "client.chunkSize" {
				return 1 << 20, nil
			} else if configStr == "client.bufferSize" {
				return 1 << 10, nil
			}
		}
	}
	return atoi, nil
}

// parseBitwiseExpression 解析二进制表达式
func parseBitwiseExpression(expression string) (int, error) {
	// 检查字符串是否以 "1 << " 开头
	if len(expression) < 6 || expression[:4] != "1 <<" {
		return 0, fmt.Errorf("无效的位操作表达式")
	}

	// 解析右移的位数
	shiftCount, err := strconv.Atoi(expression[5:])
	if err != nil {
		return 0, err
	}

	// 计算结果
	result := 1 << shiftCount

	return result, nil
}

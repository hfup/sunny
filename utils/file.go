package utils

import (
	"os"

	"encoding/json"

	"gopkg.in/yaml.v3"
)

// 判断文件是否存在
// 参数：
//  - path 文件路径
// 返回：
//  - 是否存在
func IsFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

// 判断是否文件夹
// 参数：
//  - path 文件路径
// 返回：
//  - 是否是文件夹
func IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}


// 读取yaml文件
// 参数：
//  - path 文件路径
//  - dest 目标结构体
// 返回：
//  - 错误
func ReadYamlFile(path string, dest any) error {
	f, err := os.OpenFile(path, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	return yaml.NewDecoder(f).Decode(dest)
}


// 读取json文件
// 参数：
//  - path 文件路径
//  - dest 目标结构体
// 返回：
//  - 错误
func ReadJsonFile(path string, dest any) error {
	f, err := os.OpenFile(path, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(dest)
}
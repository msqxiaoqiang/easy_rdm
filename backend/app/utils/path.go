package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// BizResponse 统一业务响应格式
type BizResponse struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

// ExeRelative 返回相对于可执行文件的路径
func ExeRelative(rel string) string {
	exe, err := os.Executable()
	if err != nil {
		return rel
	}
	exe, _ = filepath.EvalSymlinks(exe)
	return filepath.Join(filepath.Dir(exe), rel)
}

// ResolveBasePath 解析应用根目录
func ResolveBasePath(cfgBasePath string) string {
	if cfgBasePath != "" {
		return cfgBasePath
	}
	exe, err := os.Executable()
	if err != nil {
		wd, _ := os.Getwd()
		return wd
	}
	exe, _ = filepath.EvalSymlinks(exe)
	return filepath.Dir(filepath.Dir(filepath.Dir(exe)))
}

// RemoveFile 安全删除文件
func RemoveFile(path string) {
	if _, err := os.Stat(path); err == nil {
		os.Remove(path)
	}
}

// WritePortToFile 将端口号写入文件
func WritePortToFile(portFile string, port int) error {
	return os.WriteFile(portFile, []byte(fmt.Sprintf("%d", port)), 0644)
}

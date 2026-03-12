package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
)

const gaRPCSock = "/.__gmssh/tmp/rpc.sock"

// ServerMode 当前服务模式，由 main.go 启动时设置
var ServerMode string

type rpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
}

type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   interface{}     `json:"error,omitempty"`
	ID      int             `json:"id"`
}

// CallGA 向 GA 的 rpc.sock 发送 JSON-RPC 请求
func CallGA(method string, params interface{}) (*RPCResponse, error) {
	conn, err := net.DialTimeout("unix", gaRPCSock, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("连接 GA rpc.sock 失败: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(10 * time.Second))

	req := rpcRequest{JSONRPC: "2.0", Method: method, Params: params, ID: 1}
	data, _ := json.Marshal(req)
	conn.Write(append(data, '\n'))

	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		var resp RPCResponse
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			return nil, fmt.Errorf("解析响应失败: %w", err)
		}
		return &resp, nil
	}
	return nil, fmt.Errorf("未收到响应")
}

// LogReport 上报操作日志到 GA
// 非 socket（GMSSH）模式下降级为本地日志，不连接 rpc.sock
func LogReport(pluginName, pluginNameDes, fnName, fnNameDes, data, oName string) error {
	if ServerMode != "socket" {
		log.Printf("[log_report] %s/%s %s/%s data=%s oName=%s\n",
			pluginName, pluginNameDes, fnName, fnNameDes, data, oName)
		return nil
	}
	_, err := CallGA("log_report", map[string]string{
		"plugin_name":     pluginName,
		"plugin_name_des": pluginNameDes,
		"fn_name":         fnName,
		"fn_name_des":     fnNameDes,
		"data":            data,
		"o_name":          oName,
	})
	return err
}

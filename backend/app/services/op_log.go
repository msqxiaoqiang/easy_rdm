package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// OpLogEntry 操作日志条目
type OpLogEntry struct {
	Seq    int64  `json:"seq"`
	Time   int64  `json:"time"`   // Unix 毫秒
	ConnID string `json:"conn_id"`
	Action string `json:"action"` // CREATE, DELETE, RENAME, SET, SET_TTL, ...
	Key    string `json:"key"`
	Detail string `json:"detail"` // 简要描述
}

const (
	opLogFile       = "op_log.jsonl"
	opLogMaxLen     = 50000 // 内存最大条目数
	opLogDefaultLim = 500   // 默认查询返回条目数
)

var (
	opLogMu      sync.RWMutex
	opLogEntries []OpLogEntry
	opLogSeq     int64
	opLogWriter  *bufio.Writer
	opLogFd      *os.File
	opLogDone    chan struct{} // 用于通知 flush goroutine 退出
)

func opLogPath() string {
	return filepath.Join(dataDir, opLogFile)
}

// InitOpLog 启动时从 JSONL 文件加载历史日志
func InitOpLog() {
	opLogMu.Lock()
	defer opLogMu.Unlock()

	p := opLogPath()
	// 读取已有日志
	if f, err := os.Open(p); err == nil {
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 64*1024), 64*1024)
		for scanner.Scan() {
			var e OpLogEntry
			if json.Unmarshal(scanner.Bytes(), &e) == nil {
				opLogEntries = append(opLogEntries, e)
				if e.Seq > opLogSeq {
					opLogSeq = e.Seq
				}
			}
		}
		f.Close()

		// 超出上限，截断保留最新一半
		if len(opLogEntries) > opLogMaxLen {
			opLogEntries = opLogEntries[len(opLogEntries)-opLogMaxLen/2:]
			rewriteOpLogFile()
		}
	}

	// 以追加模式打开写入
	fd, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		fmt.Printf("warning: cannot open op log file: %v\n", err)
		return
	}
	opLogFd = fd
	opLogWriter = bufio.NewWriterSize(fd, 4096)

	// 定期 flush 缓冲区到磁盘
	opLogDone = make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				opLogMu.Lock()
				if opLogWriter != nil {
					opLogWriter.Flush()
				}
				opLogMu.Unlock()
			case <-opLogDone:
				return
			}
		}
	}()
}

// FlushOpLog 立即将缓冲区写入磁盘并停止后台 flush goroutine（关闭时调用）
func FlushOpLog() {
	opLogMu.Lock()
	defer opLogMu.Unlock()
	// 停止 flush goroutine
	if opLogDone != nil {
		close(opLogDone)
		opLogDone = nil
	}
	if opLogWriter != nil {
		opLogWriter.Flush()
	}
	if opLogFd != nil {
		opLogFd.Sync()
		opLogFd.Close()
		opLogFd = nil
		opLogWriter = nil
	}
}

// rewriteOpLogFile 重写文件（截断时调用，调用者必须持有 opLogMu 写锁）
func rewriteOpLogFile() {
	p := opLogPath()
	tmpPath := p + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return
	}
	w := bufio.NewWriterSize(f, 8192)
	for _, e := range opLogEntries {
		if line, err := json.Marshal(e); err == nil {
			w.Write(line)
			w.WriteByte('\n')
		}
	}
	w.Flush()
	f.Sync()
	f.Close()
	// Windows 兼容：先删除目标文件再重命名
	os.Remove(p)
	os.Rename(tmpPath, p)
}

// AddOpLog 记录一条操作日志
func AddOpLog(connID, action, key, detail string) {
	opLogMu.Lock()
	defer opLogMu.Unlock()
	opLogSeq++
	entry := OpLogEntry{
		Seq:    opLogSeq,
		Time:   time.Now().UnixMilli(),
		ConnID: connID,
		Action: action,
		Key:    key,
		Detail: detail,
	}
	opLogEntries = append(opLogEntries, entry)

	// 追加写入 JSONL
	if opLogWriter != nil {
		if line, err := json.Marshal(entry); err == nil {
			opLogWriter.Write(line)
			opLogWriter.WriteByte('\n')
		}
	}

	// 内存超出上限，截断并重写文件
	if len(opLogEntries) > opLogMaxLen {
		opLogEntries = opLogEntries[len(opLogEntries)-opLogMaxLen/2:]
		// 先关闭当前 writer
		if opLogWriter != nil {
			opLogWriter.Flush()
		}
		if opLogFd != nil {
			opLogFd.Close()
		}
		rewriteOpLogFile()
		// 重新打开追加模式
		if fd, err := os.OpenFile(opLogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600); err == nil {
			opLogFd = fd
			opLogWriter = bufio.NewWriterSize(fd, 4096)
		}
	}
}

// RegisterOpLogHandlers 注册操作日志 RPC 方法
func RegisterOpLogHandlers(register func(string, RPCHandlerFunc)) {
	register("get_op_log", handleGetOpLog)
	register("clear_op_log", handleClearOpLog)
}

func handleGetOpLog(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"` // 可选，按连接过滤
		After  int64  `json:"after"`   // 可选，返回此 seq 之后的条目
		Limit  int    `json:"limit"`   // 可选，限制返回条目数（默认 500）
	}
	json.Unmarshal(params, &req)

	if req.Limit <= 0 {
		req.Limit = opLogDefaultLim
	}

	opLogMu.RLock()
	defer opLogMu.RUnlock()

	// 使用二分查找定位 after 起始位置（seq 单调递增）
	startIdx := 0
	if req.After > 0 {
		startIdx = sort.Search(len(opLogEntries), func(i int) bool {
			return opLogEntries[i].Seq > req.After
		})
	}

	var result []OpLogEntry
	for i := startIdx; i < len(opLogEntries); i++ {
		e := opLogEntries[i]
		if req.ConnID != "" && e.ConnID != req.ConnID {
			continue
		}
		result = append(result, e)
		if len(result) >= req.Limit {
			break
		}
	}
	if result == nil {
		result = []OpLogEntry{}
	}

	return result, nil
}

func handleClearOpLog(_ json.RawMessage) (any, error) {
	opLogMu.Lock()
	defer opLogMu.Unlock()
	cleared := len(opLogEntries)
	opLogEntries = opLogEntries[:0]
	// 不重置 opLogSeq，保持单调递增，避免多客户端 after 参数失效
	// 清空文件
	if opLogWriter != nil {
		opLogWriter.Flush()
	}
	if opLogFd != nil {
		opLogFd.Close()
	}
	os.WriteFile(opLogPath(), nil, 0600)
	// 重新打开
	if fd, err := os.OpenFile(opLogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600); err == nil {
		opLogFd = fd
		opLogWriter = bufio.NewWriterSize(fd, 4096)
	}
	return cleared, nil
}

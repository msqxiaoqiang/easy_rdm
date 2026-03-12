package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	dataDir string
	mu      sync.RWMutex
)

// InitStorage 初始化数据存储目录
func InitStorage(dir string) {
	dataDir = dir
	// 创建所需子目录
	dirs := []string{
		"",
		"exports",
	}
	for _, d := range dirs {
		os.MkdirAll(filepath.Join(dataDir, d), 0700)
	}

	// 确保核心 JSON 文件存在
	ensureFile("connections.json", "[]")
	ensureFile("settings.json", "{}")
	ensureFile("session.json", "{}")
	ensureFile("favorites.json", "{}")
	ensureFile("groups.json", "[]")
	ensureFile("group_meta.json", "{}")

	// 迁移旧格式分组数据（__group__名称 → __group__唯一ID + group_meta.json）
	MigrateGroupFormat()

	// 启动 exports 定期清理
	cleanExpiredExports()
	storageCtx, storageCancel = context.WithCancel(context.Background())
	go startExportsCleanupTimer(storageCtx)
}

var (
	storageCtx    context.Context
	storageCancel context.CancelFunc
)

// StopStorage 停止后台清理任务
func StopStorage() {
	if storageCancel != nil {
		storageCancel()
	}
}

func ensureFile(relPath, defaultContent string) {
	p := filepath.Join(dataDir, relPath)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		os.WriteFile(p, []byte(defaultContent), 0600)
	}
}

// ReadJSON 读取 JSON 文件到目标结构
func ReadJSON(relPath string, target interface{}) error {
	mu.RLock()
	defer mu.RUnlock()
	data, err := os.ReadFile(filepath.Join(dataDir, relPath))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// WriteJSON 将数据写入 JSON 文件（原子写入：临时文件 → Rename）
func WriteJSON(relPath string, data interface{}) error {
	mu.Lock()
	defer mu.Unlock()
	return writeJSONInternal(relPath, data)
}

// writeJSONInternal 不加锁的内部写入，调用方必须已持有 mu.Lock
func writeJSONInternal(relPath string, data interface{}) error {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	p := filepath.Join(dataDir, relPath)
	os.MkdirAll(filepath.Dir(p), 0700)
	tmpPath := p + ".tmp"
	if err := os.WriteFile(tmpPath, bytes, 0600); err != nil {
		return err
	}
	return os.Rename(tmpPath, p)
}

// UpdateJSON 在持锁状态下完成读-修改-写，避免并发竞态
func UpdateJSON(relPath string, target interface{}, updateFn func() error) error {
	mu.Lock()
	defer mu.Unlock()
	data, err := os.ReadFile(filepath.Join(dataDir, relPath))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return err
	}
	if err := updateFn(); err != nil {
		return err
	}
	return writeJSONInternal(relPath, target)
}

// GetDataDir 返回数据目录路径
func GetDataDir() string {
	return dataDir
}

// MigrateGroupFormat 迁移旧格式：__group__名称 → __group__唯一ID + group_meta.json
// 旧格式: groups.json = ["connId", "__group__生产环境"], connections.group = "生产环境"
// 新格式: groups.json = ["connId", "__group__grp_xxx"], group_meta.json = {"grp_xxx": "生产环境"}, connections.group = "grp_xxx"
func MigrateGroupFormat() {
	var groupMeta map[string]string
	if err := ReadJSON("group_meta.json", &groupMeta); err == nil && len(groupMeta) > 0 {
		return // 已有 group_meta，不需要迁移
	}

	var groups []string
	if err := ReadJSON("groups.json", &groups); err != nil {
		return
	}

	// 检查是否有旧格式的 __group__名称 条目（非唯一ID格式）
	const groupPrefix = "__group__"
	const grpIDPrefix = "grp_"
	hasOldFormat := false
	for _, key := range groups {
		if len(key) > len(groupPrefix) && key[:len(groupPrefix)] == groupPrefix {
			suffix := key[len(groupPrefix):]
			if len(suffix) < len(grpIDPrefix) || suffix[:len(grpIDPrefix)] != grpIDPrefix {
				hasOldFormat = true
				break
			}
		}
	}
	if !hasOldFormat {
		return // 没有旧格式条目，无需迁移
	}

	// 执行迁移
	newMeta := make(map[string]string)
	nameToID := make(map[string]string) // 旧分组名 → 新唯一ID

	// 第一遍：为每个旧格式分组生成唯一 ID
	for i, key := range groups {
		if len(key) > len(groupPrefix) && key[:len(groupPrefix)] == groupPrefix {
			suffix := key[len(groupPrefix):]
			if len(suffix) >= len(grpIDPrefix) && suffix[:len(grpIDPrefix)] == grpIDPrefix {
				continue // 已经是新格式
			}
			// 旧格式：suffix 是分组名
			groupName := suffix
			newID := fmt.Sprintf("grp_%d%s", time.Now().UnixNano(), randMigrateStr(4))
			nameToID[groupName] = newID
			newMeta[newID] = groupName
			groups[i] = groupPrefix + newID
		}
	}

	// 第二遍：更新 connections.json 中的 group 字段
	var conns []map[string]interface{}
	if err := ReadJSON("connections.json", &conns); err == nil {
		changed := false
		for _, c := range conns {
			if g, ok := c["group"].(string); ok && g != "" {
				if newID, exists := nameToID[g]; exists {
					c["group"] = newID
					changed = true
				}
			}
		}
		if changed {
			WriteJSON("connections.json", conns)
		}
	}

	WriteJSON("groups.json", groups)
	WriteJSON("group_meta.json", newMeta)
	fmt.Printf("分组格式迁移完成: %d 个分组\n", len(newMeta))
}

// randMigrateStr 迁移用随机字符串（避免与 key_handlers.go 中的 randStr 冲突）
func randMigrateStr(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	seed := time.Now().UnixNano()
	for i := range b {
		idx := seed % int64(len(letters))
		if idx < 0 {
			idx = -idx
		}
		b[i] = letters[idx]
		seed = seed*1103515245 + 12345
	}
	return string(b)
}

// cleanExpiredExports 清理超过 24 小时的导出文件
func cleanExpiredExports() {
	exportsDir := filepath.Join(dataDir, "exports")
	entries, err := os.ReadDir(exportsDir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-24 * time.Hour)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(exportsDir, entry.Name()))
			fmt.Printf("清理过期导出文件: %s\n", entry.Name())
		}
	}
}

// startExportsCleanupTimer 每小时执行一次导出文件清理
func startExportsCleanupTimer(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			cleanExpiredExports()
		case <-ctx.Done():
			return
		}
	}
}

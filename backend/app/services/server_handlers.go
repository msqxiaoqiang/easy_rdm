package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

)

// RegisterServerHandlers 注册 P3 运维工具相关的 RPC 方法
func RegisterServerHandlers(register func(string, RPCHandlerFunc)) {
	// Redis 配置管理
	register("config_get", handleConfigGet)
	register("config_set", handleConfigSet)
	register("config_rewrite", handleConfigRewrite)

	// 客户端管理
	register("client_list", handleClientList)
	register("client_kill", handleClientKill)

	// 慢日志
	register("slowlog_get", handleSlowlogGet)
	register("slowlog_reset", handleSlowlogReset)

	// 数据备份
	register("persistence_info", handlePersistenceInfo)
	register("bgsave", handleBgsave)
	register("bgrewriteaof", handleBgrewriteaof)

	// ACL 权限管理
	register("acl_list", handleACLList)
	register("acl_setuser", handleACLSetUser)
	register("acl_deluser", handleACLDelUser)
	register("acl_whoami", handleACLWhoAmI)
}

// ========== Redis 配置管理 ==========

func handleConfigGet(params json.RawMessage) (any, error) {
	var req struct {
		ConnID  string `json:"conn_id"`
		Pattern string `json:"pattern"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	pattern := req.Pattern
	if pattern == "" {
		pattern = "*"
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	result, err := conn.Cmd().ConfigGet(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	// result is []interface{} with alternating key/value pairs
	items := make([]map[string]string, 0, len(result)/2)
	for i := 0; i+1 < len(result); i += 2 {
		k, _ := result[i].(string)
		v, _ := result[i+1].(string)
		items = append(items, map[string]string{"key": k, "value": v})
	}

	return items, nil
}

func handleConfigSet(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Key    string `json:"key"`
		Value  string `json:"value"`
	}
	if err := json.Unmarshal(params, &req); err != nil || req.Key == "" {
		return nil, fmt.Errorf("参数错误: key 必填")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	if err := conn.Cmd().ConfigSet(ctx, req.Key, req.Value).Err(); err != nil {
		return nil, err
	}

	AddOpLog(req.ConnID, "config_set", req.Key, fmt.Sprintf("%s = %s", req.Key, req.Value))
	return nil, nil
}

func handleConfigRewrite(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	if err := conn.Cmd().ConfigRewrite(ctx).Err(); err != nil {
		return nil, err
	}

	AddOpLog(req.ConnID, "config_rewrite", "", "CONFIG REWRITE")
	return nil, nil
}

// ========== 客户端管理 ==========

func handleClientList(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	result, err := conn.Cmd().ClientList(ctx).Result()
	if err != nil {
		return nil, err
	}

	clients := parseClientList(result)
	return clients, nil
}

func parseClientList(raw string) []map[string]string {
	var clients []map[string]string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		client := make(map[string]string)
		for _, field := range strings.Fields(line) {
			parts := strings.SplitN(field, "=", 2)
			if len(parts) == 2 {
				client[parts[0]] = parts[1]
			}
		}
		if len(client) > 0 {
			clients = append(clients, client)
		}
	}
	return clients
}

func handleClientKill(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Addr   string `json:"addr"`
		ID     string `json:"id"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	// Use CLIENT KILL ID if provided, otherwise by ADDR
	var err error
	if req.ID != "" {
		id, parseErr := strconv.ParseInt(req.ID, 10, 64)
		if parseErr != nil {
			return nil, fmt.Errorf("无效的客户端 ID")
		}
		err = conn.Do(ctx, "CLIENT", "KILL", "ID", id).Err()
	} else if req.Addr != "" {
		err = conn.Cmd().ClientKill(ctx, req.Addr).Err()
	} else {
		return nil, fmt.Errorf("addr 或 id 必填")
	}

	if err != nil {
		return nil, err
	}

	target := req.Addr
	if req.ID != "" {
		target = "ID:" + req.ID
	}
	AddOpLog(req.ConnID, "client_kill", target, fmt.Sprintf("CLIENT KILL %s", target))
	return nil, nil
}

// ========== 慢日志 ==========

func handleSlowlogGet(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Count  int64  `json:"count"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	count := req.Count
	if count <= 0 {
		count = 128
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	// SLOWLOG GET count
	result, err := conn.Do(ctx, "SLOWLOG", "GET", count).Result()
	if err != nil {
		return nil, err
	}

	entries := parseSlowlogEntries(result)

	// Also get SLOWLOG LEN and current threshold
	lenResult, _ := conn.Do(ctx, "SLOWLOG", "LEN").Result()
	thresholdResult, _ := conn.Cmd().ConfigGet(ctx, "slowlog-log-slower-than").Result()

	totalLen, _ := lenResult.(int64)
	threshold := ""
	if len(thresholdResult) >= 2 {
		threshold, _ = thresholdResult[1].(string)
	}

	return map[string]interface{}{
		"entries":   entries,
		"total":     totalLen,
		"threshold": threshold,
	}, nil
}

type slowlogEntry struct {
	ID            int64    `json:"id"`
	Timestamp     int64    `json:"timestamp"`
	Duration      int64    `json:"duration"`
	Command       string   `json:"command"`
	ClientAddr    string   `json:"client_addr"`
	ClientName    string   `json:"client_name"`
}

func parseSlowlogEntries(result interface{}) []slowlogEntry {
	var entries []slowlogEntry
	items, ok := result.([]interface{})
	if !ok {
		return entries
	}

	for _, item := range items {
		fields, ok := item.([]interface{})
		if !ok || len(fields) < 4 {
			continue
		}

		entry := slowlogEntry{}
		entry.ID, _ = fields[0].(int64)
		entry.Timestamp, _ = fields[1].(int64)
		entry.Duration, _ = fields[2].(int64)

		// fields[3] is the command args array
		if args, ok := fields[3].([]interface{}); ok {
			parts := make([]string, 0, len(args))
			for _, a := range args {
				switch v := a.(type) {
				case string:
					parts = append(parts, v)
				default:
					parts = append(parts, fmt.Sprintf("%v", v))
				}
			}
			entry.Command = strings.Join(parts, " ")
		}

		// fields[4] is client addr (Redis 4.0+)
		if len(fields) > 4 {
			entry.ClientAddr, _ = fields[4].(string)
		}
		// fields[5] is client name (Redis 4.0+)
		if len(fields) > 5 {
			entry.ClientName, _ = fields[5].(string)
		}

		entries = append(entries, entry)
	}
	return entries
}

func handleSlowlogReset(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	if err := conn.Do(ctx, "SLOWLOG", "RESET").Err(); err != nil {
		return nil, err
	}

	AddOpLog(req.ConnID, "slowlog_reset", "", "SLOWLOG RESET")
	return nil, nil
}

// ========== 数据备份 ==========

func handlePersistenceInfo(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 5*time.Second)
	defer cancel()

	info, err := conn.Cmd().Info(ctx, "persistence").Result()
	if err != nil {
		return nil, err
	}

	sections := parseInfoSections(info)
	persistence := sections["Persistence"]
	if persistence == nil {
		persistence = make(map[string]string)
	}

	return persistence, nil
}

func handleBgsave(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	if err := conn.Cmd().BgSave(ctx).Err(); err != nil {
		return nil, err
	}

	AddOpLog(req.ConnID, "bgsave", "", "BGSAVE")
	return nil, nil
}

func handleBgrewriteaof(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	if err := conn.Cmd().BgRewriteAOF(ctx).Err(); err != nil {
		return nil, err
	}

	AddOpLog(req.ConnID, "bgrewriteaof", "", "BGREWRITEAOF")
	return nil, nil
}

// ========== ACL 权限管理 ==========

func handleACLWhoAmI(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}
	ctx, cancel := context.WithTimeout(conn.Ctx, 5*time.Second)
	defer cancel()

	result, err := conn.Do(ctx, "ACL", "WHOAMI").Result()
	if err != nil {
		return nil, err
	}
	return result, nil
}

func handleACLList(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}
	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	// ACL USERS to get all usernames
	usersResult, err := conn.Do(ctx, "ACL", "USERS").Result()
	if err != nil {
		return nil, err
	}

	usernames, ok := usersResult.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected ACL USERS response")
	}

	// Get current user
	whoami, _ := conn.Do(ctx, "ACL", "WHOAMI").Result()
	currentUser, _ := whoami.(string)

	// For each user, get details via ACL GETUSER
	users := make([]map[string]interface{}, 0, len(usernames))
	for _, u := range usernames {
		username, _ := u.(string)
		if username == "" {
			continue
		}

		detail, err := conn.Do(ctx, "ACL", "GETUSER", username).Result()
		if err != nil {
			users = append(users, map[string]interface{}{
				"username": username,
				"error":    err.Error(),
			})
			continue
		}

		user := parseACLGetUser(username, detail)
		user["is_current"] = username == currentUser
		users = append(users, user)
	}

	return users, nil
}

func parseACLGetUser(username string, result interface{}) map[string]interface{} {
	user := map[string]interface{}{
		"username": username,
	}

	items, ok := result.([]interface{})
	if !ok || len(items) < 2 {
		return user
	}

	// ACL GETUSER returns alternating key/value pairs
	for i := 0; i+1 < len(items); i += 2 {
		key, _ := items[i].(string)
		val := items[i+1]

		switch key {
		case "flags":
			if flags, ok := val.([]interface{}); ok {
				flagStrs := make([]string, 0, len(flags))
				for _, f := range flags {
					if s, ok := f.(string); ok {
						flagStrs = append(flagStrs, s)
					}
				}
				user["flags"] = flagStrs
			}
		case "passwords":
			if pwds, ok := val.([]interface{}); ok {
				user["password_count"] = len(pwds)
			}
		case "commands":
			if s, ok := val.(string); ok {
				user["commands"] = s
			}
		case "keys":
			if s, ok := val.(string); ok {
				user["keys"] = s
			}
		case "channels":
			if s, ok := val.(string); ok {
				user["channels"] = s
			}
		}
	}

	return user
}

func handleACLSetUser(params json.RawMessage) (any, error) {
	var req struct {
		ConnID   string   `json:"conn_id"`
		Username string   `json:"username"`
		Rules    []string `json:"rules"`
	}
	if err := json.Unmarshal(params, &req); err != nil || req.Username == "" {
		return nil, fmt.Errorf("参数错误: username 必填")
	}
	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}
	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	// Build ACL SETUSER command args
	args := []interface{}{"ACL", "SETUSER", req.Username}
	for _, rule := range req.Rules {
		args = append(args, rule)
	}

	if err := conn.Do(ctx, args...).Err(); err != nil {
		return nil, err
	}

	AddOpLog(req.ConnID, "acl_setuser", req.Username, fmt.Sprintf("ACL SETUSER %s %s", req.Username, strings.Join(req.Rules, " ")))
	return nil, nil
}

func handleACLDelUser(params json.RawMessage) (any, error) {
	var req struct {
		ConnID   string `json:"conn_id"`
		Username string `json:"username"`
	}
	if err := json.Unmarshal(params, &req); err != nil || req.Username == "" {
		return nil, fmt.Errorf("参数错误: username 必填")
	}
	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}
	ctx, cancel := context.WithTimeout(conn.Ctx, 10*time.Second)
	defer cancel()

	if err := conn.Do(ctx, "ACL", "DELUSER", req.Username).Err(); err != nil {
		return nil, err
	}

	AddOpLog(req.ConnID, "acl_deluser", req.Username, fmt.Sprintf("ACL DELUSER %s", req.Username))
	return nil, nil
}

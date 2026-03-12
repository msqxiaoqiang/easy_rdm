package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"easy_rdm/app/utils"

	"github.com/go-redis/redis/v8"
)

// RPCHandlerFunc RPC 业务处理函数签名
type RPCHandlerFunc func(params json.RawMessage) (any, error)

// RegisterHandlers 注册所有业务 RPC 方法
func RegisterHandlers(register func(string, RPCHandlerFunc)) {
	// 连接管理
	register("connect", handleConnect)
	register("disconnect", handleDisconnect)
	register("test_connection", handleTestConnection)
	register("get_connections", handleGetConnections)
	register("save_connection", handleSaveConnection)
	register("delete_connection", handleDeleteConnection)
	register("reorder_connections", handleReorderConnections)
	register("save_groups", handleSaveGroups)
	register("get_groups", handleGetGroups)
	register("get_group_meta", handleGetGroupMeta)
	register("save_group_meta", handleSaveGroupMeta)

	// 设置
	register("get_settings", handleGetSettings)
	register("save_settings", handleSaveSettings)

	// 基础设施
	register("ping", handlePing)
	register("save_session", handleSaveSession)
	register("get_session", handleGetSession)
	register("get_sentinel_info", handleGetSentinelInfo)
	register("get_cluster_info", handleGetClusterInfo)

	// Key 操作、DB、CLI、服务器状态等
	RegisterKeyHandlers(register)

	// 集合类型字段级操作（Hash/List/Set/ZSet）
	RegisterCollectionHandlers(register)

	// 扩展类型操作（Stream/Geo/Bitmap/Bitfield/HLL）
	RegisterExtendedHandlers(register)

	// 运维工具（配置管理、客户端管理、慢日志、数据备份）
	RegisterServerHandlers(register)

	// 数据导入/导出
	RegisterExportHandlers(register)

	// 实时功能（Pub/Sub、MONITOR、延迟诊断）
	RegisterRealtimeHandlers(register)

	// 内存分析
	RegisterMemoryHandlers(register)

	// Lua 脚本
	RegisterLuaHandlers(register)

	// 批量操作与数据迁移
	RegisterBatchHandlers(register)

	// 操作日志
	RegisterOpLogHandlers(register)

	// 收藏夹
	RegisterFavoritesHandlers(register)

	// 自定义解码器
	RegisterDecoderHandlers(register)
}

// ========== 连接管理 ==========

func handleConnect(params json.RawMessage) (any, error) {
	var req struct {
		ID string  `json:"id"`
		DB *int    `json:"db"` // 可选，覆盖配置中的默认 DB
	}
	if err := json.Unmarshal(params, &req); err != nil || req.ID == "" {
		return nil, fmt.Errorf("参数错误: id 必填")
	}

	// 从持久化存储加载连接配置
	var conns []map[string]interface{}
	if err := ReadJSON("connections.json", &conns); err != nil {
		return nil, fmt.Errorf("读取连接配置失败")
	}

	var connMap map[string]interface{}
	for _, c := range conns {
		if cid, _ := c["id"].(string); cid == req.ID {
			connMap = c
			break
		}
	}
	if connMap == nil {
		return nil, fmt.Errorf("连接不存在")
	}

	// 构建 ConnConfig
	cfg := ConnConfig{ID: req.ID}
	if v, ok := connMap["name"].(string); ok {
		cfg.Name = v
	}
	if v, ok := connMap["host"].(string); ok {
		cfg.Host = v
	}
	if v, ok := connMap["port"].(float64); ok {
		cfg.Port = int(v)
	}
	if v, ok := connMap["username"].(string); ok {
		cfg.Username = v
	}
	if v, ok := connMap["conn_type"].(string); ok {
		cfg.ConnType = v
	}
	if v, ok := connMap["unix_socket"].(string); ok {
		cfg.UnixSocket = v
	}
	if v, ok := connMap["db"].(float64); ok {
		cfg.DB = int(v)
	}
	if v, ok := connMap["conn_timeout"].(float64); ok {
		cfg.ConnTimeout = time.Duration(v) * time.Second
	}
	if v, ok := connMap["exec_timeout"].(float64); ok {
		cfg.ExecTimeout = time.Duration(v) * time.Second
	}
	// TLS/SSL
	if v, ok := connMap["use_tls"].(bool); ok {
		cfg.UseTLS = v
	}
	if v, ok := connMap["tls_cert_file"].(string); ok {
		cfg.TLSCertFile = v
	}
	if v, ok := connMap["tls_key_file"].(string); ok {
		cfg.TLSKeyFile = v
	}
	if v, ok := connMap["tls_ca_file"].(string); ok {
		cfg.TLSCAFile = v
	}
	if v, ok := connMap["tls_skip_verify"].(bool); ok {
		cfg.TLSSkipVerify = v
	}
	// SSH 隧道
	if v, ok := connMap["use_ssh"].(bool); ok {
		cfg.UseSSH = v
	}
	if v, ok := connMap["ssh_host"].(string); ok {
		cfg.SSHHost = v
	}
	if v, ok := connMap["ssh_port"].(float64); ok {
		cfg.SSHPort = int(v)
	}
	if v, ok := connMap["ssh_username"].(string); ok {
		cfg.SSHUsername = v
	}
	if v, ok := connMap["ssh_private_key"].(string); ok {
		cfg.SSHPrivateKey = v
	}
	if v, ok := connMap["ssh_passphrase"].(string); ok {
		cfg.SSHPassphrase = v
	}
	// 网络代理
	if v, ok := connMap["use_proxy"].(bool); ok {
		cfg.UseProxy = v
	}
	if v, ok := connMap["proxy_type"].(string); ok {
		cfg.ProxyType = v
	}
	if v, ok := connMap["proxy_host"].(string); ok {
		cfg.ProxyHost = v
	}
	if v, ok := connMap["proxy_port"].(float64); ok {
		cfg.ProxyPort = int(v)
	}
	if v, ok := connMap["proxy_username"].(string); ok {
		cfg.ProxyUsername = v
	}
	if v, ok := connMap["proxy_password"].(string); ok {
		cfg.ProxyPassword = v
	}
	// 哨兵模式
	if v, ok := connMap["use_sentinel"].(bool); ok {
		cfg.UseSentinel = v
	}
	if v, ok := connMap["sentinel_addrs"].(string); ok {
		cfg.SentinelAddrs = v
	}
	if v, ok := connMap["sentinel_master_name"].(string); ok {
		cfg.SentinelMasterName = v
	}
	// 集群模式
	if v, ok := connMap["use_cluster"].(bool); ok {
		cfg.UseCluster = v
	}
	if v, ok := connMap["cluster_addrs"].(string); ok {
		cfg.ClusterAddrs = v
	}

	// 前端传入的 db 覆盖配置默认值（恢复上次选中的 DB）
	if req.DB != nil {
		cfg.DB = *req.DB
	}

	// 解密密码
	if pwd, ok := connMap["password"].(string); ok && pwd != "" {
		if encrypted, _ := connMap["password_encrypted"].(bool); encrypted {
			decrypted, err := utils.Decrypt(pwd)
			if err != nil {
				return nil, fmt.Errorf("密码解密失败")
			}
			cfg.Password = decrypted
		} else {
			cfg.Password = pwd
		}
	}
	// 解密 SSH 密码
	if pwd, ok := connMap["ssh_password"].(string); ok && pwd != "" {
		if encrypted, _ := connMap["ssh_password_encrypted"].(bool); encrypted {
			decrypted, err := utils.Decrypt(pwd)
			if err != nil {
				return nil, fmt.Errorf("SSH 密码解密失败")
			}
			cfg.SSHPassword = decrypted
		} else {
			cfg.SSHPassword = pwd
		}
	}
	// 解密哨兵密码
	if pwd, ok := connMap["sentinel_password"].(string); ok && pwd != "" {
		if encrypted, _ := connMap["sentinel_password_encrypted"].(bool); encrypted {
			decrypted, err := utils.Decrypt(pwd)
			if err != nil {
				return nil, fmt.Errorf("哨兵密码解密失败")
			}
			cfg.SentinelPassword = decrypted
		} else {
			cfg.SentinelPassword = pwd
		}
	}

	conn, err := Connect(&cfg)
	if err != nil {
		return nil, err
	}
	info, _ := conn.GetServerInfo()
	AddOpLog(req.ID, "CONNECT", "", cfg.Name+"@"+cfg.Host)
	return map[string]interface{}{"info": info}, nil
}

func handleDisconnect(params json.RawMessage) (any, error) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	AddOpLog(req.ID, "DISCONNECT", "", "")
	CleanupRealtimeSessions(req.ID)
	Disconnect(req.ID)
	RemoveBuffers(req.ID)
	return nil, nil
}

func handleTestConnection(params json.RawMessage) (any, error) {
	var raw struct {
		ConnConfig
		PasswordEncrypted         bool `json:"password_encrypted"`
		SSHPasswordEncrypted      bool `json:"ssh_password_encrypted"`
		SentinelPasswordEncrypted bool `json:"sentinel_password_encrypted"`
	}
	if err := json.Unmarshal(params, &raw); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	cfg := raw.ConnConfig
	// 前端传的是秒数（如 10），json.Unmarshal 到 time.Duration 会变成纳秒
	// 需要乘以 time.Second 转换为正确的 Duration
	if cfg.ConnTimeout > 0 && cfg.ConnTimeout < time.Second {
		cfg.ConnTimeout = cfg.ConnTimeout * time.Second
	}
	if cfg.ExecTimeout > 0 && cfg.ExecTimeout < time.Second {
		cfg.ExecTimeout = cfg.ExecTimeout * time.Second
	}
	if cfg.ConnTimeout <= 0 {
		cfg.ConnTimeout = 10 * time.Second
	}
	if cfg.ExecTimeout <= 0 {
		cfg.ExecTimeout = 10 * time.Second
	}

	// 编辑模式下密码为空但传了 ID + password_encrypted，从存储中查找已加密的密码
	needLookup := (cfg.Password == "" && raw.PasswordEncrypted) ||
		(cfg.SSHPassword == "" && raw.SSHPasswordEncrypted) ||
		(cfg.SentinelPassword == "" && raw.SentinelPasswordEncrypted)
	if needLookup && cfg.ID != "" {
		var conns []map[string]interface{}
		if err := ReadJSON("connections.json", &conns); err == nil {
			for _, c := range conns {
				if cid, _ := c["id"].(string); cid == cfg.ID {
					// Redis 密码
					if cfg.Password == "" && raw.PasswordEncrypted {
						if pwd, ok := c["password"].(string); ok && pwd != "" {
							if enc, _ := c["password_encrypted"].(bool); enc {
								decrypted, err := utils.Decrypt(pwd)
								if err != nil {
									return nil, fmt.Errorf("密码解密失败")
								}
								cfg.Password = decrypted
							} else {
								cfg.Password = pwd
							}
						}
					}
					// SSH 密码
					if cfg.SSHPassword == "" && raw.SSHPasswordEncrypted {
						if pwd, ok := c["ssh_password"].(string); ok && pwd != "" {
							if enc, _ := c["ssh_password_encrypted"].(bool); enc {
								decrypted, err := utils.Decrypt(pwd)
								if err != nil {
									return nil, fmt.Errorf("SSH 密码解密失败")
								}
								cfg.SSHPassword = decrypted
							} else {
								cfg.SSHPassword = pwd
							}
						}
					}
					// 哨兵密码
					if cfg.SentinelPassword == "" && raw.SentinelPasswordEncrypted {
						if pwd, ok := c["sentinel_password"].(string); ok && pwd != "" {
							if enc, _ := c["sentinel_password_encrypted"].(bool); enc {
								decrypted, err := utils.Decrypt(pwd)
								if err != nil {
									return nil, fmt.Errorf("哨兵密码解密失败")
								}
								cfg.SentinelPassword = decrypted
							} else {
								cfg.SentinelPassword = pwd
							}
						}
					}
					break
				}
			}
		}
	}

	// 使用临时 ID 避免污染连接池
	testID := "__test_" + time.Now().Format("20060102150405.000")
	cfg.ID = testID
	conn, err := Connect(&cfg)
	if err != nil {
		return nil, err
	}
	// 测试完立即断开
	Disconnect(testID)
	_ = conn
	return "pong", nil
}

func handleGetConnections(params json.RawMessage) (any, error) {
	var conns []map[string]interface{}
	if err := ReadJSON("connections.json", &conns); err != nil {
		conns = []map[string]interface{}{}
	}
	return conns, nil
}

func handleSaveConnection(params json.RawMessage) (any, error) {
	var conn map[string]interface{}
	if err := json.Unmarshal(params, &conn); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	// 加密密码字段（仅对未加密的明文密码进行加密）
	if pwd, ok := conn["password"].(string); ok && pwd != "" {
		alreadyEncrypted, _ := conn["password_encrypted"].(bool)
		if !alreadyEncrypted {
			encrypted, err := utils.Encrypt(pwd)
			if err == nil {
				conn["password"] = encrypted
				conn["password_encrypted"] = true
			}
		}
	}
	// 加密 SSH 密码
	if pwd, ok := conn["ssh_password"].(string); ok && pwd != "" {
		alreadyEncrypted, _ := conn["ssh_password_encrypted"].(bool)
		if !alreadyEncrypted {
			encrypted, err := utils.Encrypt(pwd)
			if err == nil {
				conn["ssh_password"] = encrypted
				conn["ssh_password_encrypted"] = true
			}
		}
	}
	// 加密哨兵密码
	if pwd, ok := conn["sentinel_password"].(string); ok && pwd != "" {
		alreadyEncrypted, _ := conn["sentinel_password_encrypted"].(bool)
		if !alreadyEncrypted {
			encrypted, err := utils.Encrypt(pwd)
			if err == nil {
				conn["sentinel_password"] = encrypted
				conn["sentinel_password_encrypted"] = true
			}
		}
	}

	var conns []map[string]interface{}
	ReadJSON("connections.json", &conns)

	// 查找是否已存在（按 id 匹配）
	id, _ := conn["id"].(string)
	found := false
	for i, c := range conns {
		if cid, _ := c["id"].(string); cid == id {
			conns[i] = conn
			found = true
			break
		}
	}
	if !found {
		conns = append(conns, conn)
	}

	if err := WriteJSON("connections.json", conns); err != nil {
		return nil, err
	}
	return nil, nil
}

func handleDeleteConnection(params json.RawMessage) (any, error) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	Disconnect(req.ID)
	RemoveBuffers(req.ID)

	var conns []map[string]interface{}
	ReadJSON("connections.json", &conns)

	filtered := make([]map[string]interface{}, 0, len(conns))
	for _, c := range conns {
		if cid, _ := c["id"].(string); cid != req.ID {
			filtered = append(filtered, c)
		}
	}

	if err := WriteJSON("connections.json", filtered); err != nil {
		return nil, err
	}
	return nil, nil
}

func handleReorderConnections(params json.RawMessage) (any, error) {
	var req struct {
		Items []struct {
			ID    string `json:"id"`
			Group string `json:"group"`
		} `json:"items"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	var conns []map[string]interface{}
	ReadJSON("connections.json", &conns)

	connMap := make(map[string]map[string]interface{})
	for _, c := range conns {
		if id, ok := c["id"].(string); ok {
			connMap[id] = c
		}
	}

	reordered := make([]map[string]interface{}, 0, len(conns))
	seen := make(map[string]bool)
	for _, item := range req.Items {
		if c, ok := connMap[item.ID]; ok {
			c["group"] = item.Group
			reordered = append(reordered, c)
			seen[item.ID] = true
		}
	}
	// 缺失的 ID 按原顺序追加
	for _, c := range conns {
		if id, _ := c["id"].(string); !seen[id] {
			reordered = append(reordered, c)
		}
	}

	return nil, WriteJSON("connections.json", reordered)
}

func handleSaveGroups(params json.RawMessage) (any, error) {
	var req struct {
		Groups []string `json:"groups"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	if req.Groups == nil {
		req.Groups = []string{}
	}
	return nil, WriteJSON("groups.json", req.Groups)
}

func handleGetGroups(_ json.RawMessage) (any, error) {
	var groups []string
	if err := ReadJSON("groups.json", &groups); err != nil {
		groups = []string{}
	}
	return groups, nil
}

func handleGetGroupMeta(_ json.RawMessage) (any, error) {
	var meta map[string]string
	if err := ReadJSON("group_meta.json", &meta); err != nil {
		meta = map[string]string{}
	}
	return meta, nil
}

func handleSaveGroupMeta(params json.RawMessage) (any, error) {
	var req struct {
		GroupMeta map[string]string `json:"group_meta"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	if req.GroupMeta == nil {
		req.GroupMeta = map[string]string{}
	}
	return nil, WriteJSON("group_meta.json", req.GroupMeta)
}

// ========== 设置 ==========

func handleGetSettings(params json.RawMessage) (any, error) {
	var settings map[string]interface{}
	if err := ReadJSON("settings.json", &settings); err != nil {
		settings = map[string]interface{}{}
	}
	return settings, nil
}

func handleSaveSettings(params json.RawMessage) (any, error) {
	var settings map[string]interface{}
	if err := json.Unmarshal(params, &settings); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	if err := WriteJSON("settings.json", settings); err != nil {
		return nil, err
	}
	return nil, nil
}

// ========== 基础设施 ==========

func handlePing(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
	}
	// params 可能为空（平台健康检查）或包含 conn_id（Redis 连接心跳）
	json.Unmarshal(params, &req)
	if req.ConnID == "" {
		// 平台健康检查，直接返回 pong
		return "pong", nil
	}
	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}
	if err := conn.Ping(); err != nil {
		return nil, err
	}
	return "pong", nil
}

func handleSaveSession(params json.RawMessage) (any, error) {
	var session map[string]interface{}
	if err := json.Unmarshal(params, &session); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	if err := WriteJSON("session.json", session); err != nil {
		return nil, err
	}
	return nil, nil
}

func handleGetSession(_ json.RawMessage) (any, error) {
	var session map[string]interface{}
	if err := ReadJSON("session.json", &session); err != nil {
		session = map[string]interface{}{}
	}
	return session, nil
}

// ========== 哨兵信息 ==========

func handleGetSentinelInfo(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
	}
	if err := json.Unmarshal(params, &req); err != nil || req.ConnID == "" {
		return nil, fmt.Errorf("参数错误: conn_id 必填")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}
	if !conn.IsSentinel || conn.FailoverOpts == nil {
		return nil, fmt.Errorf("非哨兵模式连接")
	}

	// 创建临时 Sentinel 客户端获取信息
	sentinelClient := redis.NewSentinelClient(&redis.Options{
		Addr:         conn.FailoverOpts.SentinelAddrs[0],
		Password:     conn.FailoverOpts.SentinelPassword,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})
	defer sentinelClient.Close()

	ctx := conn.Ctx
	masterName := conn.FailoverOpts.MasterName
	result := map[string]interface{}{
		"master_name": masterName,
	}

	// 获取 Master 信息
	masterAddr, err := sentinelClient.GetMasterAddrByName(ctx, masterName).Result()
	if err == nil {
		result["master"] = map[string]interface{}{
			"host": masterAddr[0],
			"port": masterAddr[1],
		}
	}

	// 获取 Master 详细信息（通过 SENTINEL MASTER 命令）
	masterInfo, err := sentinelClient.Master(ctx, masterName).Result()
	if err == nil {
		result["master_info"] = masterInfo
	}

	// 获取 Slave 列表
	slaves, err := sentinelClient.Slaves(ctx, masterName).Result()
	if err == nil {
		result["slaves"] = slaves
	}

	// 获取 Sentinel 节点列表
	sentinels, err := sentinelClient.Sentinels(ctx, masterName).Result()
	if err == nil {
		result["sentinels"] = sentinels
	}

	return result, nil
}

// ========== 集群信息 ==========

func handleGetClusterInfo(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
	}
	if err := json.Unmarshal(params, &req); err != nil || req.ConnID == "" {
		return nil, fmt.Errorf("参数错误: conn_id 必填")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}
	if !conn.IsCluster {
		return nil, fmt.Errorf("非集群模式连接")
	}

	ctx := conn.Ctx
	result := map[string]interface{}{}

	// CLUSTER INFO
	clusterInfo, err := conn.ClusterClient.ClusterInfo(ctx).Result()
	if err == nil {
		result["cluster_info"] = clusterInfo
	}

	// CLUSTER NODES — 解析为结构化数据
	nodesRaw, err := conn.ClusterClient.ClusterNodes(ctx).Result()
	if err == nil {
		result["nodes_raw"] = nodesRaw
		result["nodes"] = parseClusterNodes(nodesRaw)
	}

	return result, nil
}

// parseClusterNodes 解析 CLUSTER NODES 输出为结构化数据
func parseClusterNodes(raw string) []map[string]interface{} {
	var nodes []map[string]interface{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 格式: <id> <ip:port@cport> <flags> <master> <ping-sent> <pong-recv> <config-epoch> <link-state> <slot> <slot> ...
		parts := strings.Fields(line)
		if len(parts) < 8 {
			continue
		}
		node := map[string]interface{}{
			"id":           parts[0],
			"addr":         parts[1],
			"flags":        parts[2],
			"master_id":    parts[3],
			"ping_sent":    parts[4],
			"pong_recv":    parts[5],
			"config_epoch": parts[6],
			"link_state":   parts[7],
		}
		// 提取 ip:port（去掉 @cport 部分）
		addrParts := strings.SplitN(parts[1], "@", 2)
		node["endpoint"] = addrParts[0]
		// Slot 范围
		if len(parts) > 8 {
			node["slots"] = strings.Join(parts[8:], " ")
		} else {
			node["slots"] = ""
		}
		nodes = append(nodes, node)
	}
	return nodes
}

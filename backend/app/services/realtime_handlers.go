package services

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"


	"github.com/go-redis/redis/v8"
)

// RegisterRealtimeHandlers 注册实时功能相关的 RPC 方法
func RegisterRealtimeHandlers(register func(string, RPCHandlerFunc)) {
	// 通用轮询
	register("poll", handlePoll)

	// Pub/Sub
	register("pubsub_start", handlePubSubStart)
	register("pubsub_stop", handlePubSubStop)
	register("pubsub_publish", handlePubSubPublish)

	// MONITOR
	register("monitor_start", handleMonitorStart)
	register("monitor_stop", handleMonitorStop)

	// 延迟诊断
	register("latency_test", handleLatencyTest)
}

// ========== 通用轮询 ==========

func handlePoll(params json.RawMessage) (any, error) {
	var req struct {
		ConnID  string `json:"conn_id"`
		Scene   string `json:"scene"`
		After   int64  `json:"after"`
		Timeout int    `json:"timeout"` // 长轮询超时（秒），0 表示立即返回
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	if req.ConnID == "" || req.Scene == "" {
		return nil, fmt.Errorf("conn_id 和 scene 必填")
	}

	buf := GetBuffer(req.ConnID, req.Scene, 1000)

	var events []PollEvent
	if req.Timeout > 0 {
		if req.Timeout > 30 {
			req.Timeout = 30
		}
		events = buf.WaitSince(req.After, time.Duration(req.Timeout)*time.Second)
	} else {
		events = buf.Since(req.After)
	}
	if events == nil {
		events = []PollEvent{}
	}
	return events, nil
}

// ========== Pub/Sub ==========

type pubSubSession struct {
	pubsub   *redis.PubSub
	client   *redis.Client
	cancel   context.CancelFunc
	channels []string
	patterns []string
	mu       sync.Mutex
}

var (
	pubsubSessions   = make(map[string]*pubSubSession)
	pubsubSessionsMu sync.Mutex
)

func handlePubSubStart(params json.RawMessage) (any, error) {
	var req struct {
		ConnID   string   `json:"conn_id"`
		Channels []string `json:"channels"` // 精确频道
		Patterns []string `json:"patterns"` // 模式匹配
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	// 停止已有的 session
	stopPubSubSession(req.ConnID)

	if len(req.Channels) == 0 && len(req.Patterns) == 0 {
		return nil, fmt.Errorf("至少指定一个频道或模式")
	}

	// 创建独立的 Redis 客户端用于订阅
	ctx, cancel := context.WithCancel(context.Background())
	var ps *redis.PubSub
	var subClient *redis.Client

	if conn.IsCluster {
		// 集群模式：直接使用 ClusterClient 订阅
		ps = conn.ClusterClient.Subscribe(ctx)
	} else {
		opts := conn.ClientOptions()
		subClient = redis.NewClient(opts)
		ps = subClient.Subscribe(ctx)
	}

	// 订阅频道和模式
	if len(req.Channels) > 0 {
		if err := ps.Subscribe(ctx, req.Channels...); err != nil {
			cancel()
			if subClient != nil {
				subClient.Close()
			}
			return nil, err
		}
	}
	if len(req.Patterns) > 0 {
		if err := ps.PSubscribe(ctx, req.Patterns...); err != nil {
			cancel()
			if subClient != nil {
				subClient.Close()
			}
			return nil, err
		}
	}

	session := &pubSubSession{
		pubsub:   ps,
		client:   subClient,
		cancel:   cancel,
		channels: req.Channels,
		patterns: req.Patterns,
	}

	pubsubSessionsMu.Lock()
	pubsubSessions[req.ConnID] = session
	pubsubSessionsMu.Unlock()

	buf := GetBuffer(req.ConnID, "pubsub", 2000)
	buf.Clear()

	// 启动消息接收协程
	go func() {
		ch := ps.Channel()
		for msg := range ch {
			buf.Push("message", map[string]interface{}{
				"channel": msg.Channel,
				"pattern": msg.Pattern,
				"payload": msg.Payload,
			})
		}
	}()

	AddOpLog(req.ConnID, "SUBSCRIBE", "", fmt.Sprintf("channels=%v patterns=%v", req.Channels, req.Patterns))
	return map[string]interface{}{
		"channels": req.Channels,
		"patterns": req.Patterns,
	}, nil
}

func handlePubSubStop(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	stopPubSubSession(req.ConnID)
	return nil, nil
}

func stopPubSubSession(connID string) {
	pubsubSessionsMu.Lock()
	session, ok := pubsubSessions[connID]
	if ok {
		delete(pubsubSessions, connID)
	}
	pubsubSessionsMu.Unlock()

	if ok && session != nil {
		session.cancel()
		session.pubsub.Close()
		if session.client != nil {
			session.client.Close()
		}
	}
}

func handlePubSubPublish(params json.RawMessage) (any, error) {
	var req struct {
		ConnID  string `json:"conn_id"`
		Channel string `json:"channel"`
		Message string `json:"message"`
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

	receivers, err := conn.Cmd().Publish(ctx, req.Channel, req.Message).Result()
	if err != nil {
		return nil, err
	}

	AddOpLog(req.ConnID, "PUBLISH", req.Channel, req.Message)
	return map[string]interface{}{
		"receivers": receivers,
	}, nil
}

// ========== MONITOR ==========

type monitorSession struct {
	conn   net.Conn
	cancel context.CancelFunc
}

var (
	monitorSessions   = make(map[string]*monitorSession)
	monitorSessionsMu sync.Mutex
)

func handleMonitorStart(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	redisConn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	// 停止已有的 monitor session
	stopMonitorSession(req.ConnID)

	// 集群模式不支持 MONITOR
	if redisConn.IsCluster {
		return nil, fmt.Errorf("集群模式不支持 MONITOR")
	}

	// 创建原始 TCP 连接用于 MONITOR
	opts := redisConn.ClientOptions()
	rawConn, err := net.DialTimeout(opts.Network, opts.Addr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %v", err)
	}

	// 认证
	if opts.Password != "" {
		var authCmd string
		if opts.Username != "" {
			authCmd = fmt.Sprintf("AUTH %s %s\r\n", opts.Username, opts.Password)
		} else {
			authCmd = fmt.Sprintf("AUTH %s\r\n", opts.Password)
		}
		rawConn.Write([]byte(authCmd))
		authBuf := make([]byte, 256)
		rawConn.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, _ := rawConn.Read(authBuf)
		resp := string(authBuf[:n])
		if !strings.HasPrefix(resp, "+OK") {
			rawConn.Close()
			return nil, fmt.Errorf("认证失败")
		}
	}

	// 选择 DB
	if opts.DB > 0 {
		selectCmd := fmt.Sprintf("SELECT %d\r\n", opts.DB)
		rawConn.Write([]byte(selectCmd))
		selBuf := make([]byte, 256)
		rawConn.SetReadDeadline(time.Now().Add(5 * time.Second))
		rawConn.Read(selBuf)
	}

	// 发送 MONITOR 命令
	rawConn.Write([]byte("MONITOR\r\n"))
	monBuf := make([]byte, 256)
	rawConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := rawConn.Read(monBuf)
	if err != nil || !strings.HasPrefix(string(monBuf[:n]), "+OK") {
		rawConn.Close()
		return nil, fmt.Errorf("MONITOR 启动失败")
	}

	ctx, cancel := context.WithCancel(context.Background())
	session := &monitorSession{
		conn:   rawConn,
		cancel: cancel,
	}

	monitorSessionsMu.Lock()
	monitorSessions[req.ConnID] = session
	monitorSessionsMu.Unlock()

	buf := GetBuffer(req.ConnID, "monitor", 2000)
	buf.Clear()

	// 启动读取协程
	// 注意：不能用 bufio.Scanner，因为 Scanner 遇到超时错误后进入永久错误状态，
	// 后续 Scan() 永远返回 false。改用 bufio.Reader，超时后可继续读取。
	go func() {
		reader := bufio.NewReaderSize(rawConn, 64*1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				rawConn.SetReadDeadline(time.Now().Add(2 * time.Second))
				line, err := reader.ReadString('\n')
				if err != nil {
					if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
						continue // 超时是正常的，继续等待
					}
					return // 真正的错误，停止读取
				}
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				// MONITOR 输出格式: 1234567890.123456 [0 127.0.0.1:6379] "COMMAND" "arg1" "arg2"
				parsed := parseMonitorLine(line)
				buf.Push("command", parsed)
			}
		}
	}()

	AddOpLog(req.ConnID, "MONITOR", "", "started")
	return nil, nil
}

func handleMonitorStop(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}
	stopMonitorSession(req.ConnID)
	return nil, nil
}

func stopMonitorSession(connID string) {
	monitorSessionsMu.Lock()
	session, ok := monitorSessions[connID]
	if ok {
		delete(monitorSessions, connID)
	}
	monitorSessionsMu.Unlock()

	if ok && session != nil {
		session.cancel()
		session.conn.Close()
	}
}

func parseMonitorLine(line string) map[string]interface{} {
	result := map[string]interface{}{
		"raw": line,
	}

	// 格式: 1234567890.123456 [0 127.0.0.1:6379] "COMMAND" "arg1"
	// 提取时间戳
	spaceIdx := strings.Index(line, " ")
	if spaceIdx > 0 {
		result["timestamp"] = line[:spaceIdx]
	}

	// 提取 [db addr]
	bracketStart := strings.Index(line, "[")
	bracketEnd := strings.Index(line, "]")
	if bracketStart >= 0 && bracketEnd > bracketStart {
		dbAddr := line[bracketStart+1 : bracketEnd]
		parts := strings.SplitN(dbAddr, " ", 2)
		if len(parts) >= 1 {
			result["db"] = parts[0]
		}
		if len(parts) >= 2 {
			result["addr"] = parts[1]
		}
	}

	// 提取命令和参数
	if bracketEnd > 0 && bracketEnd+2 < len(line) {
		cmdPart := strings.TrimSpace(line[bracketEnd+1:])
		result["command"] = cmdPart
	}

	return result
}

// ========== 延迟诊断 ==========

func handleLatencyTest(params json.RawMessage) (any, error) {
	var req struct {
		ConnID string `json:"conn_id"`
		Count  int    `json:"count"` // PING 次数
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("参数错误")
	}

	conn, ok := GetConn(req.ConnID)
	if !ok {
		return nil, fmt.Errorf("连接不存在")
	}

	if req.Count <= 0 || req.Count > 100 {
		req.Count = 20
	}

	samples := make([]float64, 0, req.Count)
	var totalMs float64

	for i := 0; i < req.Count; i++ {
		ctx, cancel := context.WithTimeout(conn.Ctx, 5*time.Second)
		start := time.Now()
		err := conn.Cmd().Ping(ctx).Err()
		elapsed := time.Since(start)
		cancel()

		if err != nil {
			continue
		}
		ms := float64(elapsed.Microseconds()) / 1000.0
		samples = append(samples, ms)
		totalMs += ms
	}

	if len(samples) == 0 {
		return nil, fmt.Errorf("所有 PING 均失败")
	}

	// 计算统计值
	minMs := samples[0]
	maxMs := samples[0]
	for _, s := range samples[1:] {
		if s < minMs {
			minMs = s
		}
		if s > maxMs {
			maxMs = s
		}
	}
	avgMs := totalMs / float64(len(samples))

	// LATENCY LATEST
	ctx, cancel := context.WithTimeout(conn.Ctx, 5*time.Second)
	defer cancel()
	latencyLatest, _ := conn.Do(ctx, "LATENCY", "LATEST").Result()

	return map[string]interface{}{
		"samples":  samples,
		"count":    len(samples),
		"min_ms":   minMs,
		"max_ms":   maxMs,
		"avg_ms":   avgMs,
		"latency":  latencyLatest,
	}, nil
}

// CleanupRealtimeSessions 清理指定连接的所有实时会话
func CleanupRealtimeSessions(connID string) {
	stopPubSubSession(connID)
	stopMonitorSession(connID)
}

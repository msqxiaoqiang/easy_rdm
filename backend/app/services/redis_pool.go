package services

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// ConnConfig Redis 连接配置
type ConnConfig struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	Username string `json:"username"`
	DB       int    `json:"db"`
	// TCP / UNIX
	ConnType   string `json:"conn_type"`
	UnixSocket string `json:"unix_socket"`
	// 超时配置（秒）
	ConnTimeout time.Duration `json:"conn_timeout"`
	ExecTimeout time.Duration `json:"exec_timeout"`
	// TLS/SSL
	UseTLS        bool   `json:"use_tls"`
	TLSCertFile   string `json:"tls_cert_file"`
	TLSKeyFile    string `json:"tls_key_file"`
	TLSCAFile     string `json:"tls_ca_file"`
	TLSSkipVerify bool   `json:"tls_skip_verify"`
	// SSH 隧道
	UseSSH        bool   `json:"use_ssh"`
	SSHHost       string `json:"ssh_host"`
	SSHPort       int    `json:"ssh_port"`
	SSHUsername   string `json:"ssh_username"`
	SSHPassword   string `json:"ssh_password"`
	SSHPrivateKey string `json:"ssh_private_key"`
	SSHPassphrase string `json:"ssh_passphrase"`
	// 网络代理
	UseProxy      bool   `json:"use_proxy"`
	ProxyType     string `json:"proxy_type"`     // http | https | socks5 | socks5h
	ProxyHost     string `json:"proxy_host"`
	ProxyPort     int    `json:"proxy_port"`
	ProxyUsername string `json:"proxy_username"`
	ProxyPassword string `json:"proxy_password"`
	// 哨兵模式
	UseSentinel        bool   `json:"use_sentinel"`
	SentinelAddrs      string `json:"sentinel_addrs"`       // 逗号分隔的哨兵地址列表
	SentinelMasterName string `json:"sentinel_master_name"`
	SentinelPassword   string `json:"sentinel_password"`    // 哨兵节点密码
	// 集群模式
	UseCluster   bool   `json:"use_cluster"`
	ClusterAddrs string `json:"cluster_addrs"` // 逗号分隔的集群节点地址列表
}

// RedisConn 单个 Redis 连接实例
type RedisConn struct {
	Config        *ConnConfig
	Client        *redis.Client         // 普通/哨兵模式
	ClusterClient *redis.ClusterClient  // 集群模式
	Ctx           context.Context
	Cancel        context.CancelFunc
	IsSentinel    bool                   // 是否为哨兵模式连接
	IsCluster     bool                   // 是否为集群模式连接
	FailoverOpts  *redis.FailoverOptions // 哨兵模式选项（用于 SelectDB 重建）
	ClusterOpts   *redis.ClusterOptions  // 集群模式选项
	mu            sync.RWMutex           // 保护 Client 字段，防止 SelectDB 替换时的竞态
}

// Cmd 返回通用命令接口（普通/哨兵/集群模式通用）
func (rc *RedisConn) Cmd() redis.Cmdable {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	if rc.IsCluster {
		return rc.ClusterClient
	}
	return rc.Client
}

var (
	connections = make(map[string]*RedisConn)
	connMu      sync.RWMutex

	// 哨兵故障转移监听器
	sentinelWatchers   = make(map[string]context.CancelFunc)
	sentinelWatchersMu sync.Mutex
)

// buildTLSConfig 构建 TLS 配置（Connect 和哨兵模式共用）
func buildTLSConfig(cfg *ConnConfig) (*tls.Config, error) {
	if !cfg.UseTLS {
		return nil, nil
	}
	tlsCfg := &tls.Config{
		InsecureSkipVerify: cfg.TLSSkipVerify,
	}
	if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("加载 TLS 证书失败: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}
	if cfg.TLSCAFile != "" {
		caCert, err := os.ReadFile(cfg.TLSCAFile)
		if err != nil {
			return nil, fmt.Errorf("加载 CA 证书失败: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("CA 证书格式无效")
		}
		tlsCfg.RootCAs = pool
	}
	return tlsCfg, nil
}

// parseSentinelAddrs 解析逗号分隔的哨兵地址列表
func parseSentinelAddrs(addrs string) []string {
	parts := strings.Split(addrs, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// Connect 建立 Redis 连接
func Connect(cfg *ConnConfig) (*RedisConn, error) {
	connMu.Lock()
	defer connMu.Unlock()

	// 如果已存在，先断开
	if existing, ok := connections[cfg.ID]; ok {
		existing.Close()
	}

	// TLS 配置（普通模式和哨兵模式共用）
	tlsCfg, err := buildTLSConfig(cfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	// ========== 哨兵模式 ==========
	if cfg.UseSentinel {
		sentinelAddrs := parseSentinelAddrs(cfg.SentinelAddrs)
		if len(sentinelAddrs) == 0 {
			cancel()
			return nil, fmt.Errorf("哨兵地址列表不能为空")
		}
		if cfg.SentinelMasterName == "" {
			cancel()
			return nil, fmt.Errorf("哨兵 Master 名称不能为空")
		}

		foOpts := &redis.FailoverOptions{
			MasterName:       cfg.SentinelMasterName,
			SentinelAddrs:    sentinelAddrs,
			SentinelPassword: cfg.SentinelPassword,
			DB:               cfg.DB,
			DialTimeout:      cfg.ConnTimeout,
			ReadTimeout:      cfg.ExecTimeout,
			WriteTimeout:     cfg.ExecTimeout,
			PoolSize:         5,
		}
		if cfg.Password != "" {
			foOpts.Password = cfg.Password
		}
		if cfg.Username != "" {
			foOpts.Username = cfg.Username
		}
		if tlsCfg != nil {
			foOpts.TLSConfig = tlsCfg
		}

		client := redis.NewFailoverClient(foOpts)
		if err := client.Ping(ctx).Err(); err != nil {
			cancel()
			client.Close()
			return nil, fmt.Errorf("哨兵连接失败: %w", err)
		}

		conn := &RedisConn{
			Config:       cfg,
			Client:       client,
			Ctx:          ctx,
			Cancel:       cancel,
			IsSentinel:   true,
			FailoverOpts: foOpts,
		}
		connections[cfg.ID] = conn

		// 启动哨兵故障转移监听
		startSentinelWatcher(cfg.ID, foOpts)

		return conn, nil
	}

	// ========== 集群模式 ==========
	if cfg.UseCluster {
		clusterAddrs := parseSentinelAddrs(cfg.ClusterAddrs) // 复用逗号分隔解析
		if len(clusterAddrs) == 0 {
			cancel()
			return nil, fmt.Errorf("集群节点地址列表不能为空")
		}

		clusterOpts := &redis.ClusterOptions{
			Addrs:        clusterAddrs,
			DialTimeout:  cfg.ConnTimeout,
			ReadTimeout:  cfg.ExecTimeout,
			WriteTimeout: cfg.ExecTimeout,
			PoolSize:     5,
		}
		if cfg.Password != "" {
			clusterOpts.Password = cfg.Password
		}
		if cfg.Username != "" {
			clusterOpts.Username = cfg.Username
		}
		if tlsCfg != nil {
			clusterOpts.TLSConfig = tlsCfg
		}

		clusterClient := redis.NewClusterClient(clusterOpts)
		if err := clusterClient.Ping(ctx).Err(); err != nil {
			cancel()
			clusterClient.Close()
			return nil, fmt.Errorf("集群连接失败: %w", err)
		}

		conn := &RedisConn{
			Config:        cfg,
			ClusterClient: clusterClient,
			Ctx:           ctx,
			Cancel:        cancel,
			IsCluster:     true,
			ClusterOpts:   clusterOpts,
		}
		connections[cfg.ID] = conn
		return conn, nil
	}

	// ========== 普通模式 ==========
	opts := &redis.Options{
		DB:           cfg.DB,
		DialTimeout:  cfg.ConnTimeout,
		ReadTimeout:  cfg.ExecTimeout,
		WriteTimeout: cfg.ExecTimeout,
		PoolSize:     5,
	}

	if cfg.ConnType == "unix" {
		opts.Network = "unix"
		opts.Addr = cfg.UnixSocket
	} else {
		opts.Addr = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	}

	if cfg.Password != "" {
		opts.Password = cfg.Password
	}
	if cfg.Username != "" {
		opts.Username = cfg.Username
	}
	if tlsCfg != nil {
		opts.TLSConfig = tlsCfg
	}

	// SSH 隧道：通过本地转发连接远程 Redis
	if cfg.UseSSH && cfg.ConnType != "unix" {
		sshCfg := &SSHConfig{
			Host:       cfg.SSHHost,
			Port:       cfg.SSHPort,
			Username:   cfg.SSHUsername,
			Password:   cfg.SSHPassword,
			PrivateKey: cfg.SSHPrivateKey,
			Passphrase: cfg.SSHPassphrase,
		}
		localAddr, err := CreateTunnel(cfg.ID, sshCfg, opts.Addr)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("SSH 隧道创建失败: %w", err)
		}
		opts.Addr = localAddr
	} else if cfg.UseProxy && cfg.ConnType != "unix" {
		// 网络代理（与 SSH 隧道互斥，SSH 优先）
		proxyCfg := &ProxyConfig{
			Type:     cfg.ProxyType,
			Host:     cfg.ProxyHost,
			Port:     cfg.ProxyPort,
			Username: cfg.ProxyUsername,
			Password: cfg.ProxyPassword,
		}
		dialFn, err := ProxyDialer(proxyCfg, cfg.ConnTimeout*time.Second)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("代理配置失败: %w", err)
		}
		opts.Dialer = dialFn
	}

	client := redis.NewClient(opts)

	// 测试连接
	if err := client.Ping(ctx).Err(); err != nil {
		cancel()
		client.Close()
		if cfg.UseSSH {
			CloseTunnel(cfg.ID)
		}
		return nil, fmt.Errorf("连接失败: %w", err)
	}

	conn := &RedisConn{
		Config: cfg,
		Client: client,
		Ctx:    ctx,
		Cancel: cancel,
	}
	connections[cfg.ID] = conn
	return conn, nil
}

// Disconnect 断开 Redis 连接
func Disconnect(id string) {
	stopSentinelWatcher(id)
	connMu.Lock()
	defer connMu.Unlock()
	if conn, ok := connections[id]; ok {
		conn.Close()
		delete(connections, id)
	}
	CloseTunnel(id)
}

// DisconnectAll 断开所有活跃的 Redis 连接和 SSH 隧道
func DisconnectAll() {
	connMu.Lock()
	ids := make([]string, 0, len(connections))
	for id := range connections {
		ids = append(ids, id)
	}
	connMu.Unlock()
	for _, id := range ids {
		Disconnect(id)
	}
}

// GetConn 获取已建立的连接
func GetConn(id string) (*RedisConn, bool) {
	connMu.RLock()
	defer connMu.RUnlock()
	conn, ok := connections[id]
	return conn, ok
}

// Close 关闭连接
func (rc *RedisConn) Close() {
	rc.Cancel()
	if rc.IsCluster {
		rc.ClusterClient.Close()
	} else {
		rc.Client.Close()
	}
}

// Ping 测试连接是否存活
func (rc *RedisConn) Ping() error {
	ctx, cancel := context.WithTimeout(rc.Ctx, 5*time.Second)
	defer cancel()
	return rc.Cmd().Ping(ctx).Err()
}

// SelectDB 切换数据库
// go-redis 的连接池中每条连接独立，pipeline SELECT 只影响单条连接，
// 后续命令可能分配到其他仍在旧 DB 的连接上。
// 因此必须用目标 DB 重建 Client。
// 集群模式不支持切换 DB（只能使用 DB 0）。
func (rc *RedisConn) SelectDB(db int) error {
	rc.mu.RLock()
	currentDB := rc.Config.DB
	rc.mu.RUnlock()

	if db == currentDB {
		return nil
	}
	if rc.IsCluster {
		return fmt.Errorf("集群模式不支持切换数据库")
	}

	// 在锁外构建新 Client（含网络 Ping），避免持锁阻塞其他请求
	rc.mu.RLock()
	var newClient *redis.Client
	if rc.IsSentinel && rc.FailoverOpts != nil {
		foOpts := *rc.FailoverOpts
		foOpts.DB = db
		newClient = redis.NewFailoverClient(&foOpts)
	} else {
		opts := rc.Client.Options()
		opts.DB = db
		newClient = redis.NewClient(opts)
	}
	rc.mu.RUnlock()

	if err := newClient.Ping(rc.Ctx).Err(); err != nil {
		newClient.Close()
		return fmt.Errorf("切换数据库失败: %w", err)
	}

	// 写锁替换 Client
	rc.mu.Lock()
	oldClient := rc.Client
	rc.Client = newClient
	rc.Config.DB = db
	if rc.FailoverOpts != nil {
		rc.FailoverOpts.DB = db
	}
	rc.mu.Unlock()

	oldClient.Close()
	return nil
}

// GetServerInfo 获取 Redis 服务器信息
func (rc *RedisConn) GetServerInfo() (string, error) {
	return rc.Cmd().Info(rc.Ctx).Result()
}

// GetClusterInfo 获取集群信息
func (rc *RedisConn) GetClusterInfo() (string, error) {
	if !rc.IsCluster {
		return "", fmt.Errorf("非集群模式连接")
	}
	return rc.ClusterClient.ClusterInfo(rc.Ctx).Result()
}

// GetClusterNodes 获取集群节点列表
func (rc *RedisConn) GetClusterNodes() (string, error) {
	if !rc.IsCluster {
		return "", fmt.Errorf("非集群模式连接")
	}
	return rc.ClusterClient.ClusterNodes(rc.Ctx).Result()
}

// Watch 执行乐观锁事务（普通/哨兵/集群模式通用）
func (rc *RedisConn) Watch(ctx context.Context, fn func(*redis.Tx) error, keys ...string) error {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	if rc.IsCluster {
		return rc.ClusterClient.Watch(ctx, fn, keys...)
	}
	return rc.Client.Watch(ctx, fn, keys...)
}

// Do 执行任意 Redis 命令（Do 不在 Cmdable 接口中）
func (rc *RedisConn) Do(ctx context.Context, args ...interface{}) *redis.Cmd {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	if rc.IsCluster {
		return rc.ClusterClient.Do(ctx, args...)
	}
	return rc.Client.Do(ctx, args...)
}

// ClientOptions 获取普通/哨兵模式的连接选项（集群模式返回 nil）

func (rc *RedisConn) ClientOptions() *redis.Options {
	if rc.IsCluster || rc.Client == nil {
		return nil
	}
	return rc.Client.Options()
}

// ========== 哨兵故障转移监听 ==========

// startSentinelWatcher 启动哨兵故障转移监听
// 订阅 sentinel 的 +switch-master 频道，检测到故障转移时推送事件到轮询缓冲区
func startSentinelWatcher(connID string, foOpts *redis.FailoverOptions) {
	stopSentinelWatcher(connID)

	if len(foOpts.SentinelAddrs) == 0 {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	sentinelWatchersMu.Lock()
	sentinelWatchers[connID] = cancel
	sentinelWatchersMu.Unlock()

	go func() {
		// 创建 sentinel 客户端用于订阅
		sentinelClient := redis.NewSentinelClient(&redis.Options{
			Addr:         foOpts.SentinelAddrs[0],
			Password:     foOpts.SentinelPassword,
			DialTimeout:  10 * time.Second,
			ReadTimeout:  0, // 订阅模式不设读超时
			WriteTimeout: 10 * time.Second,
		})
		defer sentinelClient.Close()

		pubsub := sentinelClient.Subscribe(ctx, "+switch-master")
		defer pubsub.Close()

		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				// +switch-master 消息格式: "mymaster old-ip old-port new-ip new-port"
				parts := strings.SplitN(msg.Payload, " ", 5)
				event := map[string]interface{}{
					"raw": msg.Payload,
				}
				if len(parts) >= 5 {
					event["master_name"] = parts[0]
					event["old_addr"] = parts[1] + ":" + parts[2]
					event["new_addr"] = parts[3] + ":" + parts[4]
				}
				buf := GetBuffer(connID, "sentinel", 100)
				buf.Push("switch-master", event)
			}
		}
	}()
}

// stopSentinelWatcher 停止哨兵故障转移监听
func stopSentinelWatcher(connID string) {
	sentinelWatchersMu.Lock()
	cancel, ok := sentinelWatchers[connID]
	if ok {
		delete(sentinelWatchers, connID)
	}
	sentinelWatchersMu.Unlock()
	if ok && cancel != nil {
		cancel()
	}
}

//go:build integration

package services

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isDockerNetwork 判断测试是否运行在 Docker 网络内
// 设置环境变量 REDIS_TEST_NETWORK=docker 时使用 Docker 服务名+内部端口
func isDockerNetwork() bool {
	return os.Getenv("REDIS_TEST_NETWORK") == "docker"
}

// testAddr 根据运行环境返回不同的地址
// Docker 模式：使用服务名 + 容器内部端口（测试进程在同一 Docker 网络）
// Host 模式：使用 localhost + 映射端口（测试进程在宿主机）
func testAddr(dockerHost string, dockerPort int, hostHost string, hostPort int) (string, int) {
	if isDockerNetwork() {
		return dockerHost, dockerPort
	}
	return hostHost, hostPort
}

// testDataPath 返回 tests/ 下的文件路径
// rel 是相对于 tests/ 的路径，如 "certs/ca.crt"
func testDataPath(rel string) string {
	// Docker 网络模式下，tests/ 目录挂载到 /testdata
	if isDockerNetwork() {
		return filepath.Join("/testdata", rel)
	}
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(filename), "..", "..", "..")
	return filepath.Join(root, "tests", rel)
}

func TestIntegration(t *testing.T) {

	// ===== TCP 连接 =====

	t.Run("TCPConnect", func(t *testing.T) {
		host, port := testAddr("redis", 6379, "localhost", 16379)
		cfg := &ConnConfig{
			ID:          "integ-tcp",
			Host:        host,
			Port:        port,
			Password:    "testpass",
			ConnTimeout: 10 * time.Second,
			ExecTimeout: 10 * time.Second,
		}
		conn, err := Connect(cfg)
		require.NoError(t, err)
		defer Disconnect(cfg.ID)

		assert.NotNil(t, conn)
		assert.NoError(t, conn.Ping())
	})

	t.Run("TCPConnectNoPass", func(t *testing.T) {
		host, port := testAddr("redis-nopass", 6379, "localhost", 16380)
		cfg := &ConnConfig{
			ID:          "integ-tcp-nopass",
			Host:        host,
			Port:        port,
			ConnTimeout: 10 * time.Second,
			ExecTimeout: 10 * time.Second,
		}
		conn, err := Connect(cfg)
		require.NoError(t, err)
		defer Disconnect(cfg.ID)

		assert.NotNil(t, conn)
		assert.NoError(t, conn.Ping())
	})

	t.Run("TCPConnectWrongPass", func(t *testing.T) {
		host, port := testAddr("redis", 6379, "localhost", 16379)
		cfg := &ConnConfig{
			ID:          "integ-tcp-wrongpass",
			Host:        host,
			Port:        port,
			Password:    "wrongpassword",
			ConnTimeout: 10 * time.Second,
			ExecTimeout: 10 * time.Second,
		}
		_, err := Connect(cfg)
		assert.Error(t, err)
		Disconnect(cfg.ID)
	})

	// ===== SSH 隧道 =====

	t.Run("SSHTunnelConnect", func(t *testing.T) {
		sshHost, sshPort := testAddr("ssh-server", 2222, "localhost", 2222)
		cfg := &ConnConfig{
			ID:          "integ-ssh-pass",
			Host:        "redis-ssh",
			Port:        6379,
			UseSSH:      true,
			SSHHost:     sshHost,
			SSHPort:     sshPort,
			SSHUsername: "testuser",
			SSHPassword: "testpass",
			ConnTimeout: 10 * time.Second,
			ExecTimeout: 10 * time.Second,
		}
		conn, err := Connect(cfg)
		require.NoError(t, err)
		defer Disconnect(cfg.ID)

		assert.NotNil(t, conn)
		assert.NoError(t, conn.Ping())
	})

	t.Run("SSHTunnelConnectKey", func(t *testing.T) {
		sshHost, sshPort := testAddr("ssh-server", 2222, "localhost", 2222)
		cfg := &ConnConfig{
			ID:            "integ-ssh-key",
			Host:          "redis-ssh",
			Port:          6379,
			UseSSH:        true,
			SSHHost:       sshHost,
			SSHPort:       sshPort,
			SSHUsername:   "testuser",
			SSHPrivateKey: testDataPath("certs/ssh_test_key"),
			ConnTimeout:   10 * time.Second,
			ExecTimeout:   10 * time.Second,
		}
		conn, err := Connect(cfg)
		require.NoError(t, err)
		defer Disconnect(cfg.ID)

		assert.NotNil(t, conn)
		assert.NoError(t, conn.Ping())
	})

	// ===== TLS 加密 =====

	t.Run("TLSConnect", func(t *testing.T) {
		host, port := testAddr("redis-tls", 6379, "localhost", 16381)
		cfg := &ConnConfig{
			ID:          "integ-tls",
			Host:        host,
			Port:        port,
			UseTLS:      true,
			TLSCAFile:   testDataPath("certs/ca.crt"),
			ConnTimeout: 10 * time.Second,
			ExecTimeout: 10 * time.Second,
		}
		conn, err := Connect(cfg)
		require.NoError(t, err)
		defer Disconnect(cfg.ID)

		assert.NotNil(t, conn)
		assert.NoError(t, conn.Ping())
	})

	t.Run("TLSConnectSkipVerify", func(t *testing.T) {
		host, port := testAddr("redis-tls", 6379, "localhost", 16381)
		cfg := &ConnConfig{
			ID:            "integ-tls-skip",
			Host:          host,
			Port:          port,
			UseTLS:        true,
			TLSSkipVerify: true,
			ConnTimeout:   10 * time.Second,
			ExecTimeout:   10 * time.Second,
		}
		conn, err := Connect(cfg)
		require.NoError(t, err)
		defer Disconnect(cfg.ID)

		assert.NotNil(t, conn)
		assert.NoError(t, conn.Ping())
	})

	// ===== 哨兵模式 =====

	t.Run("SentinelConnect", func(t *testing.T) {
		sentinelHost, _ := testAddr("sentinel-1", 26379, "localhost", 26379)
		sentinelHost2, _ := testAddr("sentinel-2", 26380, "localhost", 26380)
		sentinelHost3, _ := testAddr("sentinel-3", 26381, "localhost", 26381)
		cfg := &ConnConfig{
			ID:                 "integ-sentinel",
			Password:           "masterpass",
			UseSentinel:        true,
			SentinelAddrs:      sentinelHost + ":26379," + sentinelHost2 + ":26380," + sentinelHost3 + ":26381",
			SentinelMasterName: "mymaster",
			ConnTimeout:        10 * time.Second,
			ExecTimeout:        10 * time.Second,
		}
		conn, err := Connect(cfg)
		if err != nil {
			t.Skipf("哨兵连接失败（可能是 Docker Desktop 网络限制）: %v", err)
			return
		}
		defer Disconnect(cfg.ID)

		assert.NotNil(t, conn)
		assert.True(t, conn.IsSentinel)
		assert.NoError(t, conn.Ping())
	})

	t.Run("SentinelSelectDB", func(t *testing.T) {
		sentinelHost, _ := testAddr("sentinel-1", 26379, "localhost", 26379)
		sentinelHost2, _ := testAddr("sentinel-2", 26380, "localhost", 26380)
		sentinelHost3, _ := testAddr("sentinel-3", 26381, "localhost", 26381)
		cfg := &ConnConfig{
			ID:                 "integ-sentinel-db",
			Password:           "masterpass",
			UseSentinel:        true,
			SentinelAddrs:      sentinelHost + ":26379," + sentinelHost2 + ":26380," + sentinelHost3 + ":26381",
			SentinelMasterName: "mymaster",
			ConnTimeout:        10 * time.Second,
			ExecTimeout:        10 * time.Second,
		}
		conn, err := Connect(cfg)
		if err != nil {
			t.Skipf("哨兵连接失败（可能是 Docker Desktop 网络限制）: %v", err)
			return
		}
		defer Disconnect(cfg.ID)

		err = conn.SelectDB(1)
		assert.NoError(t, err)
		assert.Equal(t, 1, conn.Config.DB)
	})

	// ===== 集群模式 =====

	t.Run("ClusterConnect", func(t *testing.T) {
		h1, _ := testAddr("redis-node-1", 6379, "localhost", 7101)
		h2, _ := testAddr("redis-node-2", 6379, "localhost", 7102)
		h3, _ := testAddr("redis-node-3", 6379, "localhost", 7103)
		var addrs string
		if isDockerNetwork() {
			addrs = h1 + ":6379," + h2 + ":6379," + h3 + ":6379"
		} else {
			addrs = h1 + ":7101," + h2 + ":7102," + h3 + ":7103"
		}
		cfg := &ConnConfig{
			ID:           "integ-cluster",
			UseCluster:   true,
			ClusterAddrs: addrs,
			ConnTimeout:  10 * time.Second,
			ExecTimeout:  10 * time.Second,
		}
		conn, err := Connect(cfg)
		if err != nil {
			t.Skipf("集群连接失败（可能是网络环境限制）: %v", err)
			return
		}
		defer Disconnect(cfg.ID)

		assert.NotNil(t, conn)
		assert.True(t, conn.IsCluster)
	})

	t.Run("ClusterNoSelectDB", func(t *testing.T) {
		h1, _ := testAddr("redis-node-1", 6379, "localhost", 7101)
		h2, _ := testAddr("redis-node-2", 6379, "localhost", 7102)
		h3, _ := testAddr("redis-node-3", 6379, "localhost", 7103)
		var addrs string
		if isDockerNetwork() {
			addrs = h1 + ":6379," + h2 + ":6379," + h3 + ":6379"
		} else {
			addrs = h1 + ":7101," + h2 + ":7102," + h3 + ":7103"
		}
		cfg := &ConnConfig{
			ID:           "integ-cluster-selectdb",
			UseCluster:   true,
			ClusterAddrs: addrs,
			ConnTimeout:  10 * time.Second,
			ExecTimeout:  10 * time.Second,
		}
		conn, err := Connect(cfg)
		if err != nil {
			t.Skipf("集群连接失败（可能是网络环境限制）: %v", err)
			return
		}
		defer Disconnect(cfg.ID)

		err = conn.SelectDB(1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "集群模式不支持切换数据库")
	})

	// ===== SOCKS5 代理 =====

	t.Run("SOCKS5ProxyConnect", func(t *testing.T) {
		proxyHost, proxyPort := testAddr("socks5-proxy", 1080, "localhost", 11080)
		cfg := &ConnConfig{
			ID:            "integ-socks5",
			Host:          "redis-nopass",
			Port:          6379,
			UseProxy:      true,
			ProxyType:     "socks5",
			ProxyHost:     proxyHost,
			ProxyPort:     proxyPort,
			ProxyUsername: "testuser",
			ProxyPassword: "testpass",
			ConnTimeout:   10 * time.Second,
			ExecTimeout:   10 * time.Second,
		}
		conn, err := Connect(cfg)
		require.NoError(t, err)
		defer Disconnect(cfg.ID)

		assert.NotNil(t, conn)
		assert.NoError(t, conn.Ping())
	})

	// ===== 通用操作 =====

	t.Run("ConnectDisconnect", func(t *testing.T) {
		host, port := testAddr("redis-nopass", 6379, "localhost", 16380)
		cfg := &ConnConfig{
			ID:          "integ-disconnect",
			Host:        host,
			Port:        port,
			ConnTimeout: 10 * time.Second,
			ExecTimeout: 10 * time.Second,
		}
		_, err := Connect(cfg)
		require.NoError(t, err)

		_, ok := GetConn(cfg.ID)
		assert.True(t, ok)

		Disconnect(cfg.ID)

		_, ok = GetConn(cfg.ID)
		assert.False(t, ok)
	})

	t.Run("Ping", func(t *testing.T) {
		host, port := testAddr("redis-nopass", 6379, "localhost", 16380)
		cfg := &ConnConfig{
			ID:          "integ-ping",
			Host:        host,
			Port:        port,
			ConnTimeout: 10 * time.Second,
			ExecTimeout: 10 * time.Second,
		}
		conn, err := Connect(cfg)
		require.NoError(t, err)
		defer Disconnect(cfg.ID)

		err = conn.Ping()
		assert.NoError(t, err)
	})
}

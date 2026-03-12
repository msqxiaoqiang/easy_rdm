//go:build integration

package tests

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// RedisContainer 封装测试用 Redis 容器
type RedisContainer struct {
	Container testcontainers.Container
	Host      string
	Port      string
}

// StartRedis 启动 Redis 容器，返回地址信息
func StartRedis(t *testing.T) *RedisContainer {
	t.Helper()
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "redis:7.2-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
		Cmd:          []string{"redis-server", "--save", "", "--appendonly", "no"},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start redis container: %v", err)
	}

	host, _ := container.Host(ctx)
	mappedPort, _ := container.MappedPort(ctx, "6379")

	t.Cleanup(func() {
		container.Terminate(ctx)
	})

	return &RedisContainer{
		Container: container,
		Host:      host,
		Port:      mappedPort.Port(),
	}
}

// Addr 返回 host:port 格式地址
func (r *RedisContainer) Addr() string {
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
}

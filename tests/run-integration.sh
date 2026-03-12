#!/bin/bash
# 集成测试一键脚本
# 生成证书 -> 启动 Docker Compose -> 等待服务就绪 -> 运行测试 -> 清理
#
# 用法:
#   bash run-integration.sh          # Host 模式（macOS 上 sentinel/cluster 会 skip）
#   bash run-integration.sh --docker # Docker 模式（测试在 Docker 网络内运行，全部可通过）

set -e

cd "$(dirname "$0")"
PROJECT_ROOT="$(cd .. && pwd)"

DOCKER_MODE=false
if [ "$1" = "--docker" ]; then
  DOCKER_MODE=true
fi

# 1. 生成 TLS 证书和 SSH 密钥（如不存在）
if [ ! -f certs/ca.crt ]; then
  echo ">>> 生成 TLS 证书和 SSH 密钥..."
  bash certs/gen-certs.sh
fi

# 2. 启动服务
echo ">>> 启动 Docker Compose 服务..."
docker compose -f docker-compose.test.yml up -d --build

# 3. 等待服务就绪
echo ">>> 等待服务启动..."
for port in 16379 16380 2222 16381 26379 7101 11080; do
  echo -n "  等待端口 $port..."
  timeout=60
  elapsed=0
  until nc -z localhost $port 2>/dev/null; do
    sleep 1
    elapsed=$((elapsed + 1))
    if [ $elapsed -ge $timeout ]; then
      echo " 超时！"
      echo "错误：端口 $port 在 ${timeout}s 内未就绪"
      docker compose -f docker-compose.test.yml logs
      docker compose -f docker-compose.test.yml down -v
      exit 1
    fi
  done
  echo " 就绪"
done

# 额外等待哨兵形成 quorum 和集群初始化
echo "  等待哨兵 quorum 和集群初始化 (5s)..."
sleep 5

echo ">>> 所有服务已就绪"

# 4. 运行测试
TEST_EXIT=0

if [ "$DOCKER_MODE" = true ]; then
  echo ">>> 运行集成测试（Docker 网络模式）..."

  # 在宿主机交叉编译 linux 测试二进制
  echo "  编译测试二进制..."
  cd "${PROJECT_ROOT}/backend"
  CGO_ENABLED=0 GOOS=linux GOARCH=$(docker info --format '{{.Architecture}}' 2>/dev/null | sed 's/x86_64/amd64/;s/aarch64/arm64/') \
    go test -tags integration -c -o /tmp/integration.test ./app/services/
  cd "${PROJECT_ROOT}/tests"

  # 获取 docker-compose 默认网络名（也连接 cluster-net 以测试集群）
  NETWORK_NAME=$(docker network ls --filter "name=tests_default" --format '{{.Name}}' | head -1)
  CLUSTER_NET=$(docker network ls --filter "name=tests_cluster-net" --format '{{.Name}}' | head -1)
  if [ -z "$NETWORK_NAME" ]; then
    NETWORK_NAME="tests_default"
  fi

  # 用轻量 alpine 容器在 Docker 网络内运行编译好的测试
  docker run --rm \
    --network "$NETWORK_NAME" \
    $([ -n "$CLUSTER_NET" ] && echo "--network $CLUSTER_NET") \
    -v /tmp/integration.test:/test:ro \
    -v "${PROJECT_ROOT}/tests:/testdata:ro" \
    -e REDIS_TEST_NETWORK=docker \
    redis:7.2-alpine \
    /test -test.v -test.count=1 -test.timeout=120s -test.run TestIntegration || TEST_EXIT=$?
else
  echo ">>> 运行集成测试（Host 模式）..."
  cd ../backend
  go test -tags integration -v -count=1 -timeout 120s ./app/services/ -run TestIntegration || TEST_EXIT=$?
  cd ../tests
fi

# 5. 清理
echo ">>> 清理 Docker Compose 服务..."
docker compose -f docker-compose.test.yml down -v

if [ $TEST_EXIT -eq 0 ]; then
  echo ">>> 所有测试通过！"
else
  echo ">>> 测试失败（退出码: $TEST_EXIT）"
fi

exit $TEST_EXIT

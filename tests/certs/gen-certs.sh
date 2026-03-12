#!/bin/bash
# 生成 TLS 自签名证书 + SSH 测试密钥
# 用于 Docker Compose 集成测试环境

set -e

CERT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$CERT_DIR"

echo "=== 生成 CA 证书 ==="
openssl req -x509 -new -nodes \
  -keyout ca.key -out ca.crt \
  -days 3650 -subj "/CN=Test CA"

echo "=== 生成 Redis 服务端证书 ==="
openssl req -new -nodes \
  -keyout redis.key -out redis.csr \
  -subj "/CN=redis-tls"

# SAN 配置：支持 Docker 服务名和 localhost
cat > server-ext.cnf <<EOF
[v3_req]
subjectAltName = DNS:redis-tls, DNS:localhost, IP:127.0.0.1
EOF

openssl x509 -req -in redis.csr \
  -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out redis.crt -days 3650 \
  -extfile server-ext.cnf -extensions v3_req

echo "=== 生成客户端证书（可选） ==="
openssl req -new -nodes \
  -keyout client.key -out client.csr \
  -subj "/CN=Redis Test Client"

openssl x509 -req -in client.csr \
  -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out client.crt -days 3650

echo "=== 生成 SSH 测试密钥 ==="
ssh-keygen -t ed25519 -f ssh_test_key -N "" -C "test@integration"

# 清理中间文件
rm -f *.csr *.srl server-ext.cnf

echo "=== 完成 ==="
echo "生成的文件："
ls -la *.crt *.key ssh_test_key*

FROM redis:7.2-alpine

RUN apk add --no-cache openssh-server && \
    ssh-keygen -A && \
    adduser -D -s /bin/sh testuser && \
    echo "testuser:testpass" | chpasswd && \
    mkdir -p /home/testuser/.ssh && \
    chmod 700 /home/testuser/.ssh && \
    chown testuser:testuser /home/testuser/.ssh

# 写入完整 sshd 配置（替换默认配置，避免 sed 不匹配的问题）
RUN printf '%s\n' \
    "Port 2222" \
    "HostKey /etc/ssh/ssh_host_rsa_key" \
    "HostKey /etc/ssh/ssh_host_ecdsa_key" \
    "HostKey /etc/ssh/ssh_host_ed25519_key" \
    "PasswordAuthentication yes" \
    "PubkeyAuthentication yes" \
    "PermitEmptyPasswords no" \
    "StrictModes no" \
    "AuthorizedKeysFile .ssh/authorized_keys" \
    "Subsystem sftp /usr/lib/ssh/sftp-server" \
    > /etc/ssh/sshd_config

EXPOSE 2222
ENTRYPOINT ["/usr/sbin/sshd"]
CMD ["-D", "-e"]

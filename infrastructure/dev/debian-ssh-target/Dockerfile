FROM docker.io/library/debian:12-slim

RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends openssh-server netcat-openbsd && \
    rm -rf /var/lib/apt/lists/* && \
    mkdir -p /run/sshd && \
    echo 'root:root' | chpasswd && \
    ssh-keygen -A && \
    printf '%s\n' \
      'Port 22' \
      'Protocol 2' \
      'PidFile /run/sshd.pid' \
      'HostKey /etc/ssh/ssh_host_rsa_key' \
      'HostKey /etc/ssh/ssh_host_ecdsa_key' \
      'HostKey /etc/ssh/ssh_host_ed25519_key' \
      'PermitRootLogin yes' \
      'PasswordAuthentication yes' \
      'PubkeyAuthentication no' \
      'ChallengeResponseAuthentication no' \
      'KbdInteractiveAuthentication no' \
      'UsePAM yes' \
      'PermitTTY yes' \
      'AllowTcpForwarding no' \
      'X11Forwarding no' \
      'PrintMotd no' \
      'Subsystem sftp internal-sftp' \
      > /etc/ssh/sshd_config

EXPOSE 22

CMD ["/bin/sh", "-lc", "mkdir -p /run/sshd && exec /usr/sbin/sshd -D -e"]

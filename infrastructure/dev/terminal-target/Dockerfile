FROM alpine:3.22

RUN apk add --no-cache bash openssh shadow netcat-openbsd && \
    adduser -D -s /bin/sh acceptance && \
    echo 'acceptance:acceptance' | chpasswd && \
    ssh-keygen -A

COPY infrastructure/dev/terminal-target/sshd_config /etc/ssh/sshd_config
COPY infrastructure/dev/terminal-target/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 2224

ENTRYPOINT ["/entrypoint.sh"]

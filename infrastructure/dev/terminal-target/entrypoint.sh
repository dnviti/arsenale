#!/bin/sh
set -e

mkdir -p /run/sshd
chmod 755 /run/sshd

exec /usr/sbin/sshd -D -e
